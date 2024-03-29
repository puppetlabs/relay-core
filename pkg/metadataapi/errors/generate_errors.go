// Package errors contains errors for the domain "rma".
//
// This file is automatically generated by errawr-gen. Do not modify it.
package errors

import (
	errawr "github.com/puppetlabs/errawr-go/v2/pkg/errawr"
	impl "github.com/puppetlabs/errawr-go/v2/pkg/impl"
)

// Error is the type of all errors generated by this package.
type Error interface {
	errawr.Error
}

// External contains methods that can be used externally to help consume errors from this package.
type External struct{}

// API is a singleton instance of the External type.
var API External

// Domain is the general domain in which all errors in this package belong.
var Domain = &impl.ErrorDomain{
	Key:   "rma",
	Title: "Relay Metadata API",
}

// ActionSection defines a section of errors with the following scope:
// Action errors
var ActionSection = &impl.ErrorSection{
	Key:   "action",
	Title: "Action errors",
}

// ActionImageParseErrorCode is the code for an instance of "image_parse_error".
const ActionImageParseErrorCode = "rma_action_image_parse_error"

// IsActionImageParseError tests whether a given error is an instance of "image_parse_error".
func IsActionImageParseError(err errawr.Error) bool {
	return err != nil && err.Is(ActionImageParseErrorCode)
}

// IsActionImageParseError tests whether a given error is an instance of "image_parse_error".
func (External) IsActionImageParseError(err errawr.Error) bool {
	return IsActionImageParseError(err)
}

// ActionImageParseErrorBuilder is a builder for "image_parse_error" errors.
type ActionImageParseErrorBuilder struct {
	arguments impl.ErrorArguments
}

// Build creates the error for the code "image_parse_error" from this builder.
func (b *ActionImageParseErrorBuilder) Build() Error {
	description := &impl.ErrorDescription{
		Friendly:  "failed to parse the image and tag string for the container action.",
		Technical: "failed to parse the image and tag string for the container action.",
	}

	return &impl.Error{
		ErrorArguments:   b.arguments,
		ErrorCode:        "image_parse_error",
		ErrorDescription: description,
		ErrorDomain:      Domain,
		ErrorMetadata:    &impl.ErrorMetadata{},
		ErrorSection:     ActionSection,
		ErrorSensitivity: errawr.ErrorSensitivityNone,
		ErrorTitle:       "Image parse error",
		Version:          1,
	}
}

// NewActionImageParseErrorBuilder creates a new error builder for the code "image_parse_error".
func NewActionImageParseErrorBuilder() *ActionImageParseErrorBuilder {
	return &ActionImageParseErrorBuilder{arguments: impl.ErrorArguments{}}
}

// NewActionImageParseError creates a new error with the code "image_parse_error".
func NewActionImageParseError() Error {
	return NewActionImageParseErrorBuilder().Build()
}

// APISection defines a section of errors with the following scope:
// API errors
var APISection = &impl.ErrorSection{
	Key:   "api",
	Title: "API errors",
}

// APIAuthenticationErrorCode is the code for an instance of "authentication_error".
const APIAuthenticationErrorCode = "rma_api_authentication_error"

// IsAPIAuthenticationError tests whether a given error is an instance of "authentication_error".
func IsAPIAuthenticationError(err errawr.Error) bool {
	return err != nil && err.Is(APIAuthenticationErrorCode)
}

// IsAPIAuthenticationError tests whether a given error is an instance of "authentication_error".
func (External) IsAPIAuthenticationError(err errawr.Error) bool {
	return IsAPIAuthenticationError(err)
}

// APIAuthenticationErrorBuilder is a builder for "authentication_error" errors.
type APIAuthenticationErrorBuilder struct {
	arguments impl.ErrorArguments
}

