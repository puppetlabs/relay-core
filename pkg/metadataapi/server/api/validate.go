package api

import (
	goerrors "errors"
	"net/http"

	utilapi "github.com/puppetlabs/leg/httputil/api"
	"github.com/puppetlabs/leg/relspec/pkg/evaluate"
	"github.com/puppetlabs/relay-core/pkg/manager/specadapter"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/errors"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/server/middleware"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/spec"
	"github.com/puppetlabs/relay-core/pkg/util/image"
	"github.com/puppetlabs/relay-core/pkg/workflow/validation"
)

func (s *Server) PostValidate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if s.schemaRegistry != nil {
		ctx := r.Context()

		managers := middleware.Managers(r)

		data, err := managers.Spec().Get(ctx)
		if err != nil {
			utilapi.WriteError(ctx, w, ModelReadError(err))

			return
		}

		ev := spec.NewEvaluator(
			spec.WithConnectionTypeResolver{ConnectionTypeResolver: specadapter.NewConnectionTypeResolver(managers.Connections())},
			spec.WithSecretTypeResolver{SecretTypeResolver: specadapter.NewSecretTypeResolver(managers.Secrets())},
			spec.WithParameterTypeResolver{ParameterTypeResolver: specadapter.NewParameterTypeResolver(managers.Parameters())},
			spec.WithOutputTypeResolver{OutputTypeResolver: specadapter.NewOutputTypeResolver(managers.StepOutputs())},
			spec.WithStatusTypeResolver{StatusTypeResolver: specadapter.NewStatusTypeResolver(managers.ActionStatus())},
		)

		rv, err := evaluate.EvaluateAll(ctx, ev, data.Tree)
		if err != nil {
			utilapi.WriteError(ctx, w, errors.NewExpressionEvaluationError(err.Error()))

			return
		}

		if rv.OK() {
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

			schema, err := s.schemaRegistry.GetByImage(ref)
			if err != nil {
				var noTrackCause *validation.SchemaDoesNotExistError
				if !goerrors.As(err, &noTrackCause) {
					addStepMessage(r, err.Error(),
						nil,
						&model.SchemaValidationResult{
							Expression: data.Tree,
						})
				}
			} else {
				if err := schema.ValidateGo(rv.Value); err != nil {
					addStepMessage(r, err.Error(),
						nil,
						&model.SchemaValidationResult{
							Expression: data.Tree,
						})
				}
			}
		}
	}

	utilapi.WriteObjectOK(ctx, w, nil)
}
