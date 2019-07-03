package vault

type Config struct {
	Addr string
	// The path to the service account token file
	K8sServiceAccountTokenPath string
	// Optional token if not using kubernetes auth
	Token string
	// The role we should use when logging in
	Role string
	// The workflow we are proxying requests for secrets for
	WorkflowName string
	// The engine to use to form paths from
	EngineMount string
}
