package taskutil

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type SSHConfig struct {
	Entries map[string]SSHEntry
	Order   []string
}

func (dc *SSHConfig) String() string {
	if dc == nil {
		return ""
	}
	var urls []string
	for _, k := range dc.Order {
		v := dc.Entries[k]
		urls = append(urls, fmt.Sprintf("%s=%s", v.Name, k))
	}
	return strings.Join(urls, ",")
}

func (dc *SSHConfig) Write() error {
	sshDir := filepath.Join(os.Getenv("HOME"), ".ssh")
	if err := os.MkdirAll(sshDir, os.ModePerm); err != nil {
		return err
	}
	var configEntries []string
	var defaultPort = "22"
	var knownHosts []string
	for _, k := range dc.Order {
		var host, port string
		var err error
		if host, port, err = net.SplitHostPort(k); err != nil {
			host = k
			port = defaultPort
		}
		v := dc.Entries[k]
		if err := v.Write(sshDir); err != nil {
			return err
		}
		configEntries = append(configEntries, fmt.Sprintf(`Host %s
    HostName %s
    IdentityFile %s
    Port %s
`, host, host, v.path(sshDir), port))

		knownHosts = append(knownHosts, v.KnownHosts)
	}
	configPath := filepath.Join(sshDir, "config")
	configContent := strings.Join(configEntries, "")
	if err := ioutil.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		return err
	}
	knownHostsPath := filepath.Join(sshDir, "known_hosts")
	knownHostsContent := strings.Join(knownHosts, "\n")
	return ioutil.WriteFile(knownHostsPath, []byte(knownHostsContent), 0600)
}

type SSHEntry struct {
	Name       string
	PrivateKey string
	KnownHosts string
}

func (be *SSHEntry) path(sshDir string) string {
	return filepath.Join(sshDir, "id_"+be.Name)
}

func (be *SSHEntry) Write(sshDir string) error {
	return ioutil.WriteFile(be.path(sshDir), []byte(be.PrivateKey), 0600)
}

func SSHKeyScan(domain string) ([]byte, error) {
	c := exec.Command("ssh-keyscan", domain)
	var output bytes.Buffer
	c.Stdout = &output
	c.Stderr = &output
	if err := c.Run(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}
