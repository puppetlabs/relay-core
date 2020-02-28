package resolve

var (
	NoOpSecretTypeResolver    SecretTypeResolver    = ChainSecretTypeResolvers()
	NoOpOutputTypeResolver    OutputTypeResolver    = ChainOutputTypeResolvers()
	NoOpParameterTypeResolver ParameterTypeResolver = ChainParameterTypeResolvers()
	NoOpAnswerTypeResolver    AnswerTypeResolver    = ChainAnswerTypeResolvers()
)
