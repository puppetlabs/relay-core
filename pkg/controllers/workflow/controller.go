package workflow

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/puppetlabs/horsehead/v2/instrumentation/metrics"
	"github.com/puppetlabs/horsehead/v2/storage"
	"github.com/puppetlabs/nebula-sdk/pkg/workflow/spec/evaluate"
	"github.com/puppetlabs/nebula-sdk/pkg/workflow/spec/parse"
	"github.com/puppetlabs/nebula-sdk/pkg/workflow/spec/resolve"
	tekv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tekclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	teklisters "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
	"knative.dev/pkg/apis"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"

	nebulav1 "github.com/puppetlabs/nebula-tasks/pkg/apis/nebula.puppet.com/v1"
	"github.com/puppetlabs/nebula-tasks/pkg/config"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	clientset "github.com/puppetlabs/nebula-tasks/pkg/generated/clientset/versioned"
	neblisters "github.com/puppetlabs/nebula-tasks/pkg/generated/listers/nebula.puppet.com/v1"
	"github.com/puppetlabs/nebula-tasks/pkg/secrets/vault"
	stconfigmap "github.com/puppetlabs/nebula-tasks/pkg/state/configmap"
	"github.com/puppetlabs/nebula-tasks/pkg/util"
)

var controllerKind = nebulav1.SchemeGroupVersion.WithKind("WorkflowRun")

const (
	nebulaGroupNamePrefix = "nebula.puppet.com/"
	pipelineRunAnnotation = nebulaGroupNamePrefix + "pipelinerun"
	workflowRunAnnotation = nebulaGroupNamePrefix + "workflowrun"
	workflowRunLabel      = nebulaGroupNamePrefix + "workflow-run-id"
	workflowLabel         = nebulaGroupNamePrefix + "workflow-id"
)

type WorkflowRunStatus string

const (
	WorkflowRunStatusPending    WorkflowRunStatus = "pending"
	WorkflowRunStatusInProgress WorkflowRunStatus = "in-progress"
	WorkflowRunStatusSuccess    WorkflowRunStatus = "success"
	WorkflowRunStatusFailure    WorkflowRunStatus = "failure"
	WorkflowRunStatusCancelled  WorkflowRunStatus = "cancelled"
	WorkflowRunStatusSkipped    WorkflowRunStatus = "skipped"
	WorkflowRunStatusTimedOut   WorkflowRunStatus = "timed-out"
)

type WorkflowConditionType string

const (
	WorkflowConditionTypeApproval WorkflowConditionType = "approval"
)

const (
	// default name for the workflow metadata api pod and service
	metadataServiceName = "metadata-api"

	// name for the image pull secret used by the metadata API, if needed
	metadataImagePullSecretName = "metadata-api-docker-registry"

	// PipelineRun annotation indicating the log upload location
	logUploadAnnotationPrefix = "nebula.puppet.com/log-archive-"
)

const (
	InterpreterDirective = "#!"
	InterpreterDefault   = InterpreterDirective + "/bin/sh"

	NebulaMountPath      = "/nebula"
	NebulaEntrypointFile = "entrypoint.sh"
	NebulaSpecFile       = "spec.json"
)

const (
	ServiceAccountIdentifierCustomer = "customer"
	ServiceAccountIdentifierSystem   = "system"
	ServiceAccountIdentifierMetadata = "metadata"
)

const (
	WorkflowRunStateCancel = "cancel"
)

// TODO This needs to be exposed by Tekton in some manner
const (
	// ReasonTimedOut indicates that the PipelineRun has taken longer than its configured
	// timeout
	ReasonTimedOut = "PipelineRunTimeout"

	// ReasonConditionCheckFailed indicates that the reason for the failure status is that the
	// condition check associated to the pipeline task evaluated to false
	ReasonConditionCheckFailed = "ConditionCheckFailed"
)

const (
	conditionScript = `#!/bin/bash
JQ="${JQ:-jq}"

STATE_URL_PATH="${STATE_URL_PATH:-state}"
STATE_KEY_NAME="${STATE_KEY_NAME:-state}"
VALUE_KEY_NAME="${VALUE_KEY_NAME:-value}"
CONDITION="${CONDITION:-condition}"
POLLING_INTERVAL="${POLLING_INTERVAL:-5s}"
POLLING_ITERATIONS="${POLLING_ITERATIONS:-1080}"

for i in $(seq ${POLLING_ITERATIONS}); do
  STATE=$(curl "$METADATA_API_URL/${STATE_URL_PATH}/${STATE_KEY_NAME}")
  VALUE=$(echo $STATE | $JQ --arg value "$VALUE_KEY_NAME" -r '.[$value]')
  CONDITION_VALUE=$(echo $VALUE | $JQ --arg condition "$CONDITION" -r '.[$condition]')
  if [ -n "${CONDITION_VALUE}" ]; then
    if [ "$CONDITION_VALUE" = "true" ]; then
      exit 0
    fi
    if [ "$CONDITION_VALUE" = "false" ]; then
      exit 1
    fi
  fi
  sleep ${POLLING_INTERVAL}
done

exit 1
`
)

type podAndTaskName struct {
	PodName  string
	TaskName string
}

type StepTask struct {
	dependsOn  []string
	conditions []string
}

type StepTasks map[string]StepTask

// Controller watches for nebulav1.WorkflowRun resource changes.
// If a WorkflowRun resource is created, the controller will create a service acccount + rbac
// for the namespace, then inform vault that that service account is allowed to access
// readonly secrets under a preconfigured path related to a nebula workflow. It will then
// spin up a pod running an instance of nebula-metadata-api that knows how to
// ask kubernetes for the service account token, that it will use to proxy secrets
// between the task pods and the vault server.
type Controller struct {
	kubeclient    kubernetes.Interface
	nebclient     clientset.Interface
	tekclient     tekclientset.Interface
	secretsclient SecretAuthAccessManager
	storageclient storage.BlobStore

	wfrLister       neblisters.WorkflowRunLister
	wfrListerSynced cache.InformerSynced
	plrLister       teklisters.PipelineRunLister
	plrListerSynced cache.InformerSynced
	wfrworker       *worker
	plrworker       *worker

	namespace string
	cfg       *config.WorkflowControllerConfig
	manager   *DependencyManager
	metrics   *controllerObservations
}

// Run starts all required informers and spawns two worker goroutines
// that will pull resource objects off the workqueue. This method blocks
// until stopCh is closed or an earlier bootstrap call results in an error.
func (c *Controller) Run(numWorkers int, stopCh chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.plrworker.shutdown()
	defer c.wfrworker.shutdown()

	if !cache.WaitForCacheSync(stopCh, c.plrListerSynced, c.wfrListerSynced) {
		return fmt.Errorf("failed to wait for informer cache to sync")
	}

	c.plrworker.run(numWorkers, stopCh)
	c.wfrworker.run(numWorkers, stopCh)

	<-stopCh

	return nil
}

