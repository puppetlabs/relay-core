package cloud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/puppetlabs/nebula-tasks/hack/kops/pkg/apis/kops"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/provisioning/models"
)

const (
	DefaultNodeCount   = 2
	DefaultMasterCount = 1
)

type KopsSupporter interface {
	StateStoreURL(context.Context) (*url.URL, errors.Error)
	CredentialsFile(context.Context) (string, errors.Error)
	SSHPublicKey(context.Context) (string, errors.Error)
	EnvironmentVariables(context.Context) ([]string, errors.Error)
	PlatformID() string
}

// K8sClusterAdapter creates, updates or deletes clusters running inside cloud providers
type K8sClusterAdapter struct {
	spec    *models.K8sProvisionerSpec
	support KopsSupporter
	workdir string
}

func (k *K8sClusterAdapter) writeResource(name string, obj interface{}) errors.Error {
	f, err := os.Create(filepath.Join(k.workdir, name))
	if err != nil {
		return errors.NewK8sProvisionerIoError("failed to create resource file").WithCause(err)
	}

	if err := json.NewEncoder(f).Encode(obj); err != nil {
		return errors.NewK8sProvisionerIoError("failed to encode resource file").WithCause(err)
	}

	return nil
}

// ClusterCreate uses kops to create a kubernetes cluster
func (k *K8sClusterAdapter) ProvisionCluster(ctx context.Context) errors.Error {
	stateStoreURL, err := k.support.StateStoreURL(ctx)
	if err != nil {
		return errors.NewK8sProvisionerKopsExecError().WithCause(err)
	}

	vars, err := k.support.EnvironmentVariables(ctx)
	if err != nil {
		return errors.NewK8sProvisionerKopsExecError().WithCause(err)
	}

	var res *resources

	log.Println("checking if cluster exists")
	res, err = k.getCluster(ctx)
	if err != nil {
		if !errors.IsK8sProvisionerClusterNotFound(err) {
			return err
		}

		log.Println("creating initial cluster resources")
		res, err = k.generateInitialClusterResourcesHack(ctx)
		if err != nil {
			return err
		}
	}

	log.Println("setting cluster configuration")
	nodeCount := int32(k.spec.NodeCount)
	if nodeCount == 0 {
		nodeCount = DefaultNodeCount
	}

	masterCount := int32(k.spec.MasterCount)
	if masterCount == 0 {
		masterCount = DefaultMasterCount
	}

	if err := k.writeResource("cluster.json", res.Cluster); err != nil {
		return err
	}

	res.MasterInstanceGroup.Spec.MinSize = &masterCount
	res.MasterInstanceGroup.Spec.MaxSize = &masterCount
	res.MasterInstanceGroup.Spec.Zones = k.spec.Zones

	if err := k.writeResource("master-ig.json", res.MasterInstanceGroup); err != nil {
		return err
	}

	res.NodeInstanceGroup.Spec.MinSize = &nodeCount
	res.NodeInstanceGroup.Spec.MaxSize = &nodeCount
	res.NodeInstanceGroup.Spec.Zones = k.spec.Zones

	if err := k.writeResource("node-ig.json", res.NodeInstanceGroup); err != nil {
		return err
	}

	log.Println("synchronizing cluster state with cloud platform")
	kerr := kopsExec(ctx, command{
		args: []string{
			"--state", stateStoreURL.String(),
			"replace", "--force",
			"-f", filepath.Join(k.workdir, "cluster.json"),
			"-f", filepath.Join(k.workdir, "master-ig.json"),
			"-f", filepath.Join(k.workdir, "node-ig.json"),
		},
		env:    vars,
		stdout: os.Stdout,
		stderr: os.Stderr,
	})
	if kerr != nil {
		return errors.NewK8sProvisionerKopsExecError().WithCause(kerr)
	}

	p, err := k.support.SSHPublicKey(ctx)
	if err != nil {
		return errors.NewK8sProvisionerKopsExecError().WithCause(err)
	}

	if p != "" {
		kerr := kopsExec(ctx, command{
			args: []string{
				"--state", stateStoreURL.String(),
				"create", "secret", "sshpublickey", "admin", "-i", p,
				"--name", k.spec.ClusterName,
			},
			env:    vars,
			stdout: os.Stdout,
			stderr: os.Stderr,
		})
		if kerr != nil {
			return errors.NewK8sProvisionerKopsExecError().WithCause(kerr)
		}
	}

	kerr = kopsExec(ctx, command{
		args: []string{
			"--state", stateStoreURL.String(),
			"update", "cluster", k.spec.ClusterName,
			"--yes",
		},
		env:    vars,
		stdout: os.Stdout,
		stderr: os.Stderr,
	})
	if kerr != nil {
		return errors.NewK8sProvisionerKopsExecError().WithCause(kerr)
	}

	return nil
}

