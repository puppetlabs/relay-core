package image

import (
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

func ImageData(image string) ([]string, []string, error) {
	ref, err := name.ParseReference(image, name.WeakValidation)
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
