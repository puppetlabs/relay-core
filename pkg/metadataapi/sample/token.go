package sample

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"path"

	"github.com/puppetlabs/nebula-tasks/pkg/authenticate"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/opt"
	"github.com/puppetlabs/nebula-tasks/pkg/model"
	"gopkg.in/square/go-jose.v2/jwt"
)

type TokenGenerator struct {
	key    interface{}
	issuer authenticate.Issuer
}

func (tg *TokenGenerator) Key() interface{} {
	return tg.key
}

func (tg *TokenGenerator) LogAll(ctx context.Context, sc *opt.SampleConfig) {
	for id, run := range sc.Runs {
		rm := model.Run{ID: id}

		for name := range run.Steps {
			sm := &model.Step{Run: rm, Name: name}

			claims := &authenticate.Claims{
				Claims: &jwt.Claims{
					Subject: fmt.Sprintf(path.Join(sm.Type().Plural, sm.Hash().HexEncoding())),
				},
				RelayRunID: rm.ID,
				RelayName:  sm.Name,
			}

			tok, err := tg.issuer.Issue(ctx, claims)
			if err != nil {
				log().Error("failed to generate token for step", "run-id", rm.ID, "step-name", sm.Name, "error", err)
			}

			log().Info("generated JWT for step", "run-id", rm.ID, "step-name", sm.Name, "token", string(tok))
		}
	}
}

func NewHS256TokenGenerator(key []byte) (*TokenGenerator, error) {
	if len(key) == 0 {
		// Generate a new key.
		key = make([]byte, 64)
		if _, err := rand.Read(key); err != nil {
			return nil, err
		}

		log().Info("created new HMAC-SHA256 signing key", "key", base64.StdEncoding.EncodeToString(key))
	}

	issuer, err := authenticate.NewHS256KeySignerIssuer(key)
	if err != nil {
		return nil, err
	}

	return &TokenGenerator{
		key:    key,
		issuer: issuer,
	}, nil
}
