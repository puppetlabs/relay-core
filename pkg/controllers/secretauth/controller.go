package secretauth

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/puppetlabs/horsehead/storage"
	tekv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tekclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	tekinformers "github.com/tektoncd/pipeline/pkg/client/informers/externalversions"
	tekv1informer "github.com/tektoncd/pipeline/pkg/client/informers/externalversions/pipeline/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/klog"

	nebulav1 "github.com/puppetlabs/nebula-tasks/pkg/apis/nebula.puppet.com/v1"
	"github.com/puppetlabs/nebula-tasks/pkg/config"
	"github.com/puppetlabs/nebula-tasks/pkg/data/secrets/vault"
	clientset "github.com/puppetlabs/nebula-tasks/pkg/generated/clientset/versioned"
	informers "github.com/puppetlabs/nebula-tasks/pkg/generated/informers/externalversions"
	sainformers "github.com/puppetlabs/nebula-tasks/pkg/generated/informers/externalversions/nebula.puppet.com/v1"
)

const (
	// default name for the workflow metadata api pod and service
	metadataServiceName = "metadata-api"

	// name for the image pull secret used by the metadata API, if needed
	metadataImagePullSecretName = "metadata-api-docker-registry"

	// PipelineRun annotation indicating the log upload location
	logUploadAnnotationPrefix = "nebula.puppet.com/log-archive-"
	MaxLogArchiveTries        = 7
)

type podAndTaskName struct {
	PodName  string
	TaskName string
}

// Controller watches for nebulav1.SecretAuth resource changes.
// If a SecretAuth resource is created, the controller will create a service acccount + rbac
// for the namespace, then inform vault that that service account is allowed to access
// readonly secrets under a preconfigured path related to a nebula workflow. It will then
// spin up a pod running an instance of nebula-metadata-api that knows how to
// ask kubernetes for the service account token, that it will use to proxy secrets
// between the task pods and the vault server.
type Controller struct {
	kubeclientconfig   clientcmd.ClientConfig
	kubeclient         kubernetes.Interface
	nebclient          clientset.Interface
	tekclient          tekclientset.Interface
	nebInformerFactory informers.SharedInformerFactory
	saInformer         sainformers.SecretAuthInformer
	saInformerSynced   cache.InformerSynced
	tekInformerFactory tekinformers.SharedInformerFactory
	plrInformer        tekv1informer.PipelineRunInformer
	plrInformerSynced  cache.InformerSynced
	saworker           *worker
	plrworker          *worker
	vaultClient        *vault.VaultAuth
	blobStore          storage.BlobStore

	cfg *config.SecretAuthControllerConfig
}

// Run starts all required informers and spawns two worker goroutines
// that will pull resource objects off the workqueue. This method blocks
// until stopCh is closed or an earlier bootstrap call results in an error.
func (c *Controller) Run(numWorkers int, stopCh chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.saworker.shutdown()
	defer c.plrworker.shutdown()

	c.nebInformerFactory.Start(stopCh)

	if ok := cache.WaitForCacheSync(stopCh, c.saInformerSynced); !ok {
		return fmt.Errorf("failed to wait for informer cache to sync")
	}

	c.tekInformerFactory.Start(stopCh)

	if ok := cache.WaitForCacheSync(stopCh, c.plrInformerSynced); !ok {
		return fmt.Errorf("failed to wait for informer cache to sync")
	}

	c.saworker.run(numWorkers, stopCh)
	c.plrworker.run(numWorkers, stopCh)

	<-stopCh

	return nil
}

