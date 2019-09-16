package transfer

import (
	"encoding/base64"
	"fmt"
	"strings"
)

type encodingType string

func (p encodingType) String() string {
	return string(p)
}

const (
	Base64EncodingType encodingType = "base64"
	NoEncodingType     encodingType = ""
)

// DefaultEncodingType is the default encodingType. This makes it easier to use this
// package as the caller won't need to make any desisions around what encoder to use
// unless they really need to.
const DefaultEncodingType = Base64EncodingType

// encodingTypeMap is an internal map used to get the encodingType type from a string
var encodingTypeMap = map[string]encodingType{
	"base64": Base64EncodingType,
}

// ParseEncodedValue will attempt to split on : and extract an encoding identifer
// from the prefix of the string. It then returns the discovered encodingType and the
// value without the encodingType prefixed.
func ParseEncodedValue(value string) (encodingType, string) {
	parts := strings.SplitN(value, ":", 2)

	if len(parts) < 2 {
		return NoEncodingType, value
	}

	t, ok := encodingTypeMap[parts[0]]
	if !ok {
		return NoEncodingType, value
	}

	return t, parts[1]
}

// Encoders maps encoding algorithms to their respective EncodeDecoder types.
// Example:
//
//	ed := transfer.Encoders[Base64EncodingType]()
//	encodedValue, err := ed.EncodeForTransfer("my super secret value")
var Encoders = map[encodingType]func() EncodeDecoder{
	Base64EncodingType: func() EncodeDecoder {
		return Base64Encoding{}
	},
	NoEncodingType: func() EncodeDecoder {
		return NoEncoding{}
	},
}

// Base64Encoding handles the encoding and decoding of values using base64.
// All encoded values will be prefixed with "base64:"
type Base64Encoding struct{}

// EncodeForTransfer takes a byte slice and returns it encoded as a base64 string.
// No error is ever returned.
func (e Base64Encoding) EncodeForTransfer(value []byte) (string, error) {
	s := base64.StdEncoding.EncodeToString(value)

	return fmt.Sprintf("%s:%s", Base64EncodingType, s), nil
}

// DecodeFromTransfer takes a string and attempts to decode using a base64 decoder.
// If an error is returned, it will originate from the Go encoding/base64 package.
func (e Base64Encoding) DecodeFromTransfer(value string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(value)
}

// NoEncoding just returns the values without encoding them. This is used when there
// is no encoding type algorithm prefix on the value.
type NoEncoding struct{}

// EncodeForTransfer takes a byte slice and casts it to a string. No error is ever
// returned.
func (e NoEncoding) EncodeForTransfer(value []byte) (string, error) {
	return string(value), nil
}

// DecodeFromTransfer takes a string and casts it to a byte slice. No error is ever
// returned.
func (e NoEncoding) DecodeFromTransfer(value string) ([]byte, error) {
	return []byte(value), nil
}

// EncodeForTransfer uses the DefaultEncodingType to encode value.
func EncodeForTransfer(value []byte) (string, error) {
	encoder := Encoders[DefaultEncodingType]()

	return encoder.EncodeForTransfer(value)
}

// DecodeFromTransfer uses ParseEncodedValue to find the right encoder then
// decodes value with it.
func DecodeFromTransfer(value string) ([]byte, error) {
	t, val := ParseEncodedValue(value)
	encoder := Encoders[t]()

	return encoder.DecodeFromTransfer(val)
}
