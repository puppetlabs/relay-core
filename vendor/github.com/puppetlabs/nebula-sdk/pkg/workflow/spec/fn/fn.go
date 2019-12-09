package fn

type Descriptor interface {
	Description() string
	PositionalInvoker(args []interface{}) (Invoker, error)
	KeywordInvoker(args map[string]interface{}) (Invoker, error)
}

type DescriptorFuncs struct {
	DescriptionFunc       func() string
	PositionalInvokerFunc func(args []interface{}) (Invoker, error)
	KeywordInvokerFunc    func(args map[string]interface{}) (Invoker, error)
}

var _ Descriptor = DescriptorFuncs{}

func (df DescriptorFuncs) Description() string {
	if df.DescriptionFunc == nil {
		return "<anonymous>"
	}

	return df.DescriptionFunc()
}

func (df DescriptorFuncs) PositionalInvoker(args []interface{}) (Invoker, error) {
	if df.PositionalInvokerFunc == nil {
		return nil, ErrPositionalArgsNotAccepted
	}

	return df.PositionalInvokerFunc(args)
}

func (df DescriptorFuncs) KeywordInvoker(args map[string]interface{}) (Invoker, error) {
	if df.KeywordInvokerFunc == nil {
		return nil, ErrKeywordArgsNotAccepted
	}

	return df.KeywordInvokerFunc(args)
}

type Map interface {
	Descriptor(name string) (Descriptor, error)
}

type funcMap map[string]Descriptor

func (fm funcMap) Descriptor(name string) (Descriptor, error) {
	fd, found := fm[name]
	if !found {
		return nil, ErrFunctionNotFound
	}

	return fd, nil
}

func NewMap(m map[string]Descriptor) Map {
	return funcMap(m)
}