func (c *Controller) waitForEndpoint(service *corev1.Service) error {
	var (
		conditionMet bool
		timeout      = int64(30)
	)

	endpoints, err := c.kubeclient.CoreV1().Endpoints(service.GetNamespace()).Get(service.GetName(), metav1.GetOptions{})
	if err != nil {
		return err
	}

	if endpoints.Subsets != nil && len(endpoints.Subsets) > 0 {
		for _, subset := range endpoints.Subsets {
			if subset.Addresses != nil && len(subset.Addresses) > 0 {
				return nil
			}
		}
	}

	listOptions := metav1.SingleObject(endpoints.ObjectMeta)
	listOptions.TimeoutSeconds = &timeout

	watcher, err := c.kubeclient.CoreV1().Endpoints(endpoints.GetNamespace()).Watch(listOptions)
	if err != nil {
		return err
	}

eventLoop:
	for event := range watcher.ResultChan() {
		switch event.Type {
		case watch.Modified:
			endpoints := event.Object.(*corev1.Endpoints)

			if endpoints.Subsets != nil && len(endpoints.Subsets) > 0 {
				for _, subset := range endpoints.Subsets {
					if subset.Addresses != nil && len(subset.Addresses) > 0 {
						watcher.Stop()
						conditionMet = true

						break eventLoop
					}
				}
			}
		}
	}

	if !conditionMet {
		return fmt.Errorf("timeout occurred while waiting for service %s to be ready", service.GetName())
	}

	return nil
}

func (c *Controller) waitForSuccessfulServiceResponse(service *corev1.Service) error {
	if !c.cfg.MetadataServiceCheckEnabled {
		return nil
	}

	var (
		u        = fmt.Sprintf("http://%s.%s.svc.cluster.local/healthz", service.GetName(), service.GetNamespace())
		interval = time.Millisecond * 750
		timeout  = time.Second * 10
	)

	return wait.PollImmediate(interval, timeout, func() (bool, error) {
		client := http.Client{
			Timeout: timeout,
		}
		resp, err := client.Get(u)
		if err != nil {
			klog.Infof("got an error when probing the service %s - %s", service.GetName(), err)
			return false, nil
		}

		if resp.StatusCode != http.StatusOK {
			klog.Infof("got an invalid status code when probing service %s %d", service.GetName(), resp.StatusCode)
			return false, nil
		}

		return true, nil
	})
}