// Build creates the error for the code "authentication_error" from this builder.
func (b *APIAuthenticationErrorBuilder) Build() Error {
	description := &impl.ErrorDescription{
		Friendly:  "The provided credential information did not authenticate this request.",
		Technical: "The provided credential information did not authenticate this request.",
	}

	return &impl.Error{
		ErrorArguments:   b.arguments,
		ErrorCode:        "authentication_error",
		ErrorDescription: description,
		ErrorDomain:      Domain,
		ErrorMetadata: &impl.ErrorMetadata{HTTPErrorMetadata: &impl.HTTPErrorMetadata{
			ErrorHeaders: impl.HTTPErrorMetadataHeaders{},
			ErrorStatus:  401,
		}},
		ErrorSection:     APISection,
		ErrorSensitivity: errawr.ErrorSensitivityNone,
		ErrorTitle:       "Unauthorized",
		Version:          1,
	}
}

// NewAPIAuthenticationErrorBuilder creates a new error builder for the code "authentication_error".
func NewAPIAuthenticationErrorBuilder() *APIAuthenticationErrorBuilder {
	return &APIAuthenticationErrorBuilder{arguments: impl.ErrorArguments{}}
}

// NewAPIAuthenticationError creates a new error with the code "authentication_error".
func NewAPIAuthenticationError() Error {
	return NewAPIAuthenticationErrorBuilder().Build()
}

// APIMalformedRequestErrorCode is the code for an instance of "malformed_request_error".
const APIMalformedRequestErrorCode = "rma_api_malformed_request_error"

// IsAPIMalformedRequestError tests whether a given error is an instance of "malformed_request_error".
func IsAPIMalformedRequestError(err errawr.Error) bool {
	return err != nil && err.Is(APIMalformedRequestErrorCode)
}

// IsAPIMalformedRequestError tests whether a given error is an instance of "malformed_request_error".
func (External) IsAPIMalformedRequestError(err errawr.Error) bool {
	return IsAPIMalformedRequestError(err)
}

// APIMalformedRequestErrorBuilder is a builder for "malformed_request_error" errors.
type APIMalformedRequestErrorBuilder struct {
	arguments impl.ErrorArguments
}

// Build creates the error for the code "malformed_request_error" from this builder.
func (b *APIMalformedRequestErrorBuilder) Build() Error {
	description := &impl.ErrorDescription{
		Friendly:  "The request body you provided could not be deserialized.",
		Technical: "The request body you provided could not be deserialized.",
	}

	return &impl.Error{
		ErrorArguments:   b.arguments,
		ErrorCode:        "malformed_request_error",
		ErrorDescription: description,
		ErrorDomain:      Domain,
		ErrorMetadata: &impl.ErrorMetadata{HTTPErrorMetadata: &impl.HTTPErrorMetadata{
			ErrorHeaders: impl.HTTPErrorMetadataHeaders{},
			ErrorStatus:  422,
		}},
		ErrorSection:     APISection,
		ErrorSensitivity: errawr.ErrorSensitivityNone,
		ErrorTitle:       "Malformed request",
		Version:          1,
	}
}

// NewAPIMalformedRequestErrorBuilder creates a new error builder for the code "malformed_request_error".
func NewAPIMalformedRequestErrorBuilder() *APIMalformedRequestErrorBuilder {
	return &APIMalformedRequestErrorBuilder{arguments: impl.ErrorArguments{}}
}

// NewAPIMalformedRequestError creates a new error with the code "malformed_request_error".
func NewAPIMalformedRequestError() Error {
	return NewAPIMalformedRequestErrorBuilder().Build()
}

// APIObjectSerializationErrorCode is the code for an instance of "object_serialization_error".
const APIObjectSerializationErrorCode = "rma_api_object_serialization_error"

// IsAPIObjectSerializationError tests whether a given error is an instance of "object_serialization_error".
func IsAPIObjectSerializationError(err errawr.Error) bool {
	return err != nil && err.Is(APIObjectSerializationErrorCode)
}

// IsAPIObjectSerializationError tests whether a given error is an instance of "object_serialization_error".
func (External) IsAPIObjectSerializationError(err errawr.Error) bool {
	return IsAPIObjectSerializationError(err)
}