// processSingleItem is responsible for creating all the resouces required for
// secret handling and authentication.
// TODO break this logic out into smaller chunks... especially the calls to the vault api
func (c *Controller) processSingleItem(ctx context.Context, key string) error {
	klog.Infof("syncing SecretAuth %s", key)
	defer klog.Infof("done syncing SecretAuth %s", key)

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	sa, err := c.nebclient.NebulaV1().SecretAuths(namespace).Get(name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}

	// if anything fails while creating resources, the status object will not be filled out
	// and saved. this means that if any of the keys are empty, we haven't created resources yet.
	if sa.Status.ServiceAccount != "" {
		klog.Infof("resources have already been created %s", key)
		return nil
	}

	var (
		ips       *corev1.Secret
		saccount  *corev1.ServiceAccount
		role      *rbacv1.Role
		binding   *rbacv1.RoleBinding
		pod       *corev1.Pod
		service   *corev1.Service
		configMap *corev1.ConfigMap
	)

	if c.cfg.MetadataServiceImagePullSecret != "" {
		klog.Info("copying secret for metadata service image")
		ips, err = copyImagePullSecret(c.kubeclientconfig, c.kubeclient, sa, c.cfg.MetadataServiceImagePullSecret)
		if err != nil {
			return err
		}
	}

	saccount, err = createServiceAccount(c.kubeclient, sa, ips)
	if err != nil {
		return err
	}

	klog.Infof("writing vault readonly access policy %s", sa.Spec.WorkflowID)
	// now we let vault know about the service account
	if err := c.vaultClient.WritePolicy(namespace, sa.Spec.WorkflowID); err != nil {
		return err
	}

	klog.Infof("enabling vault access for workflow service account %s", sa.Spec.WorkflowID)
	if err := c.vaultClient.WriteRole(namespace, saccount.GetName(), namespace); err != nil {
		return err
	}

	role, binding, err = createRBAC(c.kubeclient, sa)
	if err != nil {
		return err
	}

	// It is possible that the metadata service and this controller talk to
	// different Vault endpoints: each might be talking to a Vault agent (for
	// caching or additional security) instead of directly to the Vault server.
	podVaultAddr := c.cfg.MetadataServiceVaultAddr
	if podVaultAddr == "" {
		podVaultAddr = c.vaultClient.Address()
	}

	pod, err = createMetadataAPIPod(
		c.kubeclient,
		c.cfg.MetadataServiceImage,
		saccount,
		sa,
		podVaultAddr,
		c.vaultClient.EngineMount(),
	)
	if err != nil {
		return err
	}

	service, err = createMetadataAPIService(c.kubeclient, sa)
	if err != nil {
		return err
	}

	configMap, err = createWorkflowConfigMap(c.kubeclient, service, sa)
	if err != nil {
		return err
	}

	klog.Infof("waiting for metadata service to become ready %s", sa.Spec.WorkflowID)

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

	klog.Infof("metadata service is ready %s", sa.Spec.WorkflowID)

	saCopy := sa.DeepCopy()
	saCopy.Status.MetadataServicePod = pod.GetName()
	saCopy.Status.MetadataServiceService = service.GetName()
	saCopy.Status.ServiceAccount = saccount.GetName()
	saCopy.Status.ConfigMap = configMap.GetName()
	saCopy.Status.Role = role.GetName()
	saCopy.Status.RoleBinding = binding.GetName()
	saCopy.Status.VaultPolicy = namespace
	saCopy.Status.VaultAuthRole = namespace

	if ips != nil {
		saCopy.Status.MetadataServiceImagePullSecret = ips.GetName()
	}

	klog.Info("updating secretauth resource status for ", sa.Spec.WorkflowID)
	saCopy, err = c.nebclient.NebulaV1().SecretAuths(namespace).Update(saCopy)
	if err != nil {
		return err
	}

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
		return fmt.Errorf("timeout occurred while waiting for the metadata service to be ready")
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
		resp, err := http.Get(u)
		if err != nil {
			klog.Infof("got an error when probing the metadata api %s", err)

			return false, nil
		}

		if resp.StatusCode != http.StatusOK {
			klog.Infof("got an invalid status code when probing the metadata api %d", resp.StatusCode)

			return false, nil
		}

		return true, nil
	})
}

func (c *Controller) enqueueSecretAuth(obj interface{}) {
	sa := obj.(*nebulav1.SecretAuth)

	key, err := cache.MetaNamespaceKeyFunc(sa)
	if err != nil {
		utilruntime.HandleError(err)

		return
	}

	c.saworker.add(key)
}