func (c *Controller) processPipelineRun(ctx context.Context, key string) error {
	klog.Infof("syncing PipelineRun %s", key)
	defer klog.Infof("done syncing PipelineRun %s", key)

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	plr, err := c.tekclient.TektonV1alpha1().PipelineRuns(namespace).Get(name, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}

	logAnnotations := make(map[string]string, 0)

	if areWeDoneYet(plr) {
		// Upload the logs that are not defined on the PipelineRun record...
		err := c.metrics.trackDurationWithOutcome(metricWorkflowRunLogUploadDuration, func() error {
			logAnnotations, err = c.uploadLogs(ctx, plr)

			return err
		})
		if nil != err {
			klog.Warning(err)
		}
	}

	labelMap := map[string]string{
		pipelineRunAnnotation: plr.Name,
	}

	selector := labels.SelectorFromValidatedSet(labelMap)
	workflowList, err := c.wfrLister.WorkflowRuns(namespace).List(selector)
	if err != nil {
		return err
	}

	for _, workflow := range workflowList {
		if areWeDoneYet(plr) {
			klog.Infof("revoking workflow run secret access %s", workflow.GetName())
			if err := c.secretsclient.RevokeScopedAccess(ctx, namespace); err != nil {
				return err
			}
		}

		status, _ := c.updateWorkflowRunStatus(plr, workflow)

		wfrCopy := workflow.DeepCopy()
		wfrCopy.Status = *status

		for name, value := range logAnnotations {
			metav1.SetMetaDataAnnotation(&wfrCopy.ObjectMeta, name, value)
		}

		_, err = c.nebclient.NebulaV1().WorkflowRuns(namespace).Update(wfrCopy)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Controller) uploadLogs(ctx context.Context, plr *tekv1alpha1.PipelineRun) (map[string]string, error) {
	logAnnotations := make(map[string]string, 0)

	for _, pt := range extractPodAndTaskNamesFromPipelineRun(plr) {
		annotation := nebulaGroupNamePrefix + util.Slug(logUploadAnnotationPrefix+pt.TaskName)
		if _, ok := plr.Annotations[annotation]; ok {
			continue
		}
		containerName := "step-" + pt.TaskName
		logName, err := c.uploadLog(ctx, plr.Namespace, pt.PodName, containerName)
		if nil != err {
			klog.Warningf("Failed to upload log for pod=%s/%s container=%s %+v",
				plr.Namespace,
				pt.PodName,
				containerName,
				err)
			continue
		}

		logAnnotations[annotation] = logName
	}

	return logAnnotations, nil
}

func (c *Controller) uploadLog(ctx context.Context, namespace string, podName string, containerName string) (string, error) {
	key := fmt.Sprintf("%s/%s/%s", namespace, podName, containerName)

	opts := &corev1.PodLogOptions{
		Container: containerName,
	}
	rc, err := c.kubeclient.CoreV1().Pods(namespace).GetLogs(podName, opts).Stream()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	storageOpts := storage.PutOptions{
		ContentType: "application/octet-stream",
	}

	err = c.storageclient.Put(ctx, key, func(w io.Writer) error {
		_, err := io.Copy(w, rc)

		return err
	}, storageOpts)
	if err != nil {
		return "", err
	}

	return key, nil
}

func (c *Controller) processWorkflowRun(ctx context.Context, key string) error {
	klog.Infof("syncing WorkflowRun %s", key)
	defer klog.Infof("done syncing WorkflowRun %s", key)

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	wr, err := c.nebclient.NebulaV1().WorkflowRuns(namespace).Get(name, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		klog.Infof("%s %s has been deleted", wr.Kind, key)

		return nil
	}
	if err != nil {
		return err
	}

	// FIXME Debugging purposes only...
	klog.Info(wr)
	for index, value := range wr.Spec.Workflow.Steps {
		klog.Info(index, value)
	}

	err = c.handleWorkflowRun(ctx, wr)

	return err
}

func (c *Controller) writeWorkflowState(wr *nebulav1.WorkflowRun, taskHash [sha1.Size]byte, state nebulav1.WorkflowState) errors.Error {
	stm := stconfigmap.New(c.kubeclient, wr.GetNamespace())
	ctx := context.Background()

	data, err := json.Marshal(state)
	if err != nil {
		return errors.NewServerJSONEncodingError().WithCause(err)
	}

	buf := &bytes.Buffer{}
	buf.Write(data)
	err = stm.Set(ctx, taskHash, "state", buf)
	if err != nil {
		return errors.NewWorkflowExecutionError().WithCause(err)
	}

	return nil
}

func (c *Controller) handleWorkflowRun(ctx context.Context, wr *nebulav1.WorkflowRun) error {
	err := c.initializeWorkflowRun(ctx, wr)
	if err != nil {
		return err
	}

	if wr.ObjectMeta.DeletionTimestamp.IsZero() {
		cancelled := isCancelled(wr)

		if cancelled {
			if wr.Status.Status != string(WorkflowRunStatusCancelled) {
				wfrCopy := wr.DeepCopy()
				wfrCopy.Status.Status = string(WorkflowRunStatusCancelled)

				_, err = c.nebclient.NebulaV1().WorkflowRuns(wr.GetNamespace()).Update(wfrCopy)
				if err != nil {
					return err
				}
			}
		}

		if annotation, ok := wr.GetAnnotations()[pipelineRunAnnotation]; !ok && !cancelled {
			plr, err := c.createPipelineRun(wr)
			if err != nil {
				return err
			}

			pipelineId := wr.Spec.Name
			if wr.Labels == nil {
				wr.Labels = make(map[string]string, 0)
			}
			wr.Labels[pipelineRunAnnotation] = pipelineId

			metav1.SetMetaDataAnnotation(&wr.ObjectMeta, pipelineRunAnnotation, plr.Name)

			wr, err = c.nebclient.NebulaV1().WorkflowRuns(wr.GetNamespace()).Update(wr)
			if err != nil {
				return err
			}
		} else if ok && cancelled {
			plr, err := c.tekclient.TektonV1alpha1().PipelineRuns(wr.GetNamespace()).Get(annotation, metav1.GetOptions{})
			if k8serrors.IsNotFound(err) {
				return nil
			}
			if err != nil {
				return err
			}

			plr.Spec.Status = tekv1alpha1.PipelineRunSpecStatusCancelled
			_, err = c.tekclient.TektonV1alpha1().PipelineRuns(wr.GetNamespace()).Update(plr)
			if err != nil {
				return err
			}
		}
	} else {
		if containsString(wr.ObjectMeta.Finalizers, workflowRunAnnotation) {
			wr.ObjectMeta.Finalizers = removeString(wr.ObjectMeta.Finalizers, workflowRunAnnotation)

			wr, err = c.nebclient.NebulaV1().WorkflowRuns(wr.GetNamespace()).Update(wr)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *Controller) initializeWorkflowRun(ctx context.Context, wr *nebulav1.WorkflowRun) error {
	for name, value := range wr.State.Steps {
		hash := sha1.Sum([]byte(name))
		err := c.writeWorkflowState(wr, hash, value)
		if err != nil {
			return err
		}
	}

	// If we haven't set the state of the run yet, then we need to ensure all the secret access
	// and rbac is setup.
	if wr.Status.Status == "" {
		klog.Infof("unreconciled %s %s", wr.Kind, wr.GetName())

		err := c.metrics.trackDurationWithOutcome(metricWorkflowRunStartUpDuration, func() error {
			service, err := c.createAccessResources(ctx, wr)
			if err != nil {
				return err
			}

			if err := c.initializePipeline(wr, service); err != nil {
				return err
			}

			return c.ensureAccessResourcesExist(ctx, wr, service)
		})

		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Controller) createAccessResources(ctx context.Context, wr *nebulav1.WorkflowRun) (*corev1.Service, error) {
	var (
		ips      *corev1.Secret
		saccount *corev1.ServiceAccount
		err      error
	)

	namespace := wr.GetNamespace()

	if c.cfg.MetadataServiceImagePullSecret != "" {
		klog.Infof("copying secret for metadata service image %s", wr.GetName())
		ips, err = copyImagePullSecret(c.namespace, c.kubeclient, wr, c.cfg.MetadataServiceImagePullSecret)
		if err != nil {
			return nil, err
		}
	}

	_, err = createServiceAccount(c.kubeclient, wr, ServiceAccountIdentifierCustomer, nil)
	if err != nil {
		return nil, err
	}

	_, err = createServiceAccount(c.kubeclient, wr, ServiceAccountIdentifierSystem, ips)
	if err != nil {
		return nil, err
	}

	saccount, err = createServiceAccount(c.kubeclient, wr, ServiceAccountIdentifierMetadata, ips)
	if err != nil {
		return nil, err
	}

	klog.Infof("granting workflow run access to scoped secrets %s", wr.GetName())
	grant, err := c.secretsclient.GrantScopedAccess(ctx, wr.Spec.Workflow.Name, namespace, saccount.GetName())
	if err != nil {
		return nil, err
	}

	_, _, err = createRBAC(c.kubeclient, wr, saccount)
	if err != nil {
		return nil, err
	}

	// It is possible that the metadata service and this controller talk to
	// different Vault endpoints: each might be talking to a Vault agent (for
	// caching or additional security) instead of directly to the Vault server.
	podVaultAddr := c.cfg.MetadataServiceVaultAddr
	if podVaultAddr == "" {
		podVaultAddr = grant.BackendAddr
	}

	podVaultAuthMountPath := c.cfg.MetadataServiceVaultAuthMountPath
	if podVaultAuthMountPath == "" {
		podVaultAuthMountPath = "auth/kubernetes"
	}

	_, err = createMetadataAPIPod(
		c.kubeclient,
		c.cfg.MetadataServiceImage,
		saccount,
		wr,
		podVaultAddr,
		podVaultAuthMountPath,
		grant.ScopedPath,
	)
	if err != nil {
		return nil, err
	}

	service, err := createMetadataAPIService(c.kubeclient, wr)
	if err != nil {
		return nil, err
	}

	return service, nil
}

func (c *Controller) ensureAccessResourcesExist(ctx context.Context, wr *nebulav1.WorkflowRun, service *corev1.Service) error {
	return c.metrics.trackDurationWithOutcome(metricWorkflowRunWaitForMetadataAPIServiceDuration, func() error {
		klog.Infof("waiting for metadata service to become ready %s", wr.Spec.Workflow.Name)

		// This waits for a Modified watch event on a service's Endpoint object.
		// When this event is received, it will check it's addresses to see if there's
		// pods that are ready to be served.
		if err := c.waitForEndpoint(service); err != nil {
			return err
		}

		// Because of a possible race condition bug in the kernel or kubelet network stack, there's a very
		// tiny window of time where packets will get dropped if you try to make requests to the ports
		// that are supposed to be forwarded to underlying pods. This unfortunately happens quite frequently
		// since we exec task pods from Tekton very quickly. This function will make GET requests in a loop
		// to the readiness endpoint of the pod (via the service dns) to make sure it actually gets a 200
		// response before setting the status object on SecretAuth resources.
		if err := c.waitForSuccessfulServiceResponse(service); err != nil {
			return err
		}

		klog.Infof("metadata service is ready %s", wr.GetName())

		return nil
	})
}

func (c *Controller) enqueueWorkflowRun(obj interface{}) {
	wf := obj.(*nebulav1.WorkflowRun)

	key, err := cache.MetaNamespaceKeyFunc(wf)
	if err != nil {
		utilruntime.HandleError(err)

		return
	}

	c.wfrworker.add(key)
}

func (c *Controller) enqueuePipelineRun(obj interface{}) {
	plr := obj.(*tekv1alpha1.PipelineRun)

	key, err := cache.MetaNamespaceKeyFunc(plr)
	if err != nil {
		utilruntime.HandleError(err)

		return
	}

	c.plrworker.add(key)
}

func (c *Controller) createPipelineRun(wr *nebulav1.WorkflowRun) (*tekv1alpha1.PipelineRun, error) {
	klog.Infof("creating PipelineRun for WorkflowRun %s", wr.GetName())
	defer klog.Infof("done creating PipelineRun for WorkflowRun %s", wr.GetName())

	namespace := wr.GetNamespace()

	plr, err := c.tekclient.TektonV1alpha1().PipelineRuns(namespace).Get(wr.GetName(), metav1.GetOptions{})
	if plr != nil && plr != (&tekv1alpha1.PipelineRun{}) && plr.Name != "" {
		return plr, nil
	}

	runID := wr.Spec.Name

	serviceAccounts := make([]tekv1alpha1.PipelineRunSpecServiceAccountName, 0)
	for _, step := range wr.Spec.Workflow.Steps {
		if step == nil {
			continue
		}
		taskHash := sha1.Sum([]byte(step.Name))
		taskId := hex.EncodeToString(taskHash[:])
		psa := tekv1alpha1.PipelineRunSpecServiceAccountName{
			TaskName:           taskId,
			ServiceAccountName: getName(wr, ServiceAccountIdentifierCustomer),
		}
		serviceAccounts = append(serviceAccounts, psa)
	}

	pipelineRun := &tekv1alpha1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:            runID,
			Namespace:       namespace,
			Labels:          getLabels(wr, nil),
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(wr, controllerKind)},
		},
		Spec: tekv1alpha1.PipelineRunSpec{
			ServiceAccountName:  getName(wr, ServiceAccountIdentifierSystem),
			ServiceAccountNames: serviceAccounts,
			PipelineRef: &tekv1alpha1.PipelineRef{
				Name: runID,
			},
			PodTemplate: &tekv1alpha1.PodTemplate{
				NodeSelector: map[string]string{
					"nebula.puppet.com/scheduling.customer-ready": "true",
				},
				Tolerations: []corev1.Toleration{
					{
						Key:    "nebula.puppet.com/scheduling.customer-workload",
						Value:  "true",
						Effect: corev1.TaintEffectNoSchedule,
					},
				},
			},
		},
	}

	createdPipelineRun, err := c.tekclient.TektonV1alpha1().PipelineRuns(namespace).Create(pipelineRun)
	if err != nil {
		return nil, err
	}

	return createdPipelineRun, nil
}

func (c *Controller) updateWorkflowRunStatus(plr *tekv1alpha1.PipelineRun, wr *nebulav1.WorkflowRun) (*nebulav1.WorkflowRunStatus, error) {
	workflowRunSteps := make(map[string]nebulav1.WorkflowRunStatusSummary)
	workflowRunConditions := make(map[string]nebulav1.WorkflowRunStatusSummary)

	status := string(mapStatus(plr.Status.Status))

	// FIXME Not necessarily true (needs to differentiate between actual failures and cancellations)
	if isCancelled(wr) {
		status = string(WorkflowRunStatusCancelled)
	}

	for _, taskRun := range plr.Status.TaskRuns {
		for _, condition := range taskRun.ConditionChecks {
			if condition.Status == nil {
				continue
			}
			conditionStep := nebulav1.WorkflowRunStatusSummary{
				Status: string(mapStatus(condition.Status.Status)),
			}

			if condition.Status.StartTime != nil {
				conditionStep.StartTime = condition.Status.StartTime
			}
			if condition.Status.CompletionTime != nil {
				conditionStep.CompletionTime = condition.Status.CompletionTime
			}

			workflowRunConditions[condition.ConditionName] = conditionStep
		}

		if taskRun.Status == nil {
			continue
		}

		step := nebulav1.WorkflowRunStatusSummary{
			Status: string(mapStatus(taskRun.Status.Status)),
		}

		if taskRun.Status.StartTime != nil {
			step.StartTime = taskRun.Status.StartTime
		}
		if taskRun.Status.CompletionTime != nil {
			step.CompletionTime = taskRun.Status.CompletionTime
		}

		workflowRunSteps[taskRun.PipelineTaskName] = step
	}

	workflowRunStatus := &nebulav1.WorkflowRunStatus{
		Status:     status,
		Steps:      workflowRunSteps,
		Conditions: workflowRunConditions,
	}

	if plr.Status.StartTime != nil {
		workflowRunStatus.StartTime = plr.Status.StartTime
	}
	if plr.Status.CompletionTime != nil {
		workflowRunStatus.CompletionTime = plr.Status.CompletionTime
	}

	return workflowRunStatus, nil
}

func (c *Controller) initializePipeline(wr *nebulav1.WorkflowRun, service *corev1.Service) errors.Error {
	klog.Infof("initializing Pipeline %s", wr.GetName())
	defer klog.Infof("done initializing Pipeline %s", wr.GetName())

	if len(wr.Spec.Workflow.Steps) == 0 {
		return nil
	}

	pipeline, err := c.tekclient.TektonV1alpha1().Pipelines(wr.GetNamespace()).Get(wr.GetName(), metav1.GetOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		return errors.NewWorkflowExecutionError().WithCause(err)
	}

	if pipeline.Name == wr.GetName() {
		return nil
	}

	if _, err := c.createNetworkPolicies(wr.GetNamespace()); err != nil {
		return errors.NewWorkflowExecutionError().WithCause(err)
	}

	if _, err := c.createLimitRange(wr.GetNamespace()); err != nil {
		return errors.NewWorkflowExecutionError().WithCause(err)
	}

	tasks, err := c.createTasks(wr, service)
	if err != nil {
		return errors.NewWorkflowExecutionError().WithCause(err)
	}

	pipelineTasks, err := c.createPipelineTasks(tasks)
	if err != nil {
		return errors.NewWorkflowExecutionError().WithCause(err)
	}

	pipeline, err = c.createPipeline(wr.GetNamespace(), wr.GetName(), pipelineTasks)
	if err != nil {
		return errors.NewWorkflowExecutionError().WithCause(err)
	}

	return nil
}

func (c *Controller) createNetworkPolicies(namespace string) ([]*networkingv1.NetworkPolicy, errors.Error) {
	var pols []*networkingv1.NetworkPolicy

	pols = append(pols, &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "metadata-api-allow",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "nebula",
			},
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name":      "nebula",
					"app.kubernetes.io/component": "metadata-api",
				},
			},
			PolicyTypes: []networkingv1.PolicyType{"Ingress", "Egress"},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					From: []networkingv1.NetworkPolicyPeer{
						{
							// Match all pods in this namespace.
							PodSelector: &metav1.LabelSelector{},
						},
						{
							// Allow the workflow controller to check for this
							// service's status.
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"nebula.puppet.com/network-policy.tasks": "true",
								},
							},
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"app.kubernetes.io/name":      "nebula-system",
									"app.kubernetes.io/component": "tasks",
								},
							},
						},
					},
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Protocol: func(p corev1.Protocol) *corev1.Protocol { return &p }(corev1.ProtocolTCP),
							Port:     func(i intstr.IntOrString) *intstr.IntOrString { return &i }(intstr.FromInt(7000)),
						},
					},
				},
			},
			Egress: []networkingv1.NetworkPolicyEgressRule{
				{
					To: []networkingv1.NetworkPolicyPeer{
						{
							// Only allow outbound to the tasks namespace.
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"nebula.puppet.com/network-policy.tasks": "true",
								},
							},
						},
					},
				},
			},
		},
	})

	// We need to let the metadata API talk to the Kubernetes master in private
	// clusters, which use RFC 1918 space. The master is not addressable using a
	// label selector, sadly.
	//
	// Per https://github.com/kubernetes/kubernetes/issues/49978, the additive
	// nature of network policy peers should let us include 0.0.0.0/0, exclude
	// 10.0.0.0/8, and then include 10.X.Y.Z/32 (since policies are supposed to
	// be additive). However, as of 2019-12-12, this does not appear to work
	// with GKE's Project Calico networking implementation, so we'll instead
	// filter the master out of our deny list.
	initialDeny := []string{
		"0.0.0.0/8",       // "This host on this network"
		"10.0.0.0/8",      // Private-Use
		"100.64.0.0/10",   // Shared Address Space
		"169.254.0.0/16",  // Link Local
		"172.16.0.0/12",   // Private-Use
		"192.0.0.0/24",    // IETF Protocol Assignments
		"192.0.2.0/24",    // Documentation (TEST-NET-1)
		"192.31.196.0/24", // AS112-v4
		"192.52.193.0/24", // AMT
		"192.168.0.0/16",  // Private-Use
		"192.175.48.0/24", // Direct Delegation AS112 Service
		"198.18.0.0/15",   // Benchmarking
		"198.51.100.0/24", // Documentation (TEST-NET-2)
		"203.0.113.0/24",  // Documentation (TEST-NET-3)
		"240.0.0.0/4",     // Reserved (multicast)
	}

	// Get the cluster master endpoints.
	master, err := c.kubeclient.CoreV1().Endpoints("default").Get("kubernetes", metav1.GetOptions{}) // kubernetes.default.svc
	if err != nil {
		return nil, errors.NewWorkflowExecutionError().WithCause(err)
	}

	var masterIPs []net.IP
	for _, subset := range master.Subsets {
		for _, addr := range subset.Addresses {
			ip := net.ParseIP(addr.IP)
			if ip != nil {
				masterIPs = append(masterIPs, ip)
			}
		}
	}

	var filteredDeny []string
	for _, cidr := range initialDeny {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			// This shouldn't happen, but it will get caught by the admission
			// controller anyway.
			filteredDeny = append(filteredDeny, cidr)
			continue
		}

		filtered := ipNetExclude(network, masterIPs)
		for _, filteredNetwork := range filtered {
			filteredDeny = append(filteredDeny, filteredNetwork.String())
		}
	}

	pols = append(pols, &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "nebula",
			},
		},
		Spec: networkingv1.NetworkPolicySpec{
			// Empty pod selector matches all pods.
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []networkingv1.PolicyType{"Ingress", "Egress"},
			// We omit ingress to deny inbound traffic. Nothing should be
			// connecting to task pods.
			Ingress: []networkingv1.NetworkPolicyIngressRule{},
			Egress: []networkingv1.NetworkPolicyEgressRule{
				{
					To: []networkingv1.NetworkPolicyPeer{
						{
							// Allow all external traffic except RFC 1918 space
							// and IANA special-purpose address registry.
							IPBlock: &networkingv1.IPBlock{
								CIDR:   "0.0.0.0/0",
								Except: filteredDeny,
							},
						},
						{
							// Allow access to the metadata API.
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"app.kubernetes.io/name":      "nebula",
									"app.kubernetes.io/component": "metadata-api",
								},
							},
						},
						{
							// Allow access to kube-dns.
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"nebula.puppet.com/network-policy.kube-system": "true",
								},
							},
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"k8s-app": "kube-dns",
								},
							},
						},
					},
				},
			},
		},
	})

	for i := range pols {
		pol, err := c.kubeclient.NetworkingV1().NetworkPolicies(namespace).Create(pols[i])
		if err != nil && !k8serrors.IsAlreadyExists(err) {
			return nil, errors.NewWorkflowExecutionError().WithCause(err)
		}

		pols[i] = pol
	}

	return pols, nil
}

