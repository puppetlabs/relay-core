package auth

type Auth struct {
	WorkflowName string `json:"workflow_name"`
	TaskName     string `json:"task_name"`
	Token        string `json:"token"`
}
