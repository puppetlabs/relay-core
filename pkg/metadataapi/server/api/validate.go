package api

import (
	goerrors "errors"
	"net/http"

	utilapi "github.com/puppetlabs/horsehead/v2/httputil/api"
	"github.com/puppetlabs/horsehead/v2/instrumentation/alerts/trackers"
	"github.com/puppetlabs/relay-core/pkg/expr/evaluate"
	"github.com/puppetlabs/relay-core/pkg/expr/model"
	"github.com/puppetlabs/relay-core/pkg/manager/resolve"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/errors"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/server/middleware"
	"github.com/puppetlabs/relay-core/pkg/util/image"
	"github.com/puppetlabs/relay-core/pkg/workflow/validation"
)

func (s *Server) PostValidate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if s.schemaRegistry != nil {
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

		env := model.NewJSONResultEnvelope(rv)

		if env.Complete {
			am, err := managers.ActionMetadata().Get(ctx)
			if err != nil {
				utilapi.WriteError(ctx, w, ModelReadError(err))

				return
			}

			ref, err := image.RepoReference(am.Image)
			if err != nil {
				utilapi.WriteError(ctx, w, errors.NewActionImageParseError().WithCause(err))

				return
			}

			repository := ref.Context()

			capture, ok := trackers.CapturerFromContext(ctx)
			if ok {
				repo := repository.RepositoryStr()
				capture = capture.WithTags(trackers.Tag{Key: "relay.spec.validation-error", Value: repo})

				var captureErr error

				schema, err := s.schemaRegistry.GetByImage(ref)
				if err != nil {
					captureErr = err
					if !goerrors.Is(err, &validation.SchemaDoesNotExistError{}) {
						captureErr = errors.NewValidationSchemaLookupError().WithCause(err)
					}
				} else {
					if err := schema.ValidateGo(env.Value.Data); err != nil {
						captureErr = errors.NewValidationSchemaValidationError().WithCause(err)
					}
				}

				if captureErr != nil {
					report := capture.Capture(captureErr)
					report.AsWarning().Report(ctx)
				}
			}
		}
	}

	utilapi.WriteObjectOK(ctx, w, nil)
}
