package workflow

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/puppetlabs/horsehead/v2/storage"
	tekv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tekclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	teklisters "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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

	nebulav1 "github.com/puppetlabs/nebula-tasks/pkg/apis/nebula.puppet.com/v1"
	"github.com/puppetlabs/nebula-tasks/pkg/config"
	clientset "github.com/puppetlabs/nebula-tasks/pkg/generated/clientset/versioned"
	neblisters "github.com/puppetlabs/nebula-tasks/pkg/generated/listers/nebula.puppet.com/v1"
	"github.com/puppetlabs/nebula-tasks/pkg/secrets/vault"
	"github.com/puppetlabs/nebula-tasks/pkg/util"
)

var controllerKind = nebulav1.SchemeGroupVersion.WithKind("WorkflowRun")

const (
	pipelineRunAnnotation = "nebula.puppet.com/pipelinerun"
	workflowRunAnnotation = "nebula.puppet.com/workflowrun"
)

type WorkflowRunStatus string
type WorkflowRunStepStatus string

const (
	WorkflowRunStepStatusPending    WorkflowRunStepStatus = "pending"
	WorkflowRunStepStatusInProgress WorkflowRunStepStatus = "in-progress"
	WorkflowRunStepStatusSuccess    WorkflowRunStepStatus = "success"
	WorkflowRunStepStatusFailure    WorkflowRunStepStatus = "failure"
)

const (
	WorkflowRunStatusPending    WorkflowRunStatus = "pending"
	WorkflowRunStatusInProgress WorkflowRunStatus = "in-progress"
	WorkflowRunStatusSuccess    WorkflowRunStatus = "success"
	WorkflowRunStatusFailure    WorkflowRunStatus = "failure"
)

const (
	// default name for the workflow metadata api pod and service
	metadataServiceName = "metadata-api"

	// name for the image pull secret used by the metadata API, if needed
	metadataImagePullSecretName = "metadata-api-docker-registry"

	// PipelineRun annotation indicating the log upload location
	logUploadAnnotationPrefix = "nebula.puppet.com/log-archive-"
)

type podAndTaskName struct {
	PodName  string
	TaskName string
}

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

