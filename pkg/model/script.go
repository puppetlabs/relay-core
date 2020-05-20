package model

import "strings"

const (
	Shebang            = "#!"
	DefaultInterpreter = Shebang + "/bin/sh"
)

func ScriptForInput(input []string) string {
	script := strings.Join(input, "\n")
	if !strings.HasPrefix(script, Shebang) {
		script = DefaultInterpreter + "\n" + script
	}

	return script
}
