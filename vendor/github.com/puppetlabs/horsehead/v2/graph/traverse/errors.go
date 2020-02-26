package traverse

import "errors"

var (
	ErrCyclicGraph          = errors.New("traverse: graph is cyclic")
	ErrInvalidFuncSignature = errors.New("traverse: invalid function signature")
)
