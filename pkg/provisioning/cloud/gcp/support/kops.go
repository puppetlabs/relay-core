package support

import (
	"context"
	"io/ioutil"
	"net/url"
	"path"

	gcs "cloud.google.com/go/storage"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/taskutil"
	"google.golang.org/api/googleapi"
)

// KopsSupport provides functionality that supports the running of kops commands.
// This includes creating and validating gcp cloud resources needed to successfully
// run the provisioner.
type KopsSupport struct {
	CredentialsFile string
	storeName       string
	projectID       string
	gclient         *gcs.Client
}

// StateStore checks if the bucket exists and creates it if it doesn't.
// Then it will return a url to the storage bucket suitable for kops.
func (ki *KopsSupport) StateStoreURL(ctx context.Context) (*url.URL, errors.Error) {
	err := ki.gclient.Bucket(ki.storeName).Create(ctx, ki.projectID, nil)
	if err != nil {
		gerr := err.(*googleapi.Error)
		if gerr.Code != 409 {
			return nil, errors.NewK8sProvisionerStorageCreationFailed().WithCause(err)
		}
	}

	return &url.URL{Host: ki.storeName, Scheme: "gs"}, nil
}

func NewKopsSupport(projectID, serviceAccountContent, storeName string) (*KopsSupport, errors.Error) {
	if storeName == "" {
		return nil, errors.NewK8sProvisionerSupportValidationError("stateStore is empty")
	}

	ctx := context.Background()

	gclient, err := gcs.NewClient(ctx)
	if err != nil {
		return nil, errors.NewK8sProvisionerClientSetupError().WithCause(err)
	}

	p, err := writeCredentialsFile(serviceAccountContent)
	if err != nil {
		return nil, errors.NewK8sProvisionerClientSetupError().WithCause(err)
	}

	return &KopsSupport{
		CredentialsFile: p,
		storeName:       storeName,
		projectID:       projectID,
		gclient:         gclient,
	}, nil
}

func writeCredentialsFile(serviceAccountContent string) (string, errors.Error) {
	d, err := ioutil.TempDir("", "nebula-kops-support")
	if err != nil {
		return "", errors.NewK8sProvisionerCredentialsFileError().WithCause(err)
	}

	p := path.Join(d, "gcloud-service-account.json")

	if err := taskutil.WriteToFile(p, serviceAccountContent); err != nil {
		return "", errors.NewK8sProvisionerCredentialsFileError().WithCause(err)
	}

	return p, nil
}
