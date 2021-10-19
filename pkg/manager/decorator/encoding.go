package decorator

import (
	"fmt"
	"net/url"

	"github.com/mitchellh/mapstructure"
	"github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/model"
)

func DecodeInto(typ model.DecoratorType, name string, values map[string]interface{}, dec *relayv1beta1.Decorator) error {
	dec.Name = name

	switch typ {
	case model.DecoratorTypeLink:
		dl := v1beta1.DecoratorLink{}
		if err := mapstructure.Decode(values, &dl); err != nil {
			return fmt.Errorf("decorator manager: failed to map expected values to decorator: %w", err)
		}

		if _, err := url.Parse(dl.URI); err != nil {
			return fmt.Errorf("decorator manager: failed to parse uri value: %w", err)
		}

		dec.Link = &dl
	default:
		return fmt.Errorf("decorator encoding: no such decorator type: %s", typ)
	}

	return nil
}
