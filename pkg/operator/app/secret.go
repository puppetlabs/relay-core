package obj

import (
	"context"
	"errors"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ConfigureImagePullSecret(target, src *ImagePullSecret) {
	target.Object.Data = src.Object.DeepCopy().Data
}
