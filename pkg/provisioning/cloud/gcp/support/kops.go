package support

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"path"
	"sync"

	gcs "cloud.google.com/go/storage"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"

	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/taskutil"
)

type KopsSupportConfig struct {
	ProjectID                string
	ServiceAccountFileBase64 string
	StateStoreName           string
	WorkDir                  string
}

// KopsSupport provides functionality that supports the running of kops commands.
// This includes creating and validating gcp cloud resources needed to successfully
// run the provisioner.
type KopsSupport struct {
	cfg             KopsSupportConfig
	credentialsFile string
	stateStoreURL   *url.URL
	gclient         *gcs.Client

	setupOnce sync.Once
}

func (ki *KopsSupport) setup(ctx context.Context) errors.Error {
	var err errors.Error

	ki.setupOnce.Do(func() {
		log.Println("writing platform credentials to temporary location")
		ki.credentialsFile, err = ki.writeCredentialsFile(ki.cfg.ServiceAccountFileBase64)
		if err != nil {
			err = errors.NewK8sProvisionerKopsSupportSetupError().WithCause(err)

			return
		}

		log.Println("configuring access to google cloud")
		var gerr error
		ki.gclient, gerr = gcs.NewClient(ctx, option.WithCredentialsFile(ki.credentialsFile))
		if gerr != nil {
			err = errors.NewK8sProvisionerKopsSupportSetupError().WithCause(gerr)

			return
		}

		log.Println("ensuring GCS state storage exists")
		err := ki.gclient.Bucket(ki.cfg.StateStoreName).Create(ctx, ki.cfg.ProjectID, nil)
		if err != nil {
			gerr := err.(*googleapi.Error)
			if gerr.Code != 409 {
				err = errors.NewK8sProvisionerKopsStateStoreCreateFailed().WithCause(err)
			}

			log.Println("an existing GCS bucket was found")
		} else {
			log.Println("created GCS bucket")
		}

		ki.stateStoreURL = &url.URL{Host: ki.cfg.StateStoreName, Scheme: "gs"}
	})

	return err

}

// StateStore checks if the bucket exists and creates it if it doesn't.
// Then it will return a url to the storage bucket suitable for kops.
func (ki *KopsSupport) StateStoreURL(ctx context.Context) (*url.URL, errors.Error) {
	if err := ki.setup(ctx); err != nil {
		return nil, err
	}

	return ki.stateStoreURL, nil
}

// CredentialsFile returns the path to the gcp service account file that is used to authenticate
// with google cloud apis
func (ki *KopsSupport) CredentialsFile(ctx context.Context) (string, errors.Error) {
	if err := ki.setup(ctx); err != nil {
		return "", err
	}

	return ki.credentialsFile, nil
}

func (ki *KopsSupport) SSHPublicKey(ctx context.Context) (string, errors.Error) {
	if err := ki.setup(ctx); err != nil {
		return "", err
	}

	return "", nil
}

func (ki *KopsSupport) EnvironmentVariables(ctx context.Context) ([]string, errors.Error) {
	if err := ki.setup(ctx); err != nil {
		return nil, err
	}

	stateStoreURL, _ := ki.StateStoreURL(ctx)

	return []string{
		"KOPS_FEATURE_FLAGS=AlphaAllowGCE",
		fmt.Sprintf("KOPS_STATE_STORE=%s", stateStoreURL.String()),
		fmt.Sprintf("GOOGLE_APPLICATION_CREDENTIALS=%s", ki.credentialsFile),
	}, nil
}

func (ki *KopsSupport) PlatformID() string {
	return "gce"
}

func (ki *KopsSupport) writeCredentialsFile(serviceAccountContent string) (string, errors.Error) {
	p := path.Join(ki.cfg.WorkDir, "gcloud-service-account.json")

	if err := taskutil.WriteToFile(p, serviceAccountContent); err != nil {
		return "", errors.NewK8sProvisionerCredentialsFileError().WithCause(err)
	}

	return p, nil
}

func NewKopsSupport(cfg KopsSupportConfig) *KopsSupport {
	return &KopsSupport{
		cfg: cfg,
	}
}