func (c *Controller) createLimitRange(namespace string) (*corev1.LimitRange, errors.Error) {
	// Set some default (fairly generous) CPU and memory limits.

	lr := &corev1.LimitRange{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default",
			Namespace: namespace,
		},
		Spec: corev1.LimitRangeSpec{
			Limits: []corev1.LimitRangeItem{
				{
					Type: corev1.LimitTypeContainer,
					Default: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("750m"),
						corev1.ResourceMemory: resource.MustParse("2Gi"),
					},
					DefaultRequest: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("256Mi"),
					},
					Max: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("1"),
						corev1.ResourceMemory: resource.MustParse("3Gi"),
					},
				},
			},
		},
	}

	lr, err := c.kubeclient.CoreV1().LimitRanges(namespace).Create(lr)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return nil, errors.NewWorkflowExecutionError().WithCause(err)
	}

	return lr, nil
}

func (c *Controller) createTasks(wr *nebulav1.WorkflowRun, service *corev1.Service) (StepTasks, errors.Error) {
	stepTasks := make(StepTasks)

	metadataAPIURL := fmt.Sprintf("http://%s.%s.svc.cluster.local", service.GetName(), wr.GetNamespace())

	ownerReference := metav1.NewControllerRef(wr, controllerKind)

	for _, step := range wr.Spec.Workflow.Steps {
		if step == nil {
			continue
		}

		taskHash := sha1.Sum([]byte(step.Name))
		taskId := hex.EncodeToString(taskHash[:])

		err := c.createTaskConfigMap(taskId, wr.GetNamespace(), wr.Spec.Workflow.Parameters, wr.Spec.Parameters, step, ownerReference)
		if err != nil {
			return nil, errors.NewWorkflowExecutionError().WithCause(err)
		}

		_, err = c.createTaskFromStep(taskId, wr.GetNamespace(), metadataAPIURL, step)
		if err != nil {
			return nil, errors.NewWorkflowExecutionError().WithCause(err)
		}

		dependsOn := make([]string, 0)
		conditions := make([]string, 0)

		for _, dependency := range step.DependsOn {
			dependencyHash := sha1.Sum([]byte(dependency))
			dependencyId := hex.EncodeToString(dependencyHash[:])
			dependsOn = append(dependsOn, dependencyId)
		}

		for _, condition := range step.Conditions {
			if condition.Type == string(WorkflowConditionTypeApproval) {
				conditionHash := sha1.Sum([]byte(condition.Name))
				conditionId := hex.EncodeToString(conditionHash[:])
				err := c.createCondition(wr.GetNamespace(), conditionId, c.cfg.ApprovalTypeImage, metadataAPIURL, ownerReference)
				if err != nil {
					return nil, errors.NewWorkflowExecutionError().WithCause(err)
				}

				conditions = append(conditions, conditionId)
			}
		}

		stepTasks[taskId] = StepTask{
			dependsOn:  dependsOn,
			conditions: conditions,
		}
	}

	return stepTasks, nil
}

