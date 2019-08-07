package support

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"path"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
)

type KopsSupportConfig struct {
	AccessKeyID     string
	SecretAccessKey string
	StateStoreName  string
	Region          string
	SSHPublicKey    string
}

// KopsSupport provides functionality that supports the running of kops commands.
// This includes creating and validating aws cloud resources needed to successfully
// run the provisioner.
type KopsSupport struct {
	cfg              KopsSupportConfig
	stateStoreURL    *url.URL
	sshPublicKeyPath string
	s3client         *s3.S3

	setupOnce sync.Once
}

func (ki *KopsSupport) setup(ctx context.Context) errors.Error {
	var err errors.Error

	ki.setupOnce.Do(func() {
		log.Println("configuring access to aws")
		sess, goerr := session.NewSession(&aws.Config{
			Region:      aws.String(ki.cfg.Region),
			Credentials: credentials.NewStaticCredentials(ki.cfg.AccessKeyID, ki.cfg.SecretAccessKey, ""),
		})
		if goerr != nil {
			err = errors.NewK8sProvisionerKopsSupportSetupError().WithCause(goerr)

			return
		}

		ki.s3client = s3.New(sess)

		log.Println("ensuring s3 state storage exists")
		_, goerr = ki.s3client.CreateBucketWithContext(ctx, &s3.CreateBucketInput{
			Bucket: aws.String(ki.cfg.StateStoreName),
		})
		if goerr != nil {
			aerr := goerr.(awserr.Error)
			if aerr.Code() != s3.ErrCodeBucketAlreadyOwnedByYou {
				err = errors.NewK8sProvisionerKopsStateStoreCreateFailed().WithCause(goerr)

				return
			}

			log.Println("an existing s3 bucket was found")
		} else {
			log.Println("created s3 bucket")
		}

		goerr = ki.s3client.WaitUntilBucketExists(&s3.HeadBucketInput{
			Bucket: aws.String(ki.cfg.StateStoreName),
		})
		if goerr != nil {
			err = errors.NewK8sProvisionerKopsStateStoreCreateFailed().WithCause(goerr)

			return
		}

		ki.stateStoreURL = &url.URL{Host: ki.cfg.StateStoreName, Scheme: "s3"}

		log.Println("writing ssh public key")
		p, err := writeSSHPublicKeyFile(ki.cfg.SSHPublicKey)
		if err != nil {
			err = errors.NewK8sProvisionerKopsSupportSetupError().WithCause(err)

			return
		}

		ki.sshPublicKeyPath = p
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

func (ki *KopsSupport) CredentialsFile(ctx context.Context) (string, errors.Error) {
	if err := ki.setup(ctx); err != nil {
		return "", err
	}

	return "", nil
}

func (ki *KopsSupport) SSHPublicKey(ctx context.Context) (string, errors.Error) {
	if err := ki.setup(ctx); err != nil {
		return "", err
	}

	return ki.sshPublicKeyPath, nil
}

func (ki *KopsSupport) EnvironmentVariables(ctx context.Context) ([]string, errors.Error) {
	if err := ki.setup(ctx); err != nil {
		return nil, err
	}

	stateStoreURL, _ := ki.StateStoreURL(ctx)

	return []string{
		fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", ki.cfg.AccessKeyID),
		fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", ki.cfg.SecretAccessKey),
		fmt.Sprintf("KOPS_STATE_STORE=%s", stateStoreURL.String()),
	}, nil
}

func (ki *KopsSupport) PlatformID() string {
	return "aws"
}

func NewKopsSupport(cfg KopsSupportConfig) *KopsSupport {
	return &KopsSupport{
		cfg: cfg,
	}
}

func writeSSHPublicKeyFile(sshPublicKey string) (string, errors.Error) {
	d, err := ioutil.TempDir("", "nebula-kops-support")
	if err != nil {
		return "", errors.NewK8sProvisionerCredentialsFileError().WithCause(err)
	}

	p := path.Join(d, "ssh.pub")

	if err := ioutil.WriteFile(p, []byte(sshPublicKey), 0644); err != nil {
		return "", errors.NewK8sProvisionerCredentialsFileError().WithCause(err)
	}

	return p, nil
}
