package image

import (
	"fmt"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

func ValidateImage(image string) (string, error) {
	ref, err := RepoReference(image)
	if err != nil {
		return "", err
	}

	if _, ok := ref.(name.Digest); ok {
		// We don't support digests as we have no way to pull them.
		return "", err
	}

	img, err := remote.Image(ref)
	if err != nil {
		return "", err
	}

	manifest, err := img.Manifest()
	if err != nil {
		return "", err
	}

	if len(manifest.Layers) == 0 {
		return "", err
	}

	digest, err := img.Digest()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s@%s", ref.Context(), digest.String()), nil
}

func ImageData(image string) ([]string, []string, error) {
	ref, err := RepoReference(image)
	if err != nil {
		return nil, nil, err
	}

	img, err := remote.Image(ref)
	if err != nil {
		return nil, nil, err
	}

	cfg, err := img.ConfigFile()
	if err != nil {
		return nil, nil, err
	}

	return cfg.Config.Entrypoint, cfg.Config.Cmd, nil
}

func RepoReference(image string) (name.Reference, error) {
	ref, err := name.ParseReference(image, name.WeakValidation)
	if err != nil {
		return nil, err
	}

	return ref, nil
}