func (c *Controller) createPipelineTasks(stepTasks StepTasks) ([]tekv1alpha1.PipelineTask, errors.Error) {

	pipelineTasks := make([]tekv1alpha1.PipelineTask, 0)

	for taskId, stepTask := range stepTasks {
		dependencies, conditions, err := c.getTaskDependencies(stepTask)
		if err != nil {
			return nil, errors.NewWorkflowExecutionError().WithCause(err)
		}

		pipelineTask := tekv1alpha1.PipelineTask{
			Name: taskId,
			TaskRef: &tekv1alpha1.TaskRef{
				Name: taskId,
			},
			RunAfter:   dependencies,
			Conditions: conditions,
		}

		pipelineTasks = append(pipelineTasks, pipelineTask)
	}

	return pipelineTasks, nil
}

func (c *Controller) getTaskDependencies(stepTask StepTask) ([]string, []tekv1alpha1.PipelineTaskCondition, errors.Error) {
	dependencies := make([]string, 0)
	conditions := make([]tekv1alpha1.PipelineTaskCondition, 0)

	for _, dependsOn := range stepTask.dependsOn {
		dependencies = append(dependencies, dependsOn)
	}

	for _, condition := range stepTask.conditions {
		pipelineTaskCondition := tekv1alpha1.PipelineTaskCondition{
			ConditionRef: condition,
		}
		conditions = append(conditions, pipelineTaskCondition)
	}

	return dependencies, conditions, nil
}

