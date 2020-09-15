package model

const (
	EntrypointCommand             = "entrypoint"
	EntrypointCommandFlag         = "-entrypoint"
	EntrypointCommandArgSeparator = "--"
)

type Entrypoint struct {
	Entrypoint string
	Args       []string
}
