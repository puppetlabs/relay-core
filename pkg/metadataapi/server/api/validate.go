package api

import (
	goerrors "errors"
	"net/http"

	utilapi "github.com/puppetlabs/horsehead/v2/httputil/api"
	"github.com/puppetlabs/horsehead/v2/instrumentation/alerts/trackers"
	"github.com/puppetlabs/relay-core/pkg/expr/evaluate"
	"github.com/puppetlabs/relay-core/pkg/manager/resolve"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/errors"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/server/middleware"
	"github.com/puppetlabs/relay-core/pkg/util/image"
	"github.com/puppetlabs/relay-core/pkg/workflow/validation"
)

func (s *Server) PostValidate(w http.ResponseWriter, r *http.Request) {
	if s.specSchemaRegistry != nil {
		ctx := r.Context()

		managers := middleware.Managers(r)

		spec, err := managers.Spec().Get(ctx)
		if err != nil {
			utilapi.WriteError(ctx, w, ModelReadError(err))

			return
		}

		ev := evaluate.NewEvaluator(
			evaluate.WithConnectionTypeResolver(resolve.NewConnectionTypeResolver(managers.Connections())),
			evaluate.WithParameterTypeResolver(resolve.NewParameterTypeResolver(managers.Parameters())),
			evaluate.WithOutputTypeResolver(resolve.NewOutputTypeResolver(managers.StepOutputs())),
			evaluate.WithSecretTypeResolver(resolve.NewSecretTypeResolver(managers.Secrets())),
		).ScopeTo(spec.Tree)

		rv, err := ev.EvaluateAll(ctx)
		if err != nil {
			utilapi.WriteError(ctx, w, errors.NewExpressionEvaluationError(err.Error()))

			return
		}

		env := evaluate.NewJSONResultEnvelope(rv)

		if env.Complete {
			sm, err := managers.StepMetadata().Get(ctx)
			if err != nil {
				utilapi.WriteError(ctx, w, ModelReadError(err))

				return
			}

			ref, err := image.RepoReference(sm.Image)
			if err != nil {
				utilapi.WriteError(ctx, w, errors.NewStepImageParseError().WithCause(err))

				return
			}

			repository := ref.Context()

			capture, ok := trackers.CapturerFromContext(ctx)
			if ok {
				repo := repository.RepositoryStr()
				capture = capture.WithTags(trackers.Tag{Key: "relay.spec.validation-error", Value: repo})

				var captureErr error

				schema, err := s.specSchemaRegistry.GetByStepRepository(repo)
				if err != nil {
					captureErr = err
					if !goerrors.Is(err, &validation.SchemaDoesNotExistError{}) {
						captureErr = errors.NewSpecSchemaLookupError().WithCause(err)
					}
				} else {
					if err := schema.ValidateGo(env.Value.Data); err != nil {
						captureErr = errors.NewSpecSchemaValidationError().WithCause(err)
					}
				}

				if captureErr != nil {
					report := capture.Capture(captureErr)
					report.Report(ctx)
				}
			}
		}
	}
}
