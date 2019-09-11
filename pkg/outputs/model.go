package outputs

// Output is a model that represents a single data of output
// that a task wanted to make available to following tasks.
type Output struct {
	TaskName string `json:"task_name"`
	Key      string `json:"key"`
	Value    string `json:"value"`
}
