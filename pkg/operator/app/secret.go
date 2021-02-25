package obj

func ConfigureImagePullSecret(target, src *ImagePullSecret) {
	target.Object.Data = src.Object.DeepCopy().Data
}
