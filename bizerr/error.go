package bizerr

import (
	"errors"
	"fmt"
)

type Error struct {
	code    int32
	message string
	cause   error
	stack   []byte
}

func New(code int32, message string) *Error {
	return &Error{code: code, message: message, stack: stack()}
}

func Errorf(code int32, format string, args ...any) *Error {
	return New(code, fmt.Sprintf(format, args...))
}

func Wrap(code int32, err error, message string) *Error {
	if err == nil {
		return New(code, message)
	}
	if message == "" {
		message = err.Error()
	}
	return &Error{code: code, message: message, cause: err, stack: stack()}
}

func Wrapf(code int32, err error, format string, args ...any) *Error {
	return Wrap(code, err, fmt.Sprintf(format, args...))
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.cause != nil {
		return fmt.Sprintf("code=%d message=%s: %v", e.code, e.message, e.cause)
	}
	return fmt.Sprintf("code=%d message=%s", e.code, e.message)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

func (e *Error) Code() int32 {
	if e == nil {
		return 0
	}
	return e.code
}

func (e *Error) Message() string {
	if e == nil {
		return ""
	}
	return e.message
}

func (e *Error) Stack() string {
	if e == nil {
		return ""
	}
	return string(e.stack)
}

func As(err error) (*Error, bool) {
	var target *Error
	if errors.As(err, &target) {
		return target, true
	}
	return nil, false
}

func Code(err error) (int32, bool) {
	if e, ok := As(err); ok {
		return e.Code(), true
	}
	return 0, false
}

func Message(err error) string {
	if e, ok := As(err); ok {
		return e.Message()
	}
	if err == nil {
		return ""
	}
	return err.Error()
}

func Stack(err error) string {
	if e, ok := As(err); ok {
		return e.Stack()
	}
	return ""
}
