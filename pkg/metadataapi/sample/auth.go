package sample

import (
	"context"
	"net/http"

	"github.com/puppetlabs/nebula-tasks/pkg/authenticate"
	"github.com/puppetlabs/nebula-tasks/pkg/manager/builder"
	"github.com/puppetlabs/nebula-tasks/pkg/manager/memory"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/opt"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/server/middleware"
	"github.com/puppetlabs/nebula-tasks/pkg/model"
	"gopkg.in/square/go-jose.v2/jwt"
)

type Authenticator struct {
	sc   *opt.SampleConfig
	key  interface{}
	mgrs map[model.Hash]func(mgrs *builder.MetadataBuilder)
}

var _ middleware.Authenticator = &Authenticator{}

func (a *Authenticator) Authenticate(r *http.Request) (*middleware.Credential, error) {
	mgrs := builder.NewMetadataBuilder()

	auth := authenticate.NewAuthenticator(
		authenticate.NewHTTPAuthorizationHeaderIntermediary(r),
		authenticate.NewKeyResolver(
			a.key,
			authenticate.KeyResolverWithExpectation(jwt.Expected{
				Audience: jwt.Audience{authenticate.MetadataAPIAudienceV1},
			}),
		),
		authenticate.AuthenticatorWithInjector(authenticate.InjectorFunc(func(ctx context.Context, claims *authenticate.Claims) error {
			mgrs.SetSecrets(memory.NewSecretManager(a.sc.Secrets))

			// TODO: Add support for triggers!

			model.IfStep(claims.Action(), func(step *model.Step) {
				cfg, found := a.mgrs[step.Hash()]
				if !found {
					// Not a valid run, so nothing to look up here.
					return
				}

				cfg(mgrs)
			})

			return nil
		})),
	)

	if ok, err := auth.Authenticate(r.Context()); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	return &middleware.Credential{
		Managers: mgrs.Build(),
	}, nil
}

func NewAuthenticator(sc *opt.SampleConfig, key interface{}) *Authenticator {
	a := &Authenticator{
		sc:   sc,
		key:  key,
		mgrs: make(map[model.Hash]func(mgrs *builder.MetadataBuilder)),
	}

	// Pre-build managers so that changes persist across HTTP requests.
	for id, sc := range sc.Runs {
		run := model.Run{ID: id}
		som := memory.NewStepOutputMap()

		for name, sc := range sc.Steps {
			step := &model.Step{
				Run:  run,
				Name: name,
			}

			var conditionOpts []memory.ConditionManagerOption
			if sc.Conditions.Tree != nil {
				conditionOpts = append(conditionOpts, memory.ConditionManagerWithInitialCondition(sc.Conditions.Tree))
			}

			conditionManager := memory.NewConditionManager(conditionOpts...)

			var specOpts []memory.SpecManagerOption
			if sc.Spec != nil {
				specOpts = append(specOpts, memory.SpecManagerWithInitialSpec(sc.Spec.Interface()))
			}

			specManager := memory.NewSpecManager(specOpts...)

			var stateOpts []memory.StateManagerOption
			if sc.State != nil {
				stateOpts = append(stateOpts, memory.StateManagerWithInitialState(sc.State))
			}

			stateManager := memory.NewStateManager(stateOpts...)

			for name, value := range sc.Outputs {
				som.Set(step, name, value)
			}

			stepOutputManager := memory.NewStepOutputManager(step, som)

			a.mgrs[step.Hash()] = func(mgrs *builder.MetadataBuilder) {
				mgrs.SetConditions(conditionManager)
				mgrs.SetSpec(specManager)
				mgrs.SetState(stateManager)
				mgrs.SetStepOutputs(stepOutputManager)
			}
		}
	}

	return a
}