package taskutil

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/mitchellh/go-homedir"
)

func run(cmd string, args ...string) error {
	c := exec.Command(cmd, args...)
	var output bytes.Buffer
	c.Stderr = &output
	c.Stdout = &output
	if err := c.Run(); err != nil {
		return fmt.Errorf("git: %+v: %s", err, output.String())
	}
	return nil
}

// Fetch fetches the specified git repository at the revision into path.
func Fetch(revision, path, url string) error {
	// HACK: This is to get git+ssh to work since ssh doesn't respect the HOME
	// env variable.
	homepath, err := homedir.Dir()
	if err != nil {
		return err
	}
	homeenv := os.Getenv("HOME")
	euid := os.Geteuid()
	// Special case the root user/directory
	if euid == 0 {
		if err := os.Symlink(homeenv+"/.ssh", "/root/.ssh"); err != nil {
		}
	} else if homeenv != "" && homeenv != homepath {
		if _, err := os.Stat(homepath + "/.ssh"); os.IsNotExist(err) {
			if err := os.Symlink(homeenv+"/.ssh", homepath+"/.ssh"); err != nil {
			}
		}
	}

	if revision == "" {
		revision = "master"
	}
	if path != "" {
		if err := run("git", "init", path); err != nil {
			return err
		}
		if err := os.Chdir(path); err != nil {
			return nil
		}
	} else {
		if err := run("git", "init"); err != nil {
			return err
		}
	}

	trimmedURL := strings.TrimSpace(url)
	if err := run("git", "remote", "add", "origin", trimmedURL); err != nil {
		return err
	}
	if err := run("git", "fetch", "--depth=1", "--recurse-submodules=yes", "origin", revision); err != nil {
		// Fetch can fail if an old commitid was used so try git pull, performing regardless of error
		// as no guarantee that the same error is returned by all git servers gitlab, github etc...
		if err := run("git", "pull", "--recurse-submodules=yes", "origin", revision); err != nil {
			return err
		}
		if err := run("git", "checkout", revision); err != nil {
			return err
		}
	} else {
		if err := run("git", "reset", "--hard", "FETCH_HEAD"); err != nil {
			return err
		}
	}
	return nil
}
