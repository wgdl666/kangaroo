package bizerror

// NewBuilder 业务错误构造.
func NewBuilder[Code Integer]() *Builder[Code] {
	return &Builder[Code]{}
}

// Integer is the set of integer-like types accepted by Builder.
type Integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

type Builder[Code Integer] struct {
	code    int32
	message string
	stable  bool
}

func (builder *Builder[Code]) SetCode(value Code) *Builder[Code] {
	builder.code = int32(value)
	return builder
}

func (builder *Builder[Code]) SetMessage(value string) *Builder[Code] {
	builder.message = value
	return builder
}

func (builder *Builder[Code]) SetStable(value bool) *Builder[Code] {
	builder.stable = value
	return builder
}

func (builder *Builder[Code]) Build() BizError {
	bizErr := &bizErrorImpl{
		code:      builder.code,
		message:   builder.message,
		stable:    builder.stable,
		extraData: make(map[string]any),
	}
	return bizErr
}
