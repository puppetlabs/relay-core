package fnlib

import (
	"reflect"
	"strconv"
	"time"

	"github.com/puppetlabs/relay-core/pkg/expr/fn"
)

func toString(in interface{}) (string, error) {
	switch v := in.(type) {
	case nil:
		return "", nil
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	case time.Time:
		return v.Format(time.RFC3339Nano), nil
	case int:
		return strconv.FormatInt(int64(v), 10), nil
	case int64:
		return strconv.FormatInt(v, 10), nil
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	case bool:
		return strconv.FormatBool(v), nil
	default:
		return "", &fn.UnexpectedTypeError{
			Wanted: []reflect.Type{
				reflect.TypeOf(nil),
				reflect.TypeOf(""),
				reflect.TypeOf([]byte(nil)),
				reflect.TypeOf(time.Time{}),
				reflect.TypeOf(int(0)),
				reflect.TypeOf(int64(0)),
				reflect.TypeOf(float64(0)),
				reflect.TypeOf(false),
			},
			Got: reflect.TypeOf(v),
		}
	}
}
