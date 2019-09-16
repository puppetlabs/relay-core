package task

import (
	"path/filepath"

	"github.com/puppetlabs/nebula-tasks/pkg/model"
	"github.com/puppetlabs/nebula-tasks/pkg/taskutil"
)

func (ti *TaskInterface) ProcessCredentials(directory string) error {
	var spec model.CredentialSpec
	if err := taskutil.PopulateSpecFromDefaultPlan(&spec, ti.opts); err != nil {
		return err
	}

	if directory == "" {
		directory = DefaultPath
	}

	for name, credentials := range spec.Credentials {
		destination := filepath.Join(directory, name)
		err := taskutil.WriteToFile(destination, credentials)
		if err != nil {
			return err
		}
	}

	return nil
}
