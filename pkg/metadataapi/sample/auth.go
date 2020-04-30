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
)

type Authenticator struct {
	sc   *opt.SampleConfig
	tg   *TokenGenerator
	soms map[string]*memory.StepOutputMap
}

var _ middleware.Authenticator = &Authenticator{}

func (a *Authenticator) Authenticate(r *http.Request) (*middleware.Credential, error) {
	mgrs := builder.NewMetadataBuilder()

	auth := authenticate.NewAuthenticator(
		authenticate.NewHTTPAuthorizationHeaderIntermediary(r),
		authenticate.NewKeyResolver(a.tg.Key()),
		authenticate.AuthenticatorWithInjector(authenticate.InjectorFunc(func(ctx context.Context, claims *authenticate.Claims) error {
			mgrs.SetSecrets(memory.NewSecretManager(a.sc.Secrets))

			// TODO: Add support for triggers!

			model.IfStep(claims.Action(), func(step *model.Step) {
				rc, found := a.sc.Runs[step.Run.ID]
				if !found {
					// Not a valid run, so nothing to look up here.
					return
				}

				var conditionOpts []memory.ConditionManagerOption
				var specOpts []memory.SpecManagerOption
				var stateOpts []memory.StateManagerOption

				if sc, found := rc.Steps[step.Name]; found {
					if sc.Conditions.Tree != nil {
						conditionOpts = append(conditionOpts, memory.ConditionManagerWithInitialCondition(sc.Conditions.Tree))
					}
					if sc.Spec != nil {
						specOpts = append(specOpts, memory.SpecManagerWithInitialSpec(sc.Spec.Interface()))
					}
					if sc.State != nil {
						stateOpts = append(stateOpts, memory.StateManagerWithInitialState(sc.State))
					}
				}

				mgrs.SetConditions(memory.NewConditionManager(conditionOpts...))
				mgrs.SetSpec(memory.NewSpecManager(specOpts...))
				mgrs.SetState(memory.NewStateManager(stateOpts...))
				mgrs.SetStepOutputs(memory.NewStepOutputManager(step, a.soms[step.Run.ID]))
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

func NewAuthenticator(sc *opt.SampleConfig, tg *TokenGenerator) *Authenticator {
	soms := make(map[string]*memory.StepOutputMap)
	for id, sc := range sc.Runs {
		run := model.Run{ID: id}
		som := memory.NewStepOutputMap()

		for name, sc := range sc.Steps {
			step := &model.Step{
				Run:  run,
				Name: name,
			}

			for name, value := range sc.Outputs {
				som.Set(step, name, value)
			}
		}

		soms[id] = som
	}

	return &Authenticator{
		sc:   sc,
		tg:   tg,
		soms: soms,
	}
}
