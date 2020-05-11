package model

import "errors"

var (
	ErrNotFound = errors.New("model: not found")
	ErrRejected = errors.New("model: rejected")
)