func (c *Controller) enqueuePipelineRunChange(old, obj interface{}) {
	// old is ignored because we only care about the current state
	plr := obj.(*tekv1alpha1.PipelineRun)

	key, err := cache.MetaNamespaceKeyFunc(plr)
	if err != nil {
		utilruntime.HandleError(err)

		return
	}

	c.plrworker.add(key)
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
		// TODO if the pipeline run isn't found, then we will still need to clean up SecretAuth
		// resources, but the business logic for this still needs to be defined
		return nil
	}
	if err != nil {
		return err
	}

	if plr.IsDone() {
		// Upload the logs that are not defined on the PipelineRun record...
		err = c.uploadLogs(ctx, plr)
		if nil != err {
			return err
		}

		sas, err := c.nebclient.NebulaV1().SecretAuths(namespace).List(metav1.ListOptions{})
		if err != nil {
			return err
		}

		core := c.kubeclient.CoreV1()
		rbac := c.kubeclient.RbacV1()
		sac := c.nebclient.NebulaV1().SecretAuths(namespace)
		opts := &metav1.DeleteOptions{}

		for _, sa := range sas.Items {
			klog.Infof("deleting resources created by %s", sa.GetName())

			err = core.Pods(namespace).Delete(sa.Status.MetadataServicePod, opts)
			if err != nil {
				return err
			}

			err = core.Services(namespace).Delete(sa.Status.MetadataServiceService, opts)
			if err != nil {
				return err
			}

			err = core.ServiceAccounts(namespace).Delete(sa.Status.ServiceAccount, opts)
			if err != nil {
				return err
			}

			err = core.ConfigMaps(namespace).Delete(sa.Status.ConfigMap, opts)
			if err != nil {
				return err
			}

			if sa.Status.MetadataServiceImagePullSecret != "" {
				err = core.Secrets(namespace).Delete(sa.Status.MetadataServiceImagePullSecret, opts)
				if err != nil {
					return err
				}
			}

			err = rbac.RoleBindings(namespace).Delete(sa.Status.RoleBinding, opts)
			if err != nil {
				return err
			}

			err = rbac.Roles(namespace).Delete(sa.Status.Role, opts)
			if err != nil {
				return err
			}

			err = c.vaultClient.DeleteRole(sa.Status.VaultAuthRole)
			if err != nil {
				return err
			}

			err = c.vaultClient.DeletePolicy(sa.Status.VaultPolicy)
			if err != nil {
				return err
			}

			if err := sac.Delete(sa.GetName(), opts); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *Controller) uploadLogs(ctx context.Context, plr *tekv1alpha1.PipelineRun) error {
	for _, pt := range extractPodAndTaskNamesFromPipelineRun(plr) {
		annotation := logUploadAnnotationPrefix + pt.TaskName
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
			return err
		}
		for retry := uint(0); ; retry++ {
			metav1.SetMetaDataAnnotation(&plr.ObjectMeta, annotation, logName)
			plr, err = c.tekclient.TektonV1alpha1().PipelineRuns(plr.Namespace).Update(plr)
			if nil == err {
				break
			} else if !errors.IsConflict(err) {
				return err
			}
			if retry > MaxLogArchiveTries {
				return err
			}
			klog.Warningf("Conflict during pipelineRun=%s/%s update",
				plr.Namespace,
				plr.Name)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After((1 << retry) * 10 * time.Millisecond):
			}
			plr, err = c.tekclient.TektonV1alpha1().
				PipelineRuns(plr.Namespace).
				Get(plr.Name, metav1.GetOptions{})
			if nil != err {
				return err
			}
		}
	}
	return nil
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

	err = c.blobStore.Put(ctx, key, func(w io.Writer) error {
		_, err := io.Copy(w, rc)
		return err
	}, storage.PutOptions{
		ContentType: "application/octet-stream",
	})
	if err != nil {
		return "", err
	}
	return key, nil
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

