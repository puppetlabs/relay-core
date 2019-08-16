package task

import (
	"encoding/base64"
	"errors"
	"path/filepath"
	"regexp"

	"github.com/puppetlabs/nebula-tasks/pkg/model"
	"github.com/puppetlabs/nebula-tasks/pkg/taskutil"
)

var gitSSHUrl = regexp.MustCompile(`^git@([a-zA-Z0-9\-\.]+):(.+)/(.+)\.git$`)

func (ti *TaskInterface) CloneRepository(revision string, directory string) error {
	var spec model.GitSpec
	if err := taskutil.PopulateSpecFromDefaultPlan(&spec, ti.opts); err != nil {
		return err
	}

	resource := spec.GitRepository
	if resource == nil {
		return nil
	}

	if resource.Name == "" {
		resource.Name = DefaultName
	}

	if revision == "" {
		revision = DefaultRevision
	}

	if directory == "" {
		directory = DefaultPath
	}

	if resource.SSHKey != "" {
		err := writeSSHConfig(resource)
		if err != nil {
			return err
		}
	}

	err := taskutil.Fetch(revision, filepath.Join(directory, resource.Name), resource.Repository)
	if err != nil {
		return err
	}

	return nil
}

func writeSSHConfig(resource *model.GitDetails) error {
	gitConfig := taskutil.SSHConfig{}

	matches := gitSSHUrl.FindStringSubmatch(resource.Repository)
	if len(matches) <= 1 {
		return errors.New("SSH URL is malformed")
	}

	host := matches[1]

	gitConfig.Order = make([]string, 0)
	gitConfig.Order = append(gitConfig.Order, host)
	gitConfig.Entries = make(map[string]taskutil.SSHEntry, 0)

	sshKey, err := base64.StdEncoding.DecodeString(resource.SSHKey)
	if err != nil {
		return err
	}

	knownHosts, err := base64.StdEncoding.DecodeString(resource.KnownHosts)
	if err != nil {
		return err
	}

	if len(knownHosts) == 0 || err != nil {
		knownHosts, err = taskutil.SSHKeyScan(host)
		if err != nil {
			return err
		}
	}

	gitConfig.Entries[host] = taskutil.SSHEntry{
		Name:       resource.Name,
		PrivateKey: string(sshKey),
		KnownHosts: string(knownHosts),
	}

	return gitConfig.Write()
}