func (c *Controller) createCondition(namespace string, conditionName string, image string, metadataAPIURL string, ownerReference *metav1.OwnerReference) errors.Error {

	evs := buildEnvironmentVariables(metadataAPIURL, conditionName)
	evs = append(evs, buildEnvironmentVariablesForCondition()...)

	condition := tekv1alpha1.Condition{
		ObjectMeta: metav1.ObjectMeta{
			Name:            conditionName,
			Namespace:       namespace,
			OwnerReferences: []metav1.OwnerReference{*ownerReference},
			Labels: map[string]string{
				"nebula.puppet.com/task.hash": conditionName,
			},
		},
		Spec: tekv1alpha1.ConditionSpec{
			Check: tekv1alpha1.Step{
				Container: corev1.Container{
					Image: "projectnebula/core",
					Name:  conditionName,
					Env:   evs,
				},
				Script: conditionScript,
			},
		},
	}

	_, err := c.tekclient.TektonV1alpha1().Conditions(namespace).Create(&condition)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return errors.NewWorkflowExecutionError().WithCause(err)
	}

	return nil
}

func (c *Controller) createTaskConfigMap(name string, namespace string, workflowParameters nebulav1.WorkflowParameters, workflowRunParameters nebulav1.WorkflowRunParameters, step *nebulav1.WorkflowStep, ownerReference *metav1.OwnerReference) errors.Error {
	configMapData, _ := getConfigMapData(workflowParameters, workflowRunParameters, step)
	_, err := c.createConfigMap(name, configMapData, namespace, ownerReference)
	if err != nil {
		return errors.NewWorkflowExecutionError().WithCause(err)
	}

	return nil
}

func (c *Controller) createTaskFromStep(name string, namespace string, metadataAPIURL string, step *nebulav1.WorkflowStep) (*tekv1alpha1.Task, errors.Error) {
	variantStep := step
	container, volumes := getTaskContainer(metadataAPIURL, name, variantStep)
	return c.createTask(name, namespace, container, volumes)
}

func (c *Controller) createTask(taskName string, namespace string, container *corev1.Container, volumes []corev1.Volume) (*tekv1alpha1.Task, errors.Error) {
	task := &tekv1alpha1.Task{
		ObjectMeta: metav1.ObjectMeta{
			Name:      taskName,
			Namespace: namespace,
			Labels: map[string]string{
				"nebula.puppet.com/task.hash": taskName,
			},
		},
		Spec: tekv1alpha1.TaskSpec{
			Steps: []tekv1alpha1.Step{
				{
					Container: *container,
				},
			},
			Volumes: volumes,
		},
	}

	task, err := c.tekclient.TektonV1alpha1().Tasks(namespace).Create(task)
	if k8serrors.IsAlreadyExists(err) {
		task, err = c.tekclient.TektonV1alpha1().Tasks(namespace).Get(taskName, metav1.GetOptions{})
	}
	if err != nil {
		return nil, errors.NewWorkflowExecutionError().WithCause(err)
	}

	return task, nil
}

func (c *Controller) createPipeline(namespace string, pipelineId string, pipelineTasks []tekv1alpha1.PipelineTask) (*tekv1alpha1.Pipeline, errors.Error) {
	pipelineName := util.Slug(pipelineId)

	pipeline := &tekv1alpha1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pipelineName,
			Namespace: namespace,
		},
		Spec: tekv1alpha1.PipelineSpec{
			Tasks: pipelineTasks,
		},
	}

	pipeline, err := c.tekclient.TektonV1alpha1().Pipelines(namespace).Create(pipeline)
	if k8serrors.IsAlreadyExists(err) {
		pipeline, err = c.tekclient.TektonV1alpha1().Pipelines(namespace).Get(pipelineName, metav1.GetOptions{})
	}
	if err != nil {
		return nil, errors.NewWorkflowExecutionError().WithCause(err)
	}

	return pipeline, nil
}

func (c *Controller) createConfigMap(name string, data map[string]string, namespace string, ownerReference *metav1.OwnerReference) (*corev1.ConfigMap, error) {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			OwnerReferences: []metav1.OwnerReference{*ownerReference},
		},
		Data: data,
	}

	configMap, err := c.kubeclient.CoreV1().ConfigMaps(namespace).Create(configMap)
	if k8serrors.IsAlreadyExists(err) {
		configMap, err = c.kubeclient.CoreV1().ConfigMaps(namespace).Get(name, metav1.GetOptions{})
	}
	if err != nil {
		return nil, errors.NewWorkflowExecutionError().WithCause(err)
	}

	return configMap, nil
}

func NewController(manager *DependencyManager, cfg *config.WorkflowControllerConfig, vc *vault.VaultAuth, bs storage.BlobStore, namespace string, mets *metrics.Metrics) *Controller {
	wfrInformer := manager.WorkflowRunInformer()
	plrInformer := manager.PipelineRunInformer()

	c := &Controller{
		kubeclient:      manager.KubeClient,
		nebclient:       manager.NebulaClient,
		tekclient:       manager.TektonClient,
		secretsclient:   vc,
		storageclient:   bs,
		wfrLister:       wfrInformer.Lister(),
		plrLister:       plrInformer.Lister(),
		wfrListerSynced: wfrInformer.Informer().HasSynced,
		plrListerSynced: plrInformer.Informer().HasSynced,
		namespace:       namespace,
		cfg:             cfg,
		manager:         manager,
		metrics:         newControllerObservations(mets),
	}

	c.wfrworker = newWorker("WorkflowRuns", (*c).processWorkflowRun, defaultMaxRetries)
	c.plrworker = newWorker("PipelineRuns", (*c).processPipelineRun, defaultMaxRetries)

	wfrInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.enqueueWorkflowRun,
		UpdateFunc: passNew(c.enqueueWorkflowRun),
		DeleteFunc: c.enqueueWorkflowRun,
	})

	plrInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: passNew(c.enqueuePipelineRun),
	})

	return c
}

func getTaskContainer(metadataAPIURL string, name string, step *nebulav1.WorkflowStep) (*corev1.Container, []corev1.Volume) {
	volumeMounts := getVolumeMounts(name, step)
	volumes := getVolumes(volumeMounts)
	environmentVariables := buildEnvironmentVariables(metadataAPIURL, name)
	container := getContainer(name, step.Image, volumeMounts, environmentVariables)

	if len(step.Input) > 0 {
		container.Command = []string{NebulaMountPath + "/" + NebulaEntrypointFile}
	} else {
		if len(step.Command) > 0 {
			container.Command = []string{step.Command}
		}
		if len(step.Args) > 0 {
			container.Args = step.Args
		}
	}

	return container, volumes
}

