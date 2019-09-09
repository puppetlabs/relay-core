package taskutil

import (
	"bytes"
	"net/http"
	"net/url"
	"os"
	"path"

	"github.com/puppetlabs/nebula-tasks/pkg/errors"
)

const MetadataAPIURLEnvName = "METADATA_API_URL"

func SetOutput(key string, value string) errors.Error {
	if key == "" {
		return errors.NewTaskUtilSetOutputRequiredValueEmpty("key")
	}

	if value == "" {
		return errors.NewTaskUtilSetOutputRequiredValueEmpty("value")
	}

	mdURL := os.Getenv(MetadataAPIURLEnvName)

	if mdURL == "" {
		return errors.NewTaskUtilSetOutputRequiredValueEmpty("METADATA_API_URL")
	}

	u, err := url.Parse(mdURL)
	if err != nil {
		return errors.NewTaskUtilSetOutputMetadataAPIURLError().WithCause(err)
	}

	u.Path = path.Join("outputs", key)

	buf := bytes.NewBufferString(value)

	req, err := http.NewRequest(http.MethodPut, u.String(), buf)
	if err != nil {
		return errors.NewTaskUtilSetOutputError().WithCause(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.NewTaskUtilSetOutputError().WithCause(err)
	}

	if resp.StatusCode != http.StatusCreated {
		return errors.NewTaskUtilSetOutputUnexpectedStatusCode(int64(resp.StatusCode))
	}

	return nil
}
