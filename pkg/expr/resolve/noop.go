package resolve

var (
	NoOpSecretTypeResolver     SecretTypeResolver     = NewMemorySecretTypeResolver(map[string]string{})
	NoOpConnectionTypeResolver ConnectionTypeResolver = NewMemoryConnectionTypeResolver(map[MemoryConnectionKey]interface{}{})
	NoOpOutputTypeResolver     OutputTypeResolver     = NewMemoryOutputTypeResolver(map[MemoryOutputKey]interface{}{})
	NoOpParameterTypeResolver  ParameterTypeResolver  = NewMemoryParameterTypeResolver(map[string]interface{}{})
	NoOpAnswerTypeResolver     AnswerTypeResolver     = NewMemoryAnswerTypeResolver(map[MemoryAnswerKey]interface{}{})
)
