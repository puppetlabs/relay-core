package task

import (
	"io"
	"os"
	"path/filepath"

	"github.com/puppetlabs/nebula-tasks/pkg/model"
	"github.com/puppetlabs/nebula-tasks/pkg/taskutil"
	"gopkg.in/ini.v1"
)

func (ti *TaskInterface) ProcessAWS(directory string) error {
	var spec model.AWSSpec
	if err := taskutil.PopulateSpecFromDefaultPlan(&spec, ti.opts); err != nil {
		return err
	}

	if spec.AWS == nil {
		return nil
	}

	if directory == "" {
		directory = filepath.Join(DefaultPath, ".aws")
	}

	// .aws/credentials
	creds := ini.Empty()
	creds.Section("default").Key("aws_access_key_id").SetValue(spec.AWS.AccessKeyID)
	creds.Section("default").Key("aws_secret_access_key").SetValue(spec.AWS.SecretAccessKey)

	// .aws/config
	config := ini.Empty()
	if spec.AWS.Region != "" {
		config.Section("default").Key("region").SetValue(spec.AWS.Region)
	}

	if err := os.MkdirAll(directory, 0755); err != nil {
		return err
	}

	fs := []struct {
		Path string
		Perm os.FileMode
		Data io.WriterTo
	}{
		{filepath.Join(directory, "credentials"), 0600, creds},
		{filepath.Join(directory, "config"), 0644, config},
	}
	for _, f := range fs {
		fp, err := os.OpenFile(f.Path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Perm)
		if err != nil {
			return err
		}

		_, err = f.Data.WriteTo(fp)

		if cerr := fp.Close(); err == nil {
			err = cerr
		}

		if err != nil {
			return err
		}
	}

	return nil
}
