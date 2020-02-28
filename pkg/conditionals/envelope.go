package conditionals

type ResponseEnvelope struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