// APIObjectSerializationErrorBuilder is a builder for "object_serialization_error" errors.
type APIObjectSerializationErrorBuilder struct {
	arguments impl.ErrorArguments
}

// Build creates the error for the code "object_serialization_error" from this builder.
func (b *APIObjectSerializationErrorBuilder) Build() Error {
	description := &impl.ErrorDescription{
		Friendly:  "The response object failed to serialize.",
		Technical: "The response object failed to serialize.",
	}

	return &impl.Error{
		ErrorArguments:   b.arguments,
		ErrorCode:        "object_serialization_error",
		ErrorDescription: description,
		ErrorDomain:      Domain,
		ErrorMetadata: &impl.ErrorMetadata{HTTPErrorMetadata: &impl.HTTPErrorMetadata{
			ErrorHeaders: impl.HTTPErrorMetadataHeaders{},
			ErrorStatus:  422,
		}},
		ErrorSection:     APISection,
		ErrorSensitivity: errawr.ErrorSensitivityNone,
		ErrorTitle:       "Object serialization error",
		Version:          1,
	}
}

// NewAPIObjectSerializationErrorBuilder creates a new error builder for the code "object_serialization_error".
func NewAPIObjectSerializationErrorBuilder() *APIObjectSerializationErrorBuilder {
	return &APIObjectSerializationErrorBuilder{arguments: impl.ErrorArguments{}}
}

// NewAPIObjectSerializationError creates a new error with the code "object_serialization_error".
func NewAPIObjectSerializationError() Error {
	return NewAPIObjectSerializationErrorBuilder().Build()
}

// APIUnknownRequestMediaTypeErrorCode is the code for an instance of "unknown_request_media_type_error".
const APIUnknownRequestMediaTypeErrorCode = "rma_api_unknown_request_media_type_error"

// IsAPIUnknownRequestMediaTypeError tests whether a given error is an instance of "unknown_request_media_type_error".
func IsAPIUnknownRequestMediaTypeError(err errawr.Error) bool {
	return err != nil && err.Is(APIUnknownRequestMediaTypeErrorCode)
}

// IsAPIUnknownRequestMediaTypeError tests whether a given error is an instance of "unknown_request_media_type_error".
func (External) IsAPIUnknownRequestMediaTypeError(err errawr.Error) bool {
	return IsAPIUnknownRequestMediaTypeError(err)
}

// APIUnknownRequestMediaTypeErrorBuilder is a builder for "unknown_request_media_type_error" errors.
type APIUnknownRequestMediaTypeErrorBuilder struct {
	arguments impl.ErrorArguments
}

// Build creates the error for the code "unknown_request_media_type_error" from this builder.
func (b *APIUnknownRequestMediaTypeErrorBuilder) Build() Error {
	description := &impl.ErrorDescription{
		Friendly:  "We don't know how to decode a request body with media type {{pre mediaType}}.",
		Technical: "We don't know how to decode a request body with media type {{pre mediaType}}.",
	}

	return &impl.Error{
		ErrorArguments:   b.arguments,
		ErrorCode:        "unknown_request_media_type_error",
		ErrorDescription: description,
		ErrorDomain:      Domain,
		ErrorMetadata: &impl.ErrorMetadata{HTTPErrorMetadata: &impl.HTTPErrorMetadata{
			ErrorHeaders: impl.HTTPErrorMetadataHeaders{},
			ErrorStatus:  422,
		}},
		ErrorSection:     APISection,
		ErrorSensitivity: errawr.ErrorSensitivityNone,
		ErrorTitle:       "Unknown media type",
		Version:          1,
	}
}

// NewAPIUnknownRequestMediaTypeErrorBuilder creates a new error builder for the code "unknown_request_media_type_error".
func NewAPIUnknownRequestMediaTypeErrorBuilder(mediaType string) *APIUnknownRequestMediaTypeErrorBuilder {
	return &APIUnknownRequestMediaTypeErrorBuilder{arguments: impl.ErrorArguments{"mediaType": impl.NewErrorArgument(mediaType, "the unexpected media type")}}
}

