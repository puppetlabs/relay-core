package model

type CredentialSpec struct {
	Credentials map[string]string `json:"credentials"`
}

type GitSpec struct {
	GitRepository *GitDetails `json:"git"`
}

type GitDetails struct {
	Name       string `json:"name"`
	Repository string `json:"repository"`
	Branch     string `json:"branch"`
	SSHKey     string `json:"ssh_key"`
	KnownHosts string `json:"known_hosts"`
}

type ClusterSpec struct {
	Cluster *ClusterDetails `json:"cluster"`
}

type ClusterDetails struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	CAData   string `json:"cadata"`
	Token    string `json:"token"`
	Insecure bool   `json:"insecure"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type AWSSpec struct {
	AWS *AWSDetails `json:"aws"`
}

type AWSDetails struct {
	AccessKeyID     string `json:"accessKeyID"`
	SecretAccessKey string `json:"secretAccessKey"`
	Region          string `json:"region"`
}
