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

type tokenEntry struct {
	runID, stepName string
}

type TokenMap struct {
	m map[tokenEntry]string
}

func (tm *TokenMap) Get(runID, stepName string) (tok string, found bool) {
	tok, found = tm.m[tokenEntry{runID, stepName}]
	return
}

type TokenGenerator struct {
	key    interface{}
	issuer authenticate.Issuer
}

func (tg *TokenGenerator) Key() interface{} {
	return tg.key
}

func (tg *TokenGenerator) GenerateAll(ctx context.Context, sc *opt.SampleConfig) *TokenMap {
	tm := &TokenMap{
		m: make(map[tokenEntry]string),
	}

	for id, run := range sc.Runs {
		rm := model.Run{ID: id}

		for name := range run.Steps {
			sm := &model.Step{Run: rm, Name: name}

			claims := &authenticate.Claims{
				Claims: &jwt.Claims{
					Audience: jwt.Audience{authenticate.MetadataAPIAudienceV1},
					Subject:  fmt.Sprintf(path.Join(sm.Type().Plural, sm.Hash().HexEncoding())),
				},
				RelayRunID: rm.ID,
				RelayName:  sm.Name,
			}

			tok, err := tg.issuer.Issue(ctx, claims)
			if err != nil {
				log().Error("failed to generate token for step", "run-id", rm.ID, "step-name", sm.Name, "error", err)
			}

			tm.m[tokenEntry{rm.ID, sm.Name}] = string(tok)
			log().Info("generated JWT for step", "run-id", rm.ID, "step-name", sm.Name, "token", string(tok))
		}
	}

	return tm
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