// NewAPIUnknownRequestMediaTypeError creates a new error with the code "unknown_request_media_type_error".
func NewAPIUnknownRequestMediaTypeError(mediaType string) Error {
	return NewAPIUnknownRequestMediaTypeErrorBuilder(mediaType).Build()
}

// ConditionSection defines a section of errors with the following scope:
// Condition errors
var ConditionSection = &impl.ErrorSection{
	Key:   "condition",
	Title: "Condition errors",
}

// ConditionTypeErrorCode is the code for an instance of "type_error".
const ConditionTypeErrorCode = "rma_condition_type_error"

// IsConditionTypeError tests whether a given error is an instance of "type_error".
func IsConditionTypeError(err errawr.Error) bool {
	return err != nil && err.Is(ConditionTypeErrorCode)
}

// IsConditionTypeError tests whether a given error is an instance of "type_error".
func (External) IsConditionTypeError(err errawr.Error) bool {
	return IsConditionTypeError(err)
}

// ConditionTypeErrorBuilder is a builder for "type_error" errors.
type ConditionTypeErrorBuilder struct {
	arguments impl.ErrorArguments
}

// Build creates the error for the code "type_error" from this builder.
func (b *ConditionTypeErrorBuilder) Build() Error {
	description := &impl.ErrorDescription{
		Friendly:  "A condition must evaluate to a single boolean value or a list of boolean values, but your condition provided a {{pre type}}.",
		Technical: "A condition must evaluate to a single boolean value or a list of boolean values, but your condition provided a {{pre type}}.",
	}

	return &impl.Error{
		ErrorArguments:   b.arguments,
		ErrorCode:        "type_error",
		ErrorDescription: description,
		ErrorDomain:      Domain,
		ErrorMetadata: &impl.ErrorMetadata{HTTPErrorMetadata: &impl.HTTPErrorMetadata{
			ErrorHeaders: impl.HTTPErrorMetadataHeaders{},
			ErrorStatus:  422,
		}},
		ErrorSection:     ConditionSection,
		ErrorSensitivity: errawr.ErrorSensitivityNone,
		ErrorTitle:       "Type error",
		Version:          1,
	}
}

// NewConditionTypeErrorBuilder creates a new error builder for the code "type_error".
func NewConditionTypeErrorBuilder(type_ string) *ConditionTypeErrorBuilder {
	return &ConditionTypeErrorBuilder{arguments: impl.ErrorArguments{"type": impl.NewErrorArgument(type_, "the unexpected type")}}
}

// NewConditionTypeError creates a new error with the code "type_error".
func NewConditionTypeError(type_ string) Error {
	return NewConditionTypeErrorBuilder(type_).Build()
}

// ExpressionSection defines a section of errors with the following scope:
// Expression errors
var ExpressionSection = &impl.ErrorSection{
	Key:   "expression",
	Title: "Expression errors",
}

// ExpressionEvaluationErrorCode is the code for an instance of "evaluation_error".
const ExpressionEvaluationErrorCode = "rma_expression_evaluation_error"

// IsExpressionEvaluationError tests whether a given error is an instance of "evaluation_error".
func IsExpressionEvaluationError(err errawr.Error) bool {
	return err != nil && err.Is(ExpressionEvaluationErrorCode)
}

// IsExpressionEvaluationError tests whether a given error is an instance of "evaluation_error".
func (External) IsExpressionEvaluationError(err errawr.Error) bool {
	return IsExpressionEvaluationError(err)
}

// ExpressionEvaluationErrorBuilder is a builder for "evaluation_error" errors.
type ExpressionEvaluationErrorBuilder struct {
	arguments impl.ErrorArguments
}

