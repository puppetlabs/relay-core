package request

import "github.com/pborman/uuid"

type Request struct {
	Identifier string
}

func New() *Request {
	return &Request{
		Identifier: uuid.New(),
	}
}