func getContainer(name string, image string, volumeMounts []corev1.VolumeMount, environmentVariables []corev1.EnvVar) *corev1.Container {
	container := &corev1.Container{
		Name:            name,
		Image:           image,
		ImagePullPolicy: corev1.PullAlways,
		VolumeMounts:    volumeMounts,
		Env:             environmentVariables,
		SecurityContext: &corev1.SecurityContext{
			// We can't use RunAsUser et al. here because they don't allow write
			// access to the container filesystem. Eventually, we'll use gVisor
			// to protect us here.
			AllowPrivilegeEscalation: func(b bool) *bool { return &b }(false),
		},
	}

	return container
}

func buildEnvironmentVariablesForCondition() []corev1.EnvVar {
	containerVars := []corev1.EnvVar{
		{
			Name:  "CONDITION",
			Value: "approved",
		},
	}

	return containerVars
}

func buildEnvironmentVariables(metadataAPIURL string, name string) []corev1.EnvVar {
	// this sets the endpoint to the metadata service for accessing the spec
	specPath := path.Join("/", "specs", name)

	containerVars := []corev1.EnvVar{
		{
			// TODO SPEC_URL will change to something else at a later date.
			// This will more than likely become a constant in the nebula-tasks package.
			Name:  "SPEC_URL",
			Value: metadataAPIURL + specPath,
		},
		{
			Name:  "METADATA_API_URL",
			Value: metadataAPIURL,
		},
	}

	return containerVars
}

func getVolumes(volumeMounts []corev1.VolumeMount) []corev1.Volume {
	volumes := make([]corev1.Volume, 0)

	knownVolumes := make(map[string]bool)

	defaultMode := int32(0700)
	for _, volume := range volumeMounts {
		volumeName := volume.Name

		if !knownVolumes[volumeName] {
			thisVolume := corev1.Volume{
				Name: volumeName,
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: volumeName,
						},
						DefaultMode: &defaultMode,
					},
				},
			}

			knownVolumes[volumeName] = true
			volumes = append(volumes, thisVolume)
		}
	}

	return volumes
}

func getVolumeMounts(name string, step *nebulav1.WorkflowStep) []corev1.VolumeMount {
	volumeMounts := make([]corev1.VolumeMount, 0)

	if len(step.Spec) > 0 {
		thisContainerMount := corev1.VolumeMount{
			Name:      name,
			MountPath: NebulaMountPath + "/" + NebulaSpecFile,
			SubPath:   NebulaSpecFile,
		}

		volumeMounts = append(volumeMounts, thisContainerMount)
	}

	if len(step.Input) > 0 {
		thisContainerMount := corev1.VolumeMount{
			Name:      name,
			MountPath: NebulaMountPath + "/" + NebulaEntrypointFile,
			SubPath:   NebulaEntrypointFile,
		}

		volumeMounts = append(volumeMounts, thisContainerMount)
	}

	return volumeMounts
}

func getConfigMapData(workflowParameters nebulav1.WorkflowParameters, workflowRunParameters nebulav1.WorkflowRunParameters, step *nebulav1.WorkflowStep) (map[string]string, errors.Error) {
	configMapData := make(map[string]string)

	if len(step.Spec) > 0 {
		// Inject parameters.
		ev := evaluate.NewEvaluator(
			evaluate.WithResultMapper(evaluate.NewJSONResultMapper()),
			evaluate.WithParameterTypeResolver(resolve.ParameterTypeResolverFunc(func(ctx context.Context, name string) (interface{}, error) {
				if p, ok := workflowRunParameters[name]; ok {
					return p, nil
				} else if p, ok := workflowParameters[name]; ok {
					return p, nil
				}

				return nil, &resolve.ParameterNotFoundError{Name: name}
			})),
		)
		r, err := ev.EvaluateAll(context.TODO(), parse.Tree(step.Spec))
		if err != nil {
			return nil, errors.NewTaskSpecEvaluationError().WithCause(err)
		}

		configMapData[NebulaSpecFile] = string(r.Value.([]byte))
	}

	if len(step.Input) > 0 {
		entrypoint := strings.Join(step.Input, "\n")

		if !strings.HasPrefix(entrypoint, InterpreterDirective) {
			entrypoint = InterpreterDefault + "\n" + entrypoint
		}

		configMapData[NebulaEntrypointFile] = entrypoint
	}

	return configMapData, nil
}

func extractPodAndTaskNamesFromPipelineRun(plr *tekv1alpha1.PipelineRun) []podAndTaskName {
	var result []podAndTaskName
	for _, taskRun := range plr.Status.TaskRuns {
		if nil == taskRun {
			continue
		}
		if nil == taskRun.Status {
			continue
		}
		// Ensure the pod got initialized:
		init := false
		for _, step := range taskRun.Status.Steps {
			if step.Name != taskRun.PipelineTaskName {
				continue
			}
			if nil != step.Terminated || nil != step.Running {
				init = true
			}
		}
		if !init {
			continue
		}
		result = append(result, podAndTaskName{
			PodName:  taskRun.Status.PodName,
			TaskName: taskRun.PipelineTaskName,
		})
	}
	return result
}

func copyImagePullSecret(workingNamespace string, kc kubernetes.Interface, wfr *nebulav1.WorkflowRun, imagePullSecretKey string) (*corev1.Secret, error) {
	namespace, name, err := cache.SplitMetaNamespaceKey(imagePullSecretKey)
	if err != nil {
		return nil, err
	} else if namespace == "" {
		namespace = workingNamespace
	}

	ref, err := kc.CoreV1().Secrets(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	if ref.Type != corev1.SecretType("kubernetes.io/dockerconfigjson") {
		klog.Warningf("image pull secret is not of type kubernetes.io/dockerconfigjson")
	}

	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            metadataImagePullSecretName,
			Namespace:       wfr.GetNamespace(),
			Labels:          getLabels(wfr, nil),
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(wfr, controllerKind)},
		},
		Type: ref.Type,
		Data: ref.Data,
	}

	secret, err = kc.CoreV1().Secrets(wfr.GetNamespace()).Create(secret)
	if k8serrors.IsAlreadyExists(err) {
		secret, err = kc.CoreV1().Secrets(wfr.GetNamespace()).Get(metadataImagePullSecretName, metav1.GetOptions{})
	} else if err != nil {
		return nil, err
	}

	return secret, nil
}

func createServiceAccount(kc kubernetes.Interface, wfr *nebulav1.WorkflowRun, identifier string, imagePullSecret *corev1.Secret) (*corev1.ServiceAccount, error) {
	name := getName(wfr, identifier)

	saccount := &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       wfr.GetNamespace(),
			Labels:          getLabels(wfr, nil),
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(wfr, controllerKind)},
		},
	}

	if imagePullSecret != nil {
		saccount.ImagePullSecrets = []corev1.LocalObjectReference{
			{Name: imagePullSecret.GetName()},
		}
	}

	klog.Infof("creating service account %s", name)
	saccount, err := kc.CoreV1().ServiceAccounts(wfr.GetNamespace()).Create(saccount)
	if k8serrors.IsAlreadyExists(err) {
		saccount, err = kc.CoreV1().ServiceAccounts(wfr.GetNamespace()).Get(name, metav1.GetOptions{})
	}
	if err != nil {
		return nil, err
	}

	return saccount, nil
}