// Build creates the error for the code "evaluation_error" from this builder.
func (b *ExpressionEvaluationErrorBuilder) Build() Error {
	description := &impl.ErrorDescription{
		Friendly:  "We could not evaluate this resource expression.\n{{error}}",
		Technical: "We could not evaluate this resource expression.\n{{error}}",
	}

	return &impl.Error{
		ErrorArguments:   b.arguments,
		ErrorCode:        "evaluation_error",
		ErrorDescription: description,
		ErrorDomain:      Domain,
		ErrorMetadata: &impl.ErrorMetadata{HTTPErrorMetadata: &impl.HTTPErrorMetadata{
			ErrorHeaders: impl.HTTPErrorMetadataHeaders{},
			ErrorStatus:  422,
		}},
		ErrorSection:     ExpressionSection,
		ErrorSensitivity: errawr.ErrorSensitivityNone,
		ErrorTitle:       "Evaluation error",
		Version:          1,
	}
}

// NewExpressionEvaluationErrorBuilder creates a new error builder for the code "evaluation_error".
func NewExpressionEvaluationErrorBuilder(error string) *ExpressionEvaluationErrorBuilder {
	return &ExpressionEvaluationErrorBuilder{arguments: impl.ErrorArguments{"error": impl.NewErrorArgument(error, "the problem that caused the evaluation error")}}
}

// NewExpressionEvaluationError creates a new error with the code "evaluation_error".
func NewExpressionEvaluationError(error string) Error {
	return NewExpressionEvaluationErrorBuilder(error).Build()
}

// ExpressionUnresolvableErrorCode is the code for an instance of "unresolvable_error".
const ExpressionUnresolvableErrorCode = "rma_expression_unresolvable_error"

// IsExpressionUnresolvableError tests whether a given error is an instance of "unresolvable_error".
func IsExpressionUnresolvableError(err errawr.Error) bool {
	return err != nil && err.Is(ExpressionUnresolvableErrorCode)
}

// IsExpressionUnresolvableError tests whether a given error is an instance of "unresolvable_error".
func (External) IsExpressionUnresolvableError(err errawr.Error) bool {
	return IsExpressionUnresolvableError(err)
}

// ExpressionUnresolvableErrorBuilder is a builder for "unresolvable_error" errors.
type ExpressionUnresolvableErrorBuilder struct {
	arguments impl.ErrorArguments
}

// Build creates the error for the code "unresolvable_error" from this builder.
func (b *ExpressionUnresolvableErrorBuilder) Build() Error {
	description := &impl.ErrorDescription{
		Friendly:  "We could not fully evaluate this resource expression:\n{{#enum errors}}{{this}}{{/enum}}",
		Technical: "We could not fully evaluate this resource expression:\n{{#enum errors}}{{this}}{{/enum}}",
	}

	return &impl.Error{
		ErrorArguments:   b.arguments,
		ErrorCode:        "unresolvable_error",
		ErrorDescription: description,
		ErrorDomain:      Domain,
		ErrorMetadata: &impl.ErrorMetadata{HTTPErrorMetadata: &impl.HTTPErrorMetadata{
			ErrorHeaders: impl.HTTPErrorMetadataHeaders{},
			ErrorStatus:  422,
		}},
		ErrorSection:     ExpressionSection,
		ErrorSensitivity: errawr.ErrorSensitivityNone,
		ErrorTitle:       "Unresolvable",
		Version:          1,
	}
}

// NewExpressionUnresolvableErrorBuilder creates a new error builder for the code "unresolvable_error".
func NewExpressionUnresolvableErrorBuilder(errors []string) *ExpressionUnresolvableErrorBuilder {
	return &ExpressionUnresolvableErrorBuilder{arguments: impl.ErrorArguments{"errors": impl.NewErrorArgument(errors, "the unresolvables from the expression")}}
}

// NewExpressionUnresolvableError creates a new error with the code "unresolvable_error".
func NewExpressionUnresolvableError(errors []string) Error {
	return NewExpressionUnresolvableErrorBuilder(errors).Build()
}

// ExpressionUnsupportedLanguageErrorCode is the code for an instance of "unsupported_language_error".
const ExpressionUnsupportedLanguageErrorCode = "rma_expression_unsupported_language_error"

