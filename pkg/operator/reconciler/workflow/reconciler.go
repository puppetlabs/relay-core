package workflow

import (
	"context"
	"fmt"
	"io"

	"github.com/puppetlabs/leg/errmap/pkg/errmap"
	"github.com/puppetlabs/leg/errmap/pkg/errmark"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/errhandler"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	"github.com/puppetlabs/leg/storage"
	pvpoolv1alpha1 "github.com/puppetlabs/pvpool/pkg/apis/pvpool.puppet.com/v1alpha1"
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/authenticate"
	"github.com/puppetlabs/relay-core/pkg/obj"
	"github.com/puppetlabs/relay-core/pkg/operator/app"
	"github.com/puppetlabs/relay-core/pkg/operator/dependency"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/klog/v2"
	"knative.dev/pkg/apis"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const FinalizerName = "workflowrun.finalizers.controller.relay.sh"

type Reconciler struct {
	*dependency.DependencyManager

	Client client.Client
	Scheme *runtime.Scheme

	metrics *controllerObservations
	issuer  authenticate.Issuer
}

func NewReconciler(dm *dependency.DependencyManager) *Reconciler {
	return &Reconciler{
		DependencyManager: dm,

		Client: dm.Manager.GetClient(),
		Scheme: dm.Manager.GetScheme(),

		metrics: newControllerObservations(dm.Metrics),
		issuer: authenticate.IssuerFunc(func(ctx context.Context, claims *authenticate.Claims) (authenticate.Raw, error) {
			raw, err := authenticate.NewKeySignerIssuer(dm.JWTSigner).Issue(ctx, claims)
			if err != nil {
				return nil, err
			}

			return authenticate.NewVaultTransitWrapper(
				dm.VaultClient,
				dm.Config.VaultTransitPath,
				dm.Config.VaultTransitKey,
				authenticate.VaultTransitWrapperWithContext(authenticate.VaultTransitNamespaceContext(claims.KubernetesNamespaceUID)),
			).Wrap(ctx, raw)
		}),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	wr := obj.NewWorkflowRun(req.NamespacedName)
	if ok, err := wr.Load(ctx, r.Client); err != nil {
		return ctrl.Result{}, errmap.Wrap(err, "failed to load dependencies")
	} else if !ok {
		// CRD deleted from under us?
		return ctrl.Result{}, nil
	}

	var wrd *app.WorkflowRunDeps
	var pr *obj.PipelineRun
	err = r.metrics.trackDurationWithOutcome(metricWorkflowRunStartUpDuration, func() error {
		wrd, err = app.ApplyWorkflowRunDeps(
			ctx,
			r.Client,
			wr,
			r.issuer,
			r.Config.MetadataAPIURL,
			app.WorkflowRunDepsWithStandaloneMode(r.Config.Standalone),
			app.WorkflowRunDepsWithToolInjectionPool(pvpoolv1alpha1.PoolReference{
				Namespace: r.Config.WorkflowToolInjectionPool.Namespace,
				Name:      r.Config.WorkflowToolInjectionPool.Name,
			}),
		)

		if err != nil {
			err = errmark.MarkTransientIf(err, errhandler.RuleIsRequired)

			return errmap.Wrap(err, "failed to apply dependencies")
		}

		if len(wrd.Workflow.Object.Spec.Steps) == 0 {
			return nil
		}

		// We only need to build the pipeline when the workflow run is
		// initializing or finalizing. While it's running, constantly persisting
		// pipeline objects just puts strain on the Tekton webhook server.
		//
		// An exception is made of we get out of sync somehow such that the
		// PipelineRun can't load (e.g. someone deletes it from under us).
		switch {
		case IsCondition(wr, relayv1beta1.RunCompleted, corev1.ConditionFalse) && !wr.IsCancelled():
			pr = obj.NewPipelineRun(
				client.ObjectKey{
					Namespace: wrd.WorkflowDeps.TenantDeps.Namespace.Name,
					Name:      wr.Key.Name,
				},
			)
			if ok, err := pr.Load(ctx, r.Client); err != nil {
				return errmap.Wrap(err, "failed to load PipelineRun")
			} else if ok {
				break
			}
			fallthrough
		default:
			pipeline, err := app.ApplyPipelineParts(ctx, r.Client, wrd)
			if err != nil {
				return errmap.Wrap(err, "failed to apply Pipeline")
			}

			pr, err = app.ApplyPipelineRun(ctx, r.Client, pipeline)
			if err != nil {
				return errmap.Wrap(err, "failed to apply PipelineRun")
			}
		}

		return nil
	})
	if err != nil {
		klog.Error(err)
		retryOnError := true
		errmark.IfMarked(err, errmark.User, func(err error) {
			app.ConfigureWorkflowRunWithSpecificStatus(wrd.WorkflowRun, relayv1beta1.RunSucceeded, corev1.ConditionFalse)

			if ferr := wr.PersistStatus(ctx, r.Client); ferr != nil {
				return
			}

			retryOnError = false
		})

		if retryOnError {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	finalized, err := lifecycle.Finalize(ctx, r.Client, FinalizerName, wr, func() error {
		_, err := wrd.Delete(ctx, r.Client)
		return err
	})
	if err != nil || finalized {
		return ctrl.Result{}, err
	}

	if pr == nil {
		app.ConfigureWorkflowRunWithSpecificStatus(wrd.WorkflowRun, relayv1beta1.RunSucceeded, corev1.ConditionTrue)

		if err := wr.PersistStatus(ctx, r.Client); err != nil {
			return ctrl.Result{}, errmap.Wrap(err, "failed to persist WorkflowRun")
		}

		return ctrl.Result{}, nil
	}

	err = r.metrics.trackDurationWithOutcome(metricWorkflowRunLogUploadDuration, func() error {
		r.uploadLogs(ctx, wr, pr)
		return nil
	})
	if err != nil {
		klog.Warning(err)
	}

	app.ConfigureWorkflowRun(ctx, wrd, pr)

	if err := wr.PersistStatus(ctx, r.Client); err != nil {
		return ctrl.Result{}, errmap.Wrap(err, "failed to persist WorkflowRun")
	}

	return ctrl.Result{}, nil
}

func (r *Reconciler) uploadLogs(ctx context.Context, wr *obj.WorkflowRun, plr *obj.PipelineRun) {
	podNames := make(map[string]string)

	for name, tr := range plr.Object.Status.TaskRuns {
		if tr.Status == nil {
			continue
		}

		// TODO: This should support retries, possibly to different log
		// endpoints?
		if cond := tr.Status.GetCondition(apis.ConditionSucceeded); cond == nil || cond.IsUnknown() {
			continue
		}

		podNames[name] = tr.Status.PodName
	}

	for name, step := range wr.Object.Status.Steps {
		// FIXME Temporary handling for legacy logs
		if len(step.Logs) > 0 &&
			step.Logs[0] != nil && step.Logs[0].Context != "" {
			// Already uploaded.
			continue
		}

		podName, found := podNames[step.Name]
		if !found {
			// Not done yet.
			klog.Infof("WorkflowRun %s step %q is still progressing, waiting to upload logs", wr.Key, name)
			continue
		}

		klog.Infof("WorkflowRun %s step %q is complete, uploading logs for pod %s", wr.Key, name, podName)

		logKey, err := r.uploadLog(ctx, plr.Key.Namespace, podName, "step-step")
		if err != nil {
			klog.Warningf("failed to upload log for WorkflowRun %s step %q: %+v", wr.Key, name, err)
		}

		// FIXME Temporary handling for legacy logs
		step.Logs = []*relayv1beta1.Log{
			{
				Name:    podName,
				Context: logKey,
			},
		}

		wr.Object.Status.Steps[name] = step
	}
}

func (r *Reconciler) uploadLog(ctx context.Context, namespace, podName, containerName string) (string, error) {
	key := fmt.Sprintf("%s/%s/%s", namespace, podName, containerName)

	// XXX: We can't do this with the dynamic client yet.
	opts := &corev1.PodLogOptions{
		Container: containerName,
	}
	rc, err := r.KubeClient.CoreV1().Pods(namespace).GetLogs(podName, opts).Stream(ctx)
	if err != nil {
		return "", err
	}
	defer rc.Close()

	storageOpts := storage.PutOptions{
		ContentType: "application/octet-stream",
	}

	err = r.StorageClient.Put(ctx, key, func(w io.Writer) error {
		_, err := io.Copy(w, rc)

		return err
	}, storageOpts)
	if err != nil {
		return "", err
	}

	return key, nil
}

// TODO: Where does this method really belong?
func IsCondition(wr *obj.WorkflowRun, rc relayv1beta1.RunConditionType, status corev1.ConditionStatus) bool {
	for _, c := range wr.Object.Status.Conditions {
		if c.Type == rc && c.Status == status {
			return true
		}
	}

	return false
}
