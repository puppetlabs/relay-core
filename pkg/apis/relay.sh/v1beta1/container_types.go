package v1beta1

type Container struct {
	// Image is the Docker image to run when this webhook receives an event.
	Image string `json:"image"`

	// Input is the input script to provide to the container.
	//
	// +optional
	Input []string `json:"input,omitempty"`

	// Command is the path to the executable to run when the container starts.
	//
	// +optional
	Command string `json:"command,omitempty"`

	// Args are the command arguments.
	//
	// +optional
	Args []string `json:"args,omitempty"`

	// Spec is the Relay specification to be provided to the container image.
	//
	// +optional
	Spec UnstructuredObject `json:"spec,omitempty"`

	// Env allows environment variables to be provided to the container image.
	//
	// +optional
	Env UnstructuredObject `json:"env,omitempty"`
}