func createRBAC(kc kubernetes.Interface, wfr *nebulav1.WorkflowRun, sa *corev1.ServiceAccount) (*rbacv1.Role, *rbacv1.RoleBinding, error) {
	var err error

	role := &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "Role",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            getName(wfr, ""),
			Namespace:       wfr.GetNamespace(),
			Labels:          getLabels(wfr, nil),
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(wfr, controllerKind)},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"configmaps"},
				Verbs:     []string{"create", "update", "list", "watch", "get"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"list", "watch", "get"},
			},
		},
	}

	binding := &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "RoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            getName(wfr, ""),
			Namespace:       wfr.GetNamespace(),
			Labels:          getLabels(wfr, nil),
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(wfr, controllerKind)},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "Role",
			APIGroup: "rbac.authorization.k8s.io",
			Name:     getName(wfr, ""),
		},
		Subjects: []rbacv1.Subject{
			{
				Name:      sa.GetName(),
				Kind:      "ServiceAccount",
				Namespace: wfr.GetNamespace(),
			},
		},
	}

	klog.Infof("creating role %s", wfr.GetName())
	role, err = kc.RbacV1().Roles(wfr.GetNamespace()).Create(role)
	if k8serrors.IsAlreadyExists(err) {
		role, err = kc.RbacV1().Roles(wfr.GetNamespace()).Get(getName(wfr, ""), metav1.GetOptions{})
	}
	if err != nil {
		return nil, nil, err
	}

	klog.Infof("creating role binding %s", wfr.GetName())
	binding, err = kc.RbacV1().RoleBindings(wfr.GetNamespace()).Create(binding)
	if k8serrors.IsAlreadyExists(err) {
		binding, err = kc.RbacV1().RoleBindings(wfr.GetNamespace()).Get(getName(wfr, ""), metav1.GetOptions{})
	}
	if err != nil {
		return nil, nil, err
	}

	return role, binding, nil
}

func createMetadataAPIPod(kc kubernetes.Interface, image string, saccount *corev1.ServiceAccount,
	wfr *nebulav1.WorkflowRun, secretsAddr, secretsAuthMountPath, scopedSecretsPath string) (*corev1.Pod, error) {

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      metadataServiceName,
			Namespace: wfr.GetNamespace(),
			Labels: getLabels(wfr, map[string]string{
				"app.kubernetes.io/name":      "nebula",
				"app.kubernetes.io/component": metadataServiceName,
			}),
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(wfr, controllerKind)},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:            metadataServiceName,
					Image:           image,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command: []string{
						"/usr/bin/nebula-metadata-api",
						"-bind-addr",
						":7000",
						"-vault-addr",
						secretsAddr,
						"-vault-auth-mount-path",
						secretsAuthMountPath,
						"-vault-role",
						wfr.GetNamespace(),
						"-scoped-secrets-path",
						scopedSecretsPath,
						"-namespace",
						wfr.GetNamespace(),
					},
					Ports: []corev1.ContainerPort{
						{
							Name:          "http",
							ContainerPort: 7000,
						},
					},
					ReadinessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							HTTPGet: &corev1.HTTPGetAction{
								Path: "/healthz",
								Port: intstr.FromInt(7000),
							},
						},
					},
				},
			},
			ServiceAccountName: saccount.GetName(),
			RestartPolicy:      corev1.RestartPolicyOnFailure,
		},
	}

	klog.Infof("creating metadata service pod %s", wfr.GetName())

	pod, err := kc.CoreV1().Pods(wfr.GetNamespace()).Create(pod)
	if k8serrors.IsAlreadyExists(err) {
		pod, err = kc.CoreV1().Pods(wfr.GetNamespace()).Get(metadataServiceName, metav1.GetOptions{})
	}
	if err != nil {
		return nil, err
	}

	return pod, nil
}

func createMetadataAPIService(kc kubernetes.Interface, wfr *nebulav1.WorkflowRun) (*corev1.Service, error) {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      metadataServiceName,
			Namespace: wfr.GetNamespace(),
			Labels: getLabels(wfr, map[string]string{
				"app.kubernetes.io/name":      "nebula",
				"app.kubernetes.io/component": metadataServiceName,
			}),
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(wfr, controllerKind)},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port:       80,
					TargetPort: intstr.FromInt(7000),
				},
			},
			Selector: map[string]string{
				"app.kubernetes.io/name":      "nebula",
				"app.kubernetes.io/component": metadataServiceName,
			},
		},
	}

	klog.Infof("creating pod service %s", wfr.GetName())

	service, err := kc.CoreV1().Services(wfr.GetNamespace()).Create(service)
	if k8serrors.IsAlreadyExists(err) {
		service, err = kc.CoreV1().Services(wfr.GetNamespace()).Get(metadataServiceName, metav1.GetOptions{})
	}
	if err != nil {
		return nil, err
	}

	return service, nil
}

func areWeDoneYet(plr *tekv1alpha1.PipelineRun) bool {
	if !plr.IsDone() && !plr.IsCancelled() {
		return false
	}

	for _, task := range plr.Status.TaskRuns {
		if task.Status == nil {
			continue
		}

		status := mapStatus(task.Status.Status)
		if status == WorkflowRunStatusInProgress {
			return false
		}
	}

	return true
}

func isCancelled(wr *nebulav1.WorkflowRun) bool {
	cancelled := false
	workflowState := wr.State.Workflow
	if cancelState, ok := workflowState[WorkflowRunStateCancel]; ok {
		cancelled, ok = cancelState.(bool)
	}

	return cancelled
}

func getName(wfr *nebulav1.WorkflowRun, name string) string {
	prefix := "wr"

	if name == "" {
		return fmt.Sprintf("%s-%s", prefix, wfr.Spec.Name)
	}

	return fmt.Sprintf("%s-%s-%s", prefix, wfr.GetName(), name)
}

func getLabels(wfr *nebulav1.WorkflowRun, additional map[string]string) map[string]string {
	workflowRunLabels := map[string]string{
		workflowRunLabel: wfr.Spec.Name,
		workflowLabel:    wfr.Spec.Workflow.Name,
	}

	if additional != nil {
		for k, v := range additional {
			workflowRunLabels[k] = v
		}
	}

	return workflowRunLabels
}

func mapStatus(status duckv1beta1.Status) WorkflowRunStatus {
	for _, cs := range status.Conditions {
		switch cs.Type {
		case apis.ConditionSucceeded:
			switch cs.Status {
			case corev1.ConditionUnknown:
				return WorkflowRunStatusInProgress
			case corev1.ConditionTrue:
				return WorkflowRunStatusSuccess
			case corev1.ConditionFalse:
				if cs.Reason == ReasonConditionCheckFailed {
					return WorkflowRunStatusSkipped
				}
				// TODO Ignore until this is recognized as a valid status
				//if cs.Reason == ReasonTimedOut {
				//	return WorkflowRunStatusTimedOut
				//}
				return WorkflowRunStatusFailure
			}
		}
	}

	return WorkflowRunStatusPending
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}

func passNew(f func(interface{})) func(interface{}, interface{}) {
	return func(first, second interface{}) {
		f(second)
	}
}
