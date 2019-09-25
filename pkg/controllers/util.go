package controllers

import "io/ioutil"

const DefaultPodNamespacePath = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

// LookupNamespace attempts to read the namespace file from the pod's mounted
// service account. If an override string is not empty, then that is returned instead
// allowing a command line flag to override the working namespace to set in controllers.
// This function can return an error if the process is not being ran inside a pod, an indicator
// that an override should be used.
func LookupNamespace(override string) (string, error) {
	if override != "" {
		return override, nil
	}

	b, err := ioutil.ReadFile(DefaultPodNamespacePath)
	if err != nil {
		return "", err
	}

	return string(b), nil
}
