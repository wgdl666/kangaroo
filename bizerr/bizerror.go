package bizerror

import (
	"fmt"
	"strings"
)

type BizError interface {
	error
	Errorf(format string, params ...interface{}) error
	Code() int32
	Message() string
	Stable() bool
	Unwrap() error
	Stack() string

	// WithMessage returns Cloned BizError with message.
	WithMessage(message string) BizError
	// WithMessageAndStack returns Cloned BizError with message, and record current stack.
	WithMessageAndStack(message string) BizError

	GetExtraData(key string) any

	SetExtraData(key string, val any) BizError
}

type bizErrorImpl struct {
	code    int32
	stable  bool
	message string

	// chain 错误链
	chain []string

	// cause 起因错误, clone 只在第一次记录.
	cause error
	// stack 记录发生堆栈, clone 只在第一次记录.
	stack []byte

	// extraData 额外数据
	extraData map[string]any
}

func (impl *bizErrorImpl) WithMessage(message string) BizError {
	return impl.withMessage(message)
}

func (impl *bizErrorImpl) WithMessageAndStack(message string) BizError {
	eClone := impl.withMessage(message)
	eClone.stack = stack()
	return eClone
}

func (impl *bizErrorImpl) Error() string {
	return fmt.Sprintf("code=%d, message=[%s], stable=%v cause=[%v] chain=[%s] stack=%v",
		impl.code, impl.message, impl.stable, impl.cause, impl.getChainStr(), impl.Stack(),
	)
}

func (impl *bizErrorImpl) Errorf(format string, params ...interface{}) error {
	eClone := impl.clone()
	// cause 只赋值一次
	e1 := fmt.Errorf(format, params...)
	if eClone.cause == nil {
		eClone.cause = e1
		eClone.stack = stack()
	}
	eClone.chain = append(eClone.chain, fmt.Sprintf("%s", e1))
	return eClone
}

func (impl *bizErrorImpl) withMessage(message string) *bizErrorImpl {
	eClone := impl.clone()
	eClone.message = message
	return eClone
}

func (impl *bizErrorImpl) Stack() string {
	return string(impl.stack)
}

func (impl *bizErrorImpl) Code() int32 {
	return impl.code
}

func (impl *bizErrorImpl) Message() string {
	return impl.message
}

func (impl *bizErrorImpl) Stable() bool {
	return impl.stable
}

func (impl *bizErrorImpl) Unwrap() error {
	return impl.cause
}

func (impl *bizErrorImpl) clone() *bizErrorImpl {
	return &bizErrorImpl{
		code:      impl.code,
		message:   impl.message,
		stable:    impl.stable,
		cause:     impl.cause,
		chain:     impl.chain,
		stack:     impl.stack,
		extraData: impl.extraData,
	}
}

func (impl *bizErrorImpl) getChainStr() string {
	return strings.Join(impl.chain, "|")
}

func (impl *bizErrorImpl) GetExtraData(key string) any {
	return impl.extraData[key]
}

func (impl *bizErrorImpl) SetExtraData(key string, val any) BizError {
	impl.extraData[key] = val
	return impl
}
