package app

import (
	"crypto/sha256"
	"encoding/base32"
	"strings"

	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/helper"
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

func ModelStepFromName(r *obj.Run, stepName string) *model.Step {
	return &model.Step{
		Run:  model.Run{ID: r.Object.GetName()},
		Name: stepName,
	}
}

func ModelStep(r *obj.Run, step *relayv1beta1.Step) *model.Step {
	return ModelStepFromName(r, step.Name)
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

func SuffixObjectKeyWithHashOfObjectKey(key, hashable client.ObjectKey) client.ObjectKey {
	hsh := sha256.Sum256([]byte(hashable.String()))
	return helper.SuffixObjectKey(key, strings.ToLower(base32.HexEncoding.WithPadding(base32.NoPadding).EncodeToString(hsh[:12])))
}

func ModelStepObjectKey(key client.ObjectKey, ms *model.Step) client.ObjectKey {
	return client.ObjectKey{
		Namespace: key.Namespace,
		Name:      norm.MetaNameSuffixed(key.Name+"-"+ms.Name, "-"+ms.Hash().HexEncoding()),
	}
}
