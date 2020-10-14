package entrypoint

import (
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/util/image"
)

// This follows Kubernetes conventions documented at https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#notes.
// A defined command is considered an override for both the image entrypoint and command.
// More specifically, a defined command would not interact with the image entrypoint as described by
// https://docs.docker.com/engine/reference/builder/#understand-how-cmd-and-entrypoint-interact.
func ImageEntrypoint(img string, command []string, args []string) (*model.Entrypoint, error) {
	var argsForEntrypoint []string

	if len(command) > 0 && len(command[0]) > 0 {
		argsForEntrypoint = append(argsForEntrypoint, model.EntrypointCommandFlag, command[0], model.EntrypointCommandArgSeparator)
		argsForEntrypoint = append(argsForEntrypoint, command[1:]...)
		argsForEntrypoint = append(argsForEntrypoint, args...)
	} else {
		ep, cmd, err := image.ImageData(img)
		if err != nil {
			return nil, err
		}

		if len(ep) > 0 && len(ep[0]) > 0 {
			argsForEntrypoint = append(argsForEntrypoint, model.EntrypointCommandFlag, ep[0], model.EntrypointCommandArgSeparator)
			argsForEntrypoint = append(argsForEntrypoint, ep[1:]...)

			if len(args) > 0 {
				argsForEntrypoint = append(argsForEntrypoint, args...)
			} else {
				argsForEntrypoint = append(argsForEntrypoint, cmd[0:]...)
			}
		} else if len(cmd) > 0 && len(cmd[0]) > 0 {
			argsForEntrypoint = append(argsForEntrypoint, model.EntrypointCommandFlag, cmd[0], model.EntrypointCommandArgSeparator)
			argsForEntrypoint = append(argsForEntrypoint, cmd[1:]...)
			argsForEntrypoint = append(argsForEntrypoint, args...)
		}
	}

	return &model.Entrypoint{
		Entrypoint: model.EntrypointCommand,
		Args:       argsForEntrypoint,
	}, nil
}
