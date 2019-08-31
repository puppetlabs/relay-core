package main

import "errors"

var (
	ErrAuthNotSupported          = errors.New("dial: server does not support AUTH")
	ErrNoAuthMechanismsAvailable = errors.New("dial: server does not support any common authentication mechanisms (want CRAM-MD5, PLAIN, or LOGIN)")
	ErrSTARTTLSNotSupported      = errors.New("dial: server does not support STARTTLS")
)