func NewController(cfg *config.SecretAuthControllerConfig, vaultClient *vault.VaultAuth, blobStore storage.BlobStore) (*Controller, error) {
	// The following two statements are essentially equivalent to calling
	// clientcmd.BuildConfigFromFlags(), but we need the Namespace() method from
	// the clientcmd.ClientConfig, so we have to unwrap it.
	kcfg := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: cfg.Kubeconfig},
		&clientcmd.ConfigOverrides{ClusterInfo: clientcmdapi.Cluster{Server: cfg.KubeMasterURL}},
	)

	kcc, err := kcfg.ClientConfig()
	if err != nil {
		return nil, err
	}

	kc, err := kubernetes.NewForConfig(kcc)
	if err != nil {
		return nil, err
	}

	nebclient, err := clientset.NewForConfig(kcc)
	if err != nil {
		return nil, err
	}

	tekclient, err := tekclientset.NewForConfig(kcc)
	if err != nil {
		return nil, err
	}

	nebInformerFactory := informers.NewSharedInformerFactory(nebclient, time.Second*30)
	saInformer := nebInformerFactory.Nebula().V1().SecretAuths()

	tekInformerFactory := tekinformers.NewSharedInformerFactory(tekclient, time.Second*30)
	plrInformer := tekInformerFactory.Tekton().V1alpha1().PipelineRuns()

	c := &Controller{
		kubeclientconfig:   kcfg,
		kubeclient:         kc,
		nebclient:          nebclient,
		tekclient:          tekclient,
		nebInformerFactory: nebInformerFactory,
		saInformer:         saInformer,
		saInformerSynced:   saInformer.Informer().HasSynced,
		tekInformerFactory: tekInformerFactory,
		plrInformer:        plrInformer,
		plrInformerSynced:  plrInformer.Informer().HasSynced,
		vaultClient:        vaultClient,
		blobStore:          blobStore,
		cfg:                cfg,
	}

	c.saworker = newWorker("SecretAuths", (*c).processSingleItem, defaultMaxRetries)
	c.plrworker = newWorker("PipelineRuns", (*c).processPipelineRunChange, defaultMaxRetries)

	saInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueueSecretAuth,
	})

	plrInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: c.enqueuePipelineRunChange,
	})

	return c, nil
}

func copyImagePullSecret(kcfg clientcmd.ClientConfig, kc kubernetes.Interface, sa *nebulav1.SecretAuth, imagePullSecretKey string) (*corev1.Secret, error) {
	namespace, name, err := cache.SplitMetaNamespaceKey(imagePullSecretKey)
	if err != nil {
		return nil, err
	} else if namespace == "" {
		namespace, _, _ = kcfg.Namespace()
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
			Name:      metadataImagePullSecretName,
			Namespace: sa.GetNamespace(),
		},
		Type: ref.Type,
		Data: ref.Data,
	}

	secret, err = kc.CoreV1().Secrets(sa.GetNamespace()).Create(secret)
	if errors.IsAlreadyExists(err) {
		secret, err = kc.CoreV1().Secrets(sa.GetNamespace()).Get(secret.GetName(), metav1.GetOptions{})
	} else if err != nil {
		return nil, err
	}

	return secret, nil
}

func createServiceAccount(kc kubernetes.Interface, sa *nebulav1.SecretAuth, imagePullSecret *corev1.Secret) (*corev1.ServiceAccount, error) {
	saccount := &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      getName(sa, ""),
			Namespace: sa.GetNamespace(),
			Labels:    getLabels(sa, nil),
		},
	}

	if imagePullSecret != nil {
		saccount.ImagePullSecrets = []corev1.LocalObjectReference{
			{Name: imagePullSecret.GetName()},
		}
	}

	klog.Infof("creating service account %s", sa.Spec.WorkflowID)
	saccount, err := kc.CoreV1().ServiceAccounts(sa.GetNamespace()).Create(saccount)
	if errors.IsAlreadyExists(err) {
		saccount, err = kc.CoreV1().ServiceAccounts(sa.GetNamespace()).Get(getName(sa, ""), metav1.GetOptions{})
	}
	if err != nil {
		return nil, err
	}

	return saccount, nil
}

func createRBAC(kc kubernetes.Interface, sa *nebulav1.SecretAuth) (*rbacv1.Role, *rbacv1.RoleBinding, error) {
	var err error

	role := &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "Role",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      getName(sa, ""),
			Namespace: sa.GetNamespace(),
			Labels:    getLabels(sa, nil),
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"configmaps"},
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
			Name:      getName(sa, ""),
			Namespace: sa.GetNamespace(),
			Labels:    getLabels(sa, nil),
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "Role",
			APIGroup: "rbac.authorization.k8s.io",
			Name:     getName(sa, ""),
		},
		Subjects: []rbacv1.Subject{
			{
				Name:      getName(sa, ""),
				Kind:      "ServiceAccount",
				Namespace: sa.GetNamespace(),
			},
		},
	}

	klog.Infof("creating role %s", sa.Spec.WorkflowID)
	role, err = kc.RbacV1().Roles(sa.GetNamespace()).Create(role)
	if errors.IsAlreadyExists(err) {
		role, err = kc.RbacV1().Roles(sa.GetNamespace()).Get(getName(sa, ""), metav1.GetOptions{})
	}
	if err != nil {
		return nil, nil, err
	}

	klog.Infof("creating role binding %s", sa.Spec.WorkflowID)
	binding, err = kc.RbacV1().RoleBindings(sa.GetNamespace()).Create(binding)
	if errors.IsAlreadyExists(err) {
		binding, err = kc.RbacV1().RoleBindings(sa.GetNamespace()).Get(getName(sa, ""), metav1.GetOptions{})
	}
	if err != nil {
		return nil, nil, err
	}

	return role, binding, nil
}