func (k *K8sClusterAdapter) GetKubeconfig(ctx context.Context) (io.Reader, errors.Error) {
	stateStoreURL, err := k.support.StateStoreURL(ctx)
	if err != nil {
		return nil, errors.NewK8sProvisionerKopsExecError().WithCause(err)
	}

	vars, err := k.support.EnvironmentVariables(ctx)
	if err != nil {
		return nil, errors.NewK8sProvisionerKopsExecError().WithCause(err)
	}

	kubeconfigPath := filepath.Join(k.workdir, "kubeconfig")

	// sets the location to store the kubeconfig file
	vars = append(vars, fmt.Sprintf("KUBECONFIG=%s", kubeconfigPath))

	kerr := kopsExec(ctx, command{
		args: []string{
			"--state", stateStoreURL.String(),
			"export", "kubecfg", k.spec.ClusterName,
		},
		env:    vars,
		stdout: os.Stdout,
		stderr: os.Stderr,
	})
	if kerr != nil {
		return nil, errors.NewK8sProvisionerKopsExecError().WithCause(err)
	}

	f, goerr := os.Open(kubeconfigPath)
	if goerr != nil {
		return nil, errors.NewK8sProvisionerKubeconfigReadError().WithCause(goerr)
	}

	defer f.Close()

	// here we are draining the file object (ReadCloser) into a standard byte array buffer
	// so we can just return a Reader (easier for the caller to use).
	buf := &bytes.Buffer{}
	if _, err := buf.ReadFrom(f); err != nil {
		return nil, errors.NewK8sProvisionerKubeconfigReadError().WithCause(err)
	}

	return buf, nil
}

func (k *K8sClusterAdapter) GetCluster(ctx context.Context) (*models.K8sCluster, errors.Error) {
	resources, err := k.getCluster(ctx)
	if err != nil {
		return nil, err
	}

	result, err := k.validateCluster(ctx)
	if err != nil {
		return nil, err
	}

	status := models.ClusterStatusNotReady

	if result.ready {
		status = models.ClusterStatusReady
	}

	var cluster = &models.K8sCluster{
		Name:   resources.Cluster.GetName(),
		Status: status,
	}

	return cluster, nil
}

func (k *K8sClusterAdapter) getCluster(ctx context.Context) (*resources, errors.Error) {
	stateStoreURL, err := k.support.StateStoreURL(ctx)
	if err != nil {
		return nil, errors.NewK8sProvisionerKopsExecError().WithCause(err)
	}

	vars, err := k.support.EnvironmentVariables(ctx)
	if err != nil {
		return nil, errors.NewK8sProvisionerKopsExecError().WithCause(err)
	}

	var (
		buf    = &bytes.Buffer{}
		errbuf = &bytes.Buffer{}
	)

	emw := io.MultiWriter(os.Stderr, errbuf)

	log.Println("looking for cluster and exporting kubeconfig")
	kerr := kopsExec(ctx, command{
		args: []string{
			"--state", stateStoreURL.String(),
			"get", k.spec.ClusterName,
			"--output", "json",
		},
		env:    vars,
		stdout: buf,
		stderr: emw,
	})

	if kerr != nil {
		if strings.Contains(errbuf.String(), "not found") {
			log.Println("no existing cluster was found")
			return nil, errors.NewK8sProvisionerClusterNotFound(k.spec.ClusterName)
		}

		return nil, errors.NewK8sProvisionerKopsExecError().WithCause(kerr)
	}

	var res resources

	if err := json.NewDecoder(buf).Decode(&res); err != nil {
		return nil, errors.NewK8sProvisionerKopsExecError().WithCause(err)
	}

	return &res, nil
}

