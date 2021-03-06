package api

import (
	goerrors "errors"
	"net/http"

	utilapi "github.com/puppetlabs/leg/httputil/api"
	"github.com/puppetlabs/leg/instrumentation/alerts/trackers"
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
			evaluate.WithConnectionTypeResolver{ConnectionTypeResolver: resolve.NewConnectionTypeResolver(managers.Connections())},
			evaluate.WithSecretTypeResolver{SecretTypeResolver: resolve.NewSecretTypeResolver(managers.Secrets())},
			evaluate.WithParameterTypeResolver{ParameterTypeResolver: resolve.NewParameterTypeResolver(managers.Parameters())},
			evaluate.WithOutputTypeResolver{OutputTypeResolver: resolve.NewOutputTypeResolver(managers.StepOutputs())},
		)

		rv, err := model.EvaluateAll(ctx, ev, spec.Tree)
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
					var noTrackCause *validation.SchemaDoesNotExistError
					if !goerrors.As(err, &noTrackCause) {
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