// IsExpressionUnsupportedLanguageError tests whether a given error is an instance of "unsupported_language_error".
func IsExpressionUnsupportedLanguageError(err errawr.Error) bool {
	return err != nil && err.Is(ExpressionUnsupportedLanguageErrorCode)
}

// IsExpressionUnsupportedLanguageError tests whether a given error is an instance of "unsupported_language_error".
func (External) IsExpressionUnsupportedLanguageError(err errawr.Error) bool {
	return IsExpressionUnsupportedLanguageError(err)
}

// ExpressionUnsupportedLanguageErrorBuilder is a builder for "unsupported_language_error" errors.
type ExpressionUnsupportedLanguageErrorBuilder struct {
	arguments impl.ErrorArguments
}

// Build creates the error for the code "unsupported_language_error" from this builder.
func (b *ExpressionUnsupportedLanguageErrorBuilder) Build() Error {
	description := &impl.ErrorDescription{
		Friendly:  "We don't know the query language {{quote language}}.",
		Technical: "We don't know the query language {{quote language}}.",
	}

	return &impl.Error{
		ErrorArguments:   b.arguments,
		ErrorCode:        "unsupported_language_error",
		ErrorDescription: description,
		ErrorDomain:      Domain,
		ErrorMetadata: &impl.ErrorMetadata{HTTPErrorMetadata: &impl.HTTPErrorMetadata{
			ErrorHeaders: impl.HTTPErrorMetadataHeaders{},
			ErrorStatus:  422,
		}},
		ErrorSection:     ExpressionSection,
		ErrorSensitivity: errawr.ErrorSensitivityNone,
		ErrorTitle:       "Unsupported query language",
		Version:          1,
	}
}

// NewExpressionUnsupportedLanguageErrorBuilder creates a new error builder for the code "unsupported_language_error".
func NewExpressionUnsupportedLanguageErrorBuilder(language string) *ExpressionUnsupportedLanguageErrorBuilder {
	return &ExpressionUnsupportedLanguageErrorBuilder{arguments: impl.ErrorArguments{"language": impl.NewErrorArgument(language, "the requested query language")}}
}

// NewExpressionUnsupportedLanguageError creates a new error with the code "unsupported_language_error".
func NewExpressionUnsupportedLanguageError(language string) Error {
	return NewExpressionUnsupportedLanguageErrorBuilder(language).Build()
}

// ModelSection defines a section of errors with the following scope:
// Model errors
var ModelSection = &impl.ErrorSection{
	Key:   "model",
	Title: "Model errors",
}

// ModelAuthorizationErrorCode is the code for an instance of "authorization_error".
const ModelAuthorizationErrorCode = "rma_model_authorization_error"

// IsModelAuthorizationError tests whether a given error is an instance of "authorization_error".
func IsModelAuthorizationError(err errawr.Error) bool {
	return err != nil && err.Is(ModelAuthorizationErrorCode)
}

// IsModelAuthorizationError tests whether a given error is an instance of "authorization_error".
func (External) IsModelAuthorizationError(err errawr.Error) bool {
	return IsModelAuthorizationError(err)
}

// ModelAuthorizationErrorBuilder is a builder for "authorization_error" errors.
type ModelAuthorizationErrorBuilder struct {
	arguments impl.ErrorArguments
}

// Build creates the error for the code "authorization_error" from this builder.
func (b *ModelAuthorizationErrorBuilder) Build() Error {
	description := &impl.ErrorDescription{
		Friendly:  "You are not authorized to access this resource.",
		Technical: "You are not authorized to access this resource.",
	}

	return &impl.Error{
		ErrorArguments:   b.arguments,
		ErrorCode:        "authorization_error",
		ErrorDescription: description,
		ErrorDomain:      Domain,
		ErrorMetadata: &impl.ErrorMetadata{HTTPErrorMetadata: &impl.HTTPErrorMetadata{
			ErrorHeaders: impl.HTTPErrorMetadataHeaders{},
			ErrorStatus:  403,
		}},
		ErrorSection:     ModelSection,
		ErrorSensitivity: errawr.ErrorSensitivityNone,
		ErrorTitle:       "Unauthorized",
		Version:          1,
	}
}