func (c *Controller) processPipelineRunChange(ctx context.Context, key string) error {
	klog.Infof("syncing PipelineRun change %s", key)
	defer klog.Infof("done syncing PipelineRun change %s", key)

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	plr, err := c.tekclient.TektonV1alpha1().PipelineRuns(namespace).Get(name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}

	labelMap := map[string]string{
		"pipeline-id": plr.Name,
	}

	selector := labels.SelectorFromValidatedSet(labelMap)
	workflowList, err := c.wfrLister.WorkflowRuns(namespace).List(selector)
	if err != nil {
		return err
	}

	logAnnotations := make(map[string]string, 0)

	if plr.IsDone() {
		// Upload the logs that are not defined on the PipelineRun record...
		logAnnotations, err = c.uploadLogs(ctx, plr)
		if nil != err {
			return err
		}

		// FIXME This will be removed later in favor of the WorkflowRun
		for name, value := range logAnnotations {
			metav1.SetMetaDataAnnotation(&plr.ObjectMeta, name, value)
		}

		plr, err = c.tekclient.TektonV1alpha1().PipelineRuns(plr.Namespace).Update(plr)
	}

	for _, workflow := range workflowList {
		klog.Infof("revoking workflow run secret access %s", workflow.GetName())
		if err := c.secretsclient.RevokeScopedAccess(ctx, namespace); err != nil {
			return err
		}

		status, _ := c.updateWorkflowRunStatus(plr)

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
		annotation := util.Slug(logUploadAnnotationPrefix + pt.TaskName)
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
	klog.Infof("syncing WorkflowRun change %s", key)
	defer klog.Infof("done syncing WorkflowRun change %s", key)

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	wr, err := c.nebclient.NebulaV1().WorkflowRuns(namespace).Get(name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		klog.Infof("%s %s has been deleted", wr.Kind, key)

		return nil
	}
	if err != nil {
		return err
	}

	// If we haven't set the state of the run yet, then we need to ensure all the secret access
	// and rbac is setup.
	if wr.Status.Status == "" {
		if err := c.ensureAccessResourcesExist(ctx, wr); err != nil {
			return err
		}
	}

	if wr.ObjectMeta.DeletionTimestamp.IsZero() {
		if _, ok := wr.GetAnnotations()[pipelineRunAnnotation]; !ok {
			plr, err := c.createPipelineRun(wr)
			if err != nil {
				return err
			}

			pipelineId := wr.Spec.Name
			if wr.Labels == nil {
				wr.Labels = make(map[string]string, 0)
			}
			wr.Labels["pipeline-id"] = pipelineId

			metav1.SetMetaDataAnnotation(&wr.ObjectMeta, pipelineRunAnnotation, plr.Name)

			if !containsString(wr.ObjectMeta.Finalizers, workflowRunAnnotation) {
				wr.ObjectMeta.Finalizers = append(wr.ObjectMeta.Finalizers, workflowRunAnnotation)
			}

			wr, err = c.nebclient.NebulaV1().WorkflowRuns(namespace).Update(wr)
			if err != nil {
				return err
			}
		}
	} else {
		if containsString(wr.ObjectMeta.Finalizers, workflowRunAnnotation) {
			wr.ObjectMeta.Finalizers = removeString(wr.ObjectMeta.Finalizers, workflowRunAnnotation)

			wr, err = c.nebclient.NebulaV1().WorkflowRuns(namespace).Update(wr)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *Controller) ensureAccessResourcesExist(ctx context.Context, wr *nebulav1.WorkflowRun) error {
	var (
		ips      *corev1.Secret
		saccount *corev1.ServiceAccount
		service  *corev1.Service
		err      error
	)

	namespace := wr.GetNamespace()

	if c.cfg.MetadataServiceImagePullSecret != "" {
		klog.Info("copying secret for metadata service image")
		ips, err = copyImagePullSecret(c.namespace, c.kubeclient, wr, c.cfg.MetadataServiceImagePullSecret)
		if err != nil {
			return err
		}
	}

	saccount, err = createServiceAccount(c.kubeclient, wr, ips)
	if err != nil {
		return err
	}

	klog.Infof("granting workflow run access to scoped secrets %s", wr.Spec.Workflow.Name)
	grant, err := c.secretsclient.GrantScopedAccess(ctx, wr.Spec.Workflow.Name, namespace, saccount.GetName())
	if err != nil {
		return err
	}

	_, _, err = createRBAC(c.kubeclient, wr)
	if err != nil {
		return err
	}

	// It is possible that the metadata service and this controller talk to
	// different Vault endpoints: each might be talking to a Vault agent (for
	// caching or additional security) instead of directly to the Vault server.
	podVaultAddr := c.cfg.MetadataServiceVaultAddr
	if podVaultAddr == "" {
		podVaultAddr = grant.BackendAddr
	}

	_, err = createMetadataAPIPod(
		c.kubeclient,
		c.cfg.MetadataServiceImage,
		saccount,
		wr,
		podVaultAddr,
		grant.ScopedPath,
	)
	if err != nil {
		return err
	}

	service, err = createMetadataAPIService(c.kubeclient, wr)
	if err != nil {
		return err
	}

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

	klog.Infof("metadata service is ready %s", wr.Spec.Workflow.Name)

	return nil
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
	namespace := wr.GetNamespace()

	plr, err := c.tekclient.TektonV1alpha1().PipelineRuns(namespace).Get(wr.Spec.Name, metav1.GetOptions{})
	if plr != nil && plr != (&tekv1alpha1.PipelineRun{}) && plr.Name != "" {
		return plr, nil
	}

	runID := wr.Spec.Name

	labelMap := map[string]string{
		pipelineRunAnnotation: runID,
		"workflow-run-id":     runID,
		"workflow-id":         wr.Spec.Workflow.Name,
	}

	serviceAccount, err := c.createServiceAccount(wr)
	if err != nil {
		return nil, err
	}

	pipelineRun := &tekv1alpha1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:            runID,
			Namespace:       namespace,
			Labels:          labelMap,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(wr, controllerKind)},
		},
		Spec: tekv1alpha1.PipelineRunSpec{
			ServiceAccount: serviceAccount.Name,
			PipelineRef: tekv1alpha1.PipelineRef{
				Name: runID,
			},
			PodTemplate: tekv1alpha1.PodTemplate{
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

func (c *Controller) createServiceAccount(wr *nebulav1.WorkflowRun) (*corev1.ServiceAccount, error) {
	namespace := wr.GetNamespace()

	serviceAccount, _ := c.kubeclient.CoreV1().ServiceAccounts(namespace).Get(wr.Spec.Name, metav1.GetOptions{})
	if serviceAccount != nil {
		return serviceAccount, nil
	}

	serviceAccount = &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:            wr.Spec.Name,
			Namespace:       namespace,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(wr, controllerKind)},
		},
	}

	return c.kubeclient.CoreV1().ServiceAccounts(namespace).Create(serviceAccount)
}

func (c *Controller) updateWorkflowRunStatus(plr *tekv1alpha1.PipelineRun) (*nebulav1.WorkflowRunStatus, error) {
	workflowRunSteps := make(map[string]nebulav1.WorkflowRunStep)

	for _, taskRun := range plr.Status.TaskRuns {
		if nil == taskRun.Status {
			continue
		}
		step := nebulav1.WorkflowRunStep{
			Name:   taskRun.PipelineTaskName,
			Status: string(mapTaskStatus(taskRun)),
		}

		if taskRun.Status.StartTime != nil {
			step.StartTime = taskRun.Status.StartTime
		}
		if taskRun.Status.CompletionTime != nil {
			step.CompletionTime = taskRun.Status.CompletionTime
		}

		workflowRunSteps[taskRun.PipelineTaskName] = step
	}

	status := taskStatusesToRunStatus(workflowRunSteps)

	workflowRunStatus := &nebulav1.WorkflowRunStatus{
		Status: string(status),
		Steps:  workflowRunSteps,
	}

	if plr.Status.StartTime != nil {
		workflowRunStatus.StartTime = plr.Status.StartTime
	}
	if plr.Status.CompletionTime != nil {
		workflowRunStatus.CompletionTime = plr.Status.CompletionTime
	}

	return workflowRunStatus, nil
}

func NewController(manager *DependencyManager, cfg *config.WorkflowControllerConfig, vc *vault.VaultAuth, bs storage.BlobStore, namespace string) *Controller {
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
	}

	c.wfrworker = newWorker("WorkflowRuns", (*c).processWorkflowRun, defaultMaxRetries)
	c.plrworker = newWorker("PipelineRuns", (*c).processPipelineRunChange, defaultMaxRetries)

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
	if errors.IsAlreadyExists(err) {
		secret, err = kc.CoreV1().Secrets(wfr.GetNamespace()).Get(secret.GetName(), metav1.GetOptions{})
	} else if err != nil {
		return nil, err
	}

	return secret, nil
}

