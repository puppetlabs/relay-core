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

type stepTokenEntry struct {
	runID, stepName string
}

type TokenMap struct {
	steps    map[stepTokenEntry]string
	triggers map[string]string
}

func (tm *TokenMap) ForStep(runID, stepName string) (tok string, found bool) {
	tok, found = tm.steps[stepTokenEntry{runID, stepName}]
	return
}

func (tm *TokenMap) ForTrigger(triggerName string) (tok string, found bool) {
	tok, found = tm.triggers[triggerName]
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
	m := &TokenMap{
		steps:    make(map[stepTokenEntry]string),
		triggers: make(map[string]string),
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

			m.steps[stepTokenEntry{rm.ID, sm.Name}] = string(tok)
			log().Info("generated JWT for step", "run-id", rm.ID, "step-name", sm.Name, "token", string(tok))
		}
	}

	for name := range sc.Triggers {
		tm := &model.Trigger{Name: name}

		claims := &authenticate.Claims{
			Claims: &jwt.Claims{
				Audience: jwt.Audience{authenticate.MetadataAPIAudienceV1},
				Subject:  fmt.Sprintf(path.Join(tm.Type().Plural, tm.Hash().HexEncoding())),
			},
			RelayName: tm.Name,
		}

		tok, err := tg.issuer.Issue(ctx, claims)
		if err != nil {
			log().Error("failed to generate token for trigger", "trigger-name", tm.Name, "error", err)
		}

		m.triggers[tm.Name] = string(tok)
		log().Info("generated JWT for trigger", "trigger-name", tm.Name, "token", string(tok))
	}

	return m
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
