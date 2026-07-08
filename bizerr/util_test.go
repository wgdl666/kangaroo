package bizerror

import (
	"fmt"
	"strings"
	"testing"
)

type testWrapError struct {
	cause   error
	message string
}

func (err *testWrapError) Error() string {
	return err.message
}

func (err *testWrapError) Unwrap() error {
	return err.cause
}

func TestAs(t *testing.T) {
	var bizErr BizError
	var ok bool
	var wrapErr error

	wrapErr = &testWrapError{
		cause:   fmt.Errorf("hello"),
		message: "hello",
	}
	bizErr, ok = As(wrapErr)
	if ok {
		t.Fatal("As() = true, want false")
	}

	wrapErr = &testWrapError{
		cause:   NewBuilder[int32]().SetCode(9527).SetMessage("StephenChow").Build(),
		message: "hello",
	}
	bizErr, ok = As(wrapErr)
	if !ok {
		t.Fatal("As() = false, want true")
	}
	if bizErr.Code() != int32(9527) {
		t.Fatalf("Code() = %d, want %d", bizErr.Code(), int32(9527))
	}
}

func TestChainWith(t *testing.T) {
	var bizErr BizError
	var wrapErr error
	wrapErr = &testWrapError{
		cause:   fmt.Errorf("hello"),
		message: "hello",
	}

	chainErr := ChainWith(wrapErr, "helloKitty=%d", 123)
	_, ok := chainErr.(BizError)
	if ok {
		t.Fatal("ChainWith(non-BizError) returned BizError, want ordinary error")
	}
	assertStringContains(t, chainErr.Error(), "helloKitty=123")

	internalError := NewBuilder[int64]().
		SetCode(59144).
		SetMessage("internal error").
		SetStable(false).Build()

	bizErr, ok = ChainWith(internalError, "hello=%d", 123).(BizError)
	if !ok {
		t.Fatal("ChainWith(BizError) did not return BizError")
	}
	assertStringContains(t, bizErr.Error(), "hello=123")

	bizErr, ok = ChainWith(bizErr, "world=%d", 456).(BizError)
	if !ok {
		t.Fatal("ChainWith(chained BizError) did not return BizError")
	}
	assertStringContains(t, bizErr.Error(), "world=456")
	if bizErr.Code() != int32(59144) {
		t.Fatalf("Code() = %d, want %d", bizErr.Code(), int32(59144))
	}
	if bizErr.Message() != "internal error" {
		t.Fatalf("Message() = %q, want %q", bizErr.Message(), "internal error")
	}
	if bizErr.Stable() {
		t.Fatal("Stable() = true, want false")
	}
}

func TestParseCode(t *testing.T) {
	internalError := NewBuilder[int64]().
		SetCode(59144).
		SetMessage("internal error").
		SetStable(false).Build()

	code, ok := ParseCode(internalError)
	if !ok {
		t.Fatal("ParseCode(BizError) = false, want true")
	}
	if code != int32(59144) {
		t.Fatalf("ParseCode(BizError) code = %d, want %d", code, int32(59144))
	}

	_, ok = ParseCode(fmt.Errorf("hello"))
	if ok {
		t.Fatal("ParseCode(ordinary error) = true, want false")
	}
}

func assertStringContains(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("%q does not contain %q", got, want)
	}
}