func createMetadataAPIPod(kc kubernetes.Interface, image string, saccount *corev1.ServiceAccount,
	sa *nebulav1.SecretAuth, vaultAddr, vaultEngineMount string) (*corev1.Pod, error) {

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      metadataServiceName,
			Namespace: sa.GetNamespace(),
			Labels: getLabels(sa, map[string]string{
				"app": metadataServiceName,
			}),
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
						vaultAddr,
						"-vault-role",
						sa.GetNamespace(),
						"-workflow-id",
						sa.Spec.WorkflowID,
						"-vault-engine-mount",
						vaultEngineMount,
						"-namespace",
						sa.GetNamespace(),
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

	klog.Infof("creating metadata service pod %s", sa.Spec.WorkflowID)

	pod, err := kc.CoreV1().Pods(sa.GetNamespace()).Create(pod)
	if errors.IsAlreadyExists(err) {
		pod, err = kc.CoreV1().Pods(sa.GetNamespace()).Get(metadataServiceName, metav1.GetOptions{})
	}
	if err != nil {
		return nil, err
	}

	return pod, nil
}

func createMetadataAPIService(kc kubernetes.Interface, sa *nebulav1.SecretAuth) (*corev1.Service, error) {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      metadataServiceName,
			Namespace: sa.GetNamespace(),
			Labels: getLabels(sa, map[string]string{
				"app": metadataServiceName,
			}),
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port:       80,
					TargetPort: intstr.FromInt(7000),
				},
			},
			Selector: map[string]string{
				"app": metadataServiceName,
			},
		},
	}

	klog.Infof("creating pod service %s", sa.Spec.WorkflowID)

	service, err := kc.CoreV1().Services(sa.GetNamespace()).Create(service)
	if errors.IsAlreadyExists(err) {
		service, err = kc.CoreV1().Services(sa.GetNamespace()).Get(metadataServiceName, metav1.GetOptions{})
	}
	if err != nil {
		return nil, err
	}

	return service, nil
}

func createWorkflowConfigMap(kc kubernetes.Interface, service *corev1.Service, sa *nebulav1.SecretAuth) (*corev1.ConfigMap, error) {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getName(sa, ""),
			Namespace: sa.GetNamespace(),
			Labels:    getLabels(sa, nil),
		},
		Data: map[string]string{
			"metadata-api-url": fmt.Sprintf("http://%s.%s.svc.cluster.local", service.GetName(), sa.GetNamespace()),
		},
	}

	klog.Infof("creating config map %s", sa.Spec.WorkflowID)
	configMap, err := kc.CoreV1().ConfigMaps(sa.GetNamespace()).Create(configMap)
	if errors.IsAlreadyExists(err) {
		configMap, err = kc.CoreV1().ConfigMaps(sa.GetNamespace()).Get(getName(sa, ""), metav1.GetOptions{})
	}
	if err != nil {
		return nil, err
	}

	return configMap, nil
}

func getName(sa *nebulav1.SecretAuth, name string) string {
	prefix := "wr"

	if name == "" {
		return fmt.Sprintf("%s-%s", prefix, sa.Spec.WorkflowRunID)
	}

	return fmt.Sprintf("%s-%s-%s", prefix, sa.Spec.WorkflowRunID, name)
}

func getLabels(sa *nebulav1.SecretAuth, additional map[string]string) map[string]string {
	labels := map[string]string{
		"workflow-run-id": sa.Spec.WorkflowRunID,
		"workflow-id":     sa.Spec.WorkflowID,
	}

	if additional != nil {
		for k, v := range additional {
			labels[k] = v
		}
	}

	return labels
}