// Alright, about this. Kops is great and all, but it's currently in a poor state api-wise.
// I wish I had known this when I was initally evaluating it, but these things have slowly crept up
// as I got deeper into the system. This method will do a dry-run of `kops create cluster` and output
// the result as json. This json is then unmarshaled into our resources composit type. This will contain
// kops.Cluster, kops.InstanceGroup for masters and kops.InstanceGroup for nodes that are
// pre-populated with defaults from create cluster command they use for their CLI. This allows us to swap
// out the nebula-configurable values we care about and then run `kops replace --force` on the resource file
// which will upsert the cluster. I initially started to pull in the kops code as a library, but that shit
// is absolutely broken. They don't use go modules for deps, which makes locking down the kubernetes api
// client version impossible, and they also don't commit their generated code, which means we would need
// ALL of their build tooling in order to generate the cloudup resources and the models. I tried for about
// half a day to shoehorn this into our codebase. I think the better option later on is to either fork kops
// and fix everything they did wrong, or write our own cloud cluster provisioner (probably this to support
// bare metal too).
//
// https://github.com/kubernetes/kops/issues/4288
//
// <3 Kyle Terry.
func (k K8sClusterAdapter) generateInitialClusterResourcesHack(ctx context.Context) (*resources, errors.Error) {
	stateStoreURL, err := k.support.StateStoreURL(ctx)
	if err != nil {
		return nil, errors.NewK8sProvisionerKopsExecError().WithCause(err)
	}

	vars, err := k.support.EnvironmentVariables(ctx)
	if err != nil {
		return nil, errors.NewK8sProvisionerKopsExecError().WithCause(err)
	}

	var buf = &bytes.Buffer{}

	log.Println("initializing provisioner")
	kerr := kopsExec(ctx, command{
		args: []string{
			"--state", stateStoreURL.String(),
			"create", "cluster", "--dry-run",
			"--output", "json",
			"--name", k.spec.ClusterName,
			"--zones", strings.Join(k.spec.Zones, ","),
			"--project", k.spec.Project,
			"--cloud", k.support.PlatformID(),
		},
		env:    vars,
		stdout: buf,
		stderr: os.Stderr,
	})
	if kerr != nil {
		return nil, errors.NewK8sProvisionerKopsExecError().WithCause(kerr)
	}

	var res resources

	if err := json.NewDecoder(buf).Decode(&res); err != nil {
		return nil, errors.NewK8sProvisionerKopsExecError().WithCause(err)
	}

	return &res, nil
}

func (k K8sClusterAdapter) validateCluster(ctx context.Context) (*validationResult, errors.Error) {
	stateStoreURL, err := k.support.StateStoreURL(ctx)
	if err != nil {
		return nil, errors.NewK8sProvisionerKopsExecError().WithCause(err)
	}

	vars, err := k.support.EnvironmentVariables(ctx)
	if err != nil {
		return nil, errors.NewK8sProvisionerKopsExecError().WithCause(err)
	}

	var buf = &bytes.Buffer{}

	log.Println("validating cluster")
	kerr := kopsExec(ctx, command{
		args: []string{
			"--state", stateStoreURL.String(),
			"validate", "cluster",
		},
		env:    vars,
		stdout: buf,
		stderr: os.Stderr,
	})
	if kerr != nil {
		return nil, errors.NewK8sProvisionerKopsExecError().WithCause(kerr)
	}

	res := validationResult{
		ready: strings.Contains(buf.String(), "is ready"),
	}

	return &res, nil
}

// NewK8sClusterAdapter returns a new K8sClusterAdapter.
func NewK8sClusterAdapter(spec *models.K8sProvisionerSpec, ks KopsSupporter, workdir string) *K8sClusterAdapter {
	return &K8sClusterAdapter{
		spec:    spec,
		support: ks,
		workdir: workdir,
	}
}

// kind is used to extract only the object kind field from a json.RawMessage object.
type kind struct {
	Kind string `json:"kind"`
}

// resources is a type that holds all 3 objects required for a cluster creation.
type resources struct {
	Cluster             *kops.Cluster
	MasterInstanceGroup *kops.InstanceGroup
	NodeInstanceGroup   *kops.InstanceGroup
}

// UnmarshalJSON takes a list of mixed-type kubernetes objects, extracts the Kind
// field and unmarshals the RawMessage into the correct type set on resources.
func (r *resources) UnmarshalJSON(b []byte) error {
	items := []json.RawMessage{}

	if err := json.Unmarshal(b, &items); err != nil {
		return err
	}

	for _, item := range items {
		k := kind{}
		if err := json.Unmarshal(item, &k); err != nil {
			return err
		}

		switch k.Kind {
		case "Cluster":
			cluster := kops.Cluster{}
			if err := json.Unmarshal(item, &cluster); err != nil {
				return err
			}

			r.Cluster = &cluster
		case "InstanceGroup":
			ig := kops.InstanceGroup{}
			if err := json.Unmarshal(item, &ig); err != nil {
				return err
			}

			if ig.Spec.Role == kops.InstanceGroupRoleMaster {
				r.MasterInstanceGroup = &ig
			} else if ig.Spec.Role == kops.InstanceGroupRoleNode {
				r.NodeInstanceGroup = &ig
			}
		}
	}

	return nil
}

type validationResult struct {
	ready bool
}

type command struct {
	stdout, stderr io.Writer
	args           []string
	env            []string
}

func kopsExec(ctx context.Context, c command) error {
	kopsCmd := "kops"
	cmd := exec.CommandContext(ctx, kopsCmd, c.args...)

	cmd.Env = append(os.Environ(), c.env...)
	cmd.Stdout = c.stdout
	cmd.Stderr = c.stderr

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}
