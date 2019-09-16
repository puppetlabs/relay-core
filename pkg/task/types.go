package task

type SpecValueType int

const (
	SpecValueSecret SpecValueType = iota
	SpecValueOutput
)

var SpecValueMapping = map[string]SpecValueType{
	"Secret": SpecValueSecret,
	"Output": SpecValueOutput,
}