// NewModelAuthorizationErrorBuilder creates a new error builder for the code "authorization_error".
func NewModelAuthorizationErrorBuilder() *ModelAuthorizationErrorBuilder {
	return &ModelAuthorizationErrorBuilder{arguments: impl.ErrorArguments{}}
}

// NewModelAuthorizationError creates a new error with the code "authorization_error".
func NewModelAuthorizationError() Error {
	return NewModelAuthorizationErrorBuilder().Build()
}

// ModelConflictErrorCode is the code for an instance of "conflict_error".
const ModelConflictErrorCode = "rma_model_conflict_error"

// IsModelConflictError tests whether a given error is an instance of "conflict_error".
func IsModelConflictError(err errawr.Error) bool {
	return err != nil && err.Is(ModelConflictErrorCode)
}

// IsModelConflictError tests whether a given error is an instance of "conflict_error".
func (External) IsModelConflictError(err errawr.Error) bool {
	return IsModelConflictError(err)
}

// ModelConflictErrorBuilder is a builder for "conflict_error" errors.
type ModelConflictErrorBuilder struct {
	arguments impl.ErrorArguments
}

// Build creates the error for the code "conflict_error" from this builder.
func (b *ModelConflictErrorBuilder) Build() Error {
	description := &impl.ErrorDescription{
		Friendly:  "The resource you're updating already exists, and your version conflicts with the stored version.",
		Technical: "The resource you're updating already exists, and your version conflicts with the stored version.",
	}

	return &impl.Error{
		ErrorArguments:   b.arguments,
		ErrorCode:        "conflict_error",
		ErrorDescription: description,
		ErrorDomain:      Domain,
		ErrorMetadata: &impl.ErrorMetadata{HTTPErrorMetadata: &impl.HTTPErrorMetadata{
			ErrorHeaders: impl.HTTPErrorMetadataHeaders{},
			ErrorStatus:  409,
		}},
		ErrorSection:     ModelSection,
		ErrorSensitivity: errawr.ErrorSensitivityNone,
		ErrorTitle:       "Conflict",
		Version:          1,
	}
}

// NewModelConflictErrorBuilder creates a new error builder for the code "conflict_error".
func NewModelConflictErrorBuilder() *ModelConflictErrorBuilder {
	return &ModelConflictErrorBuilder{arguments: impl.ErrorArguments{}}
}

// NewModelConflictError creates a new error with the code "conflict_error".
func NewModelConflictError() Error {
	return NewModelConflictErrorBuilder().Build()
}

// ModelNotFoundErrorCode is the code for an instance of "not_found_error".
const ModelNotFoundErrorCode = "rma_model_not_found_error"

// IsModelNotFoundError tests whether a given error is an instance of "not_found_error".
func IsModelNotFoundError(err errawr.Error) bool {
	return err != nil && err.Is(ModelNotFoundErrorCode)
}

// IsModelNotFoundError tests whether a given error is an instance of "not_found_error".
func (External) IsModelNotFoundError(err errawr.Error) bool {
	return IsModelNotFoundError(err)
}

// ModelNotFoundErrorBuilder is a builder for "not_found_error" errors.
type ModelNotFoundErrorBuilder struct {
	arguments impl.ErrorArguments
}

// Build creates the error for the code "not_found_error" from this builder.
func (b *ModelNotFoundErrorBuilder) Build() Error {
	description := &impl.ErrorDescription{
		Friendly:  "The resource you requested does not exist.",
		Technical: "The resource you requested does not exist.",
	}

	return &impl.Error{
		ErrorArguments:   b.arguments,
		ErrorCode:        "not_found_error",
		ErrorDescription: description,
		ErrorDomain:      Domain,
		ErrorMetadata: &impl.ErrorMetadata{HTTPErrorMetadata: &impl.HTTPErrorMetadata{
			ErrorHeaders: impl.HTTPErrorMetadataHeaders{},
			ErrorStatus:  404,
		}},
		ErrorSection:     ModelSection,
		ErrorSensitivity: errawr.ErrorSensitivityNone,
		ErrorTitle:       "Not found",
		Version:          1,
	}
}

