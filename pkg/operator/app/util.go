package app

import (
	"crypto/sha256"
	"encoding/base32"
	"strings"

	"github.com/puppetlabs/leg/k8sutil/pkg/norm"
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/obj"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// XXX MAKE SURE THIS GETS APPLIED
	ManagedByLabelValue = "relay.sh"
)

func ModelStepFromName(wr *obj.WorkflowRun, stepName string) *model.Step {
	return &model.Step{
		Run:  model.Run{ID: wr.Object.Spec.Name},
		Name: stepName,
	}
}

func ModelStep(wr *obj.WorkflowRun, step *relayv1beta1.Step) *model.Step {
	return ModelStepFromName(wr, step.Name)
}

func ModelWebhookTrigger(wt *obj.WebhookTrigger) *model.Trigger {
	name := wt.Object.Spec.Name
	if name == "" {
		name = wt.Key.Name
	}

	return &model.Trigger{
		Name: name,
	}
}

func SuffixObjectKey(key client.ObjectKey, suffix string) client.ObjectKey {
	return client.ObjectKey{
		Namespace: key.Namespace,
		Name:      norm.MetaNameSuffixed(key.Name, "-"+suffix),
	}
}

func SuffixObjectKeyWithHashOfObjectKey(key, hashable client.ObjectKey) client.ObjectKey {
	hsh := sha256.Sum256([]byte(hashable.String()))
	return SuffixObjectKey(key, strings.ToLower(base32.HexEncoding.WithPadding(base32.NoPadding).EncodeToString(hsh[:12])))
}

func ModelStepObjectKey(key client.ObjectKey, ms *model.Step) client.ObjectKey {
	return client.ObjectKey{
		Namespace: key.Namespace,
		Name:      norm.MetaNameSuffixed(key.Name+"-"+ms.Name, "-"+ms.Hash().HexEncoding()),
	}
}