func createServiceAccount(kc kubernetes.Interface, wfr *nebulav1.WorkflowRun, imagePullSecret *corev1.Secret) (*corev1.ServiceAccount, error) {
	saccount := &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            getName(wfr, ""),
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

	klog.Infof("creating service account %s", wfr.Spec.Workflow.Name)
	saccount, err := kc.CoreV1().ServiceAccounts(wfr.GetNamespace()).Create(saccount)
	if errors.IsAlreadyExists(err) {
		saccount, err = kc.CoreV1().ServiceAccounts(wfr.GetNamespace()).Get(getName(wfr, ""), metav1.GetOptions{})
	}
	if err != nil {
		return nil, err
	}

	return saccount, nil
}

func createRBAC(kc kubernetes.Interface, wfr *nebulav1.WorkflowRun) (*rbacv1.Role, *rbacv1.RoleBinding, error) {
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
				Name:      getName(wfr, ""),
				Kind:      "ServiceAccount",
				Namespace: wfr.GetNamespace(),
			},
		},
	}

	klog.Infof("creating role %s", wfr.Spec.Workflow.Name)
	role, err = kc.RbacV1().Roles(wfr.GetNamespace()).Create(role)
	if errors.IsAlreadyExists(err) {
		role, err = kc.RbacV1().Roles(wfr.GetNamespace()).Get(getName(wfr, ""), metav1.GetOptions{})
	}
	if err != nil {
		return nil, nil, err
	}

	klog.Infof("creating role binding %s", wfr.Spec.Workflow.Name)
	binding, err = kc.RbacV1().RoleBindings(wfr.GetNamespace()).Create(binding)
	if errors.IsAlreadyExists(err) {
		binding, err = kc.RbacV1().RoleBindings(wfr.GetNamespace()).Get(getName(wfr, ""), metav1.GetOptions{})
	}
	if err != nil {
		return nil, nil, err
	}

	return role, binding, nil
}