// NewModelNotFoundErrorBuilder creates a new error builder for the code "not_found_error".
func NewModelNotFoundErrorBuilder() *ModelNotFoundErrorBuilder {
	return &ModelNotFoundErrorBuilder{arguments: impl.ErrorArguments{}}
}

// NewModelNotFoundError creates a new error with the code "not_found_error".
func NewModelNotFoundError() Error {
	return NewModelNotFoundErrorBuilder().Build()
}

// ModelReadErrorCode is the code for an instance of "read_error".
const ModelReadErrorCode = "rma_model_read_error"

// IsModelReadError tests whether a given error is an instance of "read_error".
func IsModelReadError(err errawr.Error) bool {
	return err != nil && err.Is(ModelReadErrorCode)
}

// IsModelReadError tests whether a given error is an instance of "read_error".
func (External) IsModelReadError(err errawr.Error) bool {
	return IsModelReadError(err)
}

// ModelReadErrorBuilder is a builder for "read_error" errors.
type ModelReadErrorBuilder struct {
	arguments impl.ErrorArguments
}

// Build creates the error for the code "read_error" from this builder.
func (b *ModelReadErrorBuilder) Build() Error {
	description := &impl.ErrorDescription{
		Friendly:  "We could not read this resource.",
		Technical: "We could not read this resource.",
	}

	return &impl.Error{
		ErrorArguments:   b.arguments,
		ErrorCode:        "read_error",
		ErrorDescription: description,
		ErrorDomain:      Domain,
		ErrorMetadata:    &impl.ErrorMetadata{},
		ErrorSection:     ModelSection,
		ErrorSensitivity: errawr.ErrorSensitivityNone,
		ErrorTitle:       "Read error",
		Version:          1,
	}
}

// NewModelReadErrorBuilder creates a new error builder for the code "read_error".
func NewModelReadErrorBuilder() *ModelReadErrorBuilder {
	return &ModelReadErrorBuilder{arguments: impl.ErrorArguments{}}
}

// NewModelReadError creates a new error with the code "read_error".
func NewModelReadError() Error {
	return NewModelReadErrorBuilder().Build()
}

// ModelWriteErrorCode is the code for an instance of "write_error".
const ModelWriteErrorCode = "rma_model_write_error"

// IsModelWriteError tests whether a given error is an instance of "write_error".
func IsModelWriteError(err errawr.Error) bool {
	return err != nil && err.Is(ModelWriteErrorCode)
}

// IsModelWriteError tests whether a given error is an instance of "write_error".
func (External) IsModelWriteError(err errawr.Error) bool {
	return IsModelWriteError(err)
}

// ModelWriteErrorBuilder is a builder for "write_error" errors.
type ModelWriteErrorBuilder struct {
	arguments impl.ErrorArguments
}

// Build creates the error for the code "write_error" from this builder.
func (b *ModelWriteErrorBuilder) Build() Error {
	description := &impl.ErrorDescription{
		Friendly:  "We could not persist this resource.",
		Technical: "We could not persist this resource.",
	}

	return &impl.Error{
		ErrorArguments:   b.arguments,
		ErrorCode:        "write_error",
		ErrorDescription: description,
		ErrorDomain:      Domain,
		ErrorMetadata:    &impl.ErrorMetadata{},
		ErrorSection:     ModelSection,
		ErrorSensitivity: errawr.ErrorSensitivityNone,
		ErrorTitle:       "Write error",
		Version:          1,
	}
}

// NewModelWriteErrorBuilder creates a new error builder for the code "write_error".
func NewModelWriteErrorBuilder() *ModelWriteErrorBuilder {
	return &ModelWriteErrorBuilder{arguments: impl.ErrorArguments{}}
}

// NewModelWriteError creates a new error with the code "write_error".
func NewModelWriteError() Error {
	return NewModelWriteErrorBuilder().Build()
}
