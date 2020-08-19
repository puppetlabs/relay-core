// This content has been partially derived from Tekton
// https://github.com/tektoncd/pipeline

package entrypoint

type Entrypointer struct {
	// Entrypoint is the original specified entrypoint, if any.
	Entrypoint string
	// Args are the original specified args, if any.
	Args []string

	// Runner encapsulates running commands.
	Runner Runner
}

type Runner interface {
	Run(args ...string) error
}

func (e Entrypointer) Go() error {

	if e.Entrypoint != "" {
		e.Args = append([]string{e.Entrypoint}, e.Args...)
	}

	err := e.Runner.Run(e.Args...)
	if err != nil {
		return err
	}

	return nil
}