func createMetadataAPIPod(kc kubernetes.Interface, image string, saccount *corev1.ServiceAccount,
	wfr *nebulav1.WorkflowRun, secretsAddr, scopedSecretsPath string) (*corev1.Pod, error) {

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
						"-vault-role",
						wfr.GetNamespace(),
						"-workflow-id",
						wfr.Spec.Workflow.Name,
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

	klog.Infof("creating metadata service pod %s", wfr.Spec.Workflow.Name)

	pod, err := kc.CoreV1().Pods(wfr.GetNamespace()).Create(pod)
	if errors.IsAlreadyExists(err) {
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

	klog.Infof("creating pod service %s", wfr.Spec.Workflow.Name)

	service, err := kc.CoreV1().Services(wfr.GetNamespace()).Create(service)
	if errors.IsAlreadyExists(err) {
		service, err = kc.CoreV1().Services(wfr.GetNamespace()).Get(metadataServiceName, metav1.GetOptions{})
	}
	if err != nil {
		return nil, err
	}

	return service, nil
}

func getName(wfr *nebulav1.WorkflowRun, name string) string {
	prefix := "wr"

	if name == "" {
		return fmt.Sprintf("%s-%s", prefix, wfr.Spec.Name)
	}

	return fmt.Sprintf("%s-%s-%s", prefix, wfr.Spec.Name, name)
}

func getLabels(wfr *nebulav1.WorkflowRun, additional map[string]string) map[string]string {
	labels := map[string]string{
		"nebula.puppet.com/workflow-run-id": wfr.Spec.Name,
		"nebula.puppet.com/workflow-id":     wfr.Spec.Workflow.Name,
	}

	if additional != nil {
		for k, v := range additional {
			labels[k] = v
		}
	}

	return labels
}

func mapTaskStatus(taskRun *tekv1alpha1.PipelineRunTaskRunStatus) WorkflowRunStepStatus {
	for _, cs := range taskRun.Status.Conditions {
		switch cs.Type {
		case apis.ConditionSucceeded:
			switch cs.Status {
			case corev1.ConditionUnknown:
				return WorkflowRunStepStatusInProgress
			case corev1.ConditionTrue:
				return WorkflowRunStepStatusSuccess
			case corev1.ConditionFalse:
				return WorkflowRunStepStatusFailure
			}
		}
	}

	return WorkflowRunStepStatusPending
}

func taskStatusesToRunStatus(tss map[string]nebulav1.WorkflowRunStep) WorkflowRunStatus {
	if tss == nil || len(tss) <= 0 {
		return WorkflowRunStatusPending
	}

	wrs := WorkflowRunStatusSuccess
	for _, rts := range tss {
		switch WorkflowRunStepStatus(rts.Status) {
		case WorkflowRunStepStatusFailure:
			wrs = WorkflowRunStatusFailure
			return wrs
		case WorkflowRunStepStatusPending:
			if wrs != WorkflowRunStatusInProgress {
				wrs = WorkflowRunStatusPending
			}
		case WorkflowRunStepStatusInProgress:
			wrs = WorkflowRunStatusInProgress
		}
	}

	return wrs
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
