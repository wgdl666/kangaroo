package bizerror

import (
	"errors"
	"strings"
	"testing"
)

func TestBizError(t *testing.T) {
	internalError := NewBuilder[int64]().
		SetCode(59144).
		SetMessage("internal error").
		SetStable(false).
		Build()

	if internalError.Message() != "internal error" {
		t.Fatalf("Message() = %q, want %q", internalError.Message(), "internal error")
	}

	// cloned
	internalErrorCloned := internalError.WithMessage("hello")
	if internalErrorCloned.Stable() {
		t.Fatal("Stable() = true, want false")
	}
	if internalErrorCloned.Stack() != "" {
		t.Fatalf("Stack() = %q, want empty", internalErrorCloned.Stack())
	}

	bizErr := testBizErrorStack1(1)
	assertMessage(t, bizErr, "Stack1")
	assertHasStack(t, bizErr)

	bizErr = testBizErrorStack1(2)
	assertMessage(t, bizErr, "Stack2")
	assertHasStack(t, bizErr)

	bizErr = testBizErrorStack1(3)
	assertMessage(t, bizErr, "Stack3")
	assertHasStack(t, bizErr)

	bizErr = testBizErrorStack1(4)
	assertMessage(t, bizErr, "internal error")
	if bizErr.Stack() != "" {
		t.Fatalf("Stack() = %q, want empty", bizErr.Stack())
	}

	err := testError.Errorf("HelloMoto1")
	if got := errors.Unwrap(err).Error(); got != "HelloMoto1" {
		t.Fatalf("Unwrap().Error() = %q, want %q", got, "HelloMoto1")
	}
	assertContains(t, err.Error(), "code=59144")
	assertContains(t, err.Error(), "message=[internal error]")
	assertContains(t, err.Error(), "cause=[HelloMoto1]")
	assertContains(t, err.Error(), "chain=[HelloMoto1]")

	var bizErrInst BizError
	ok := errors.As(err, &bizErrInst)
	if !ok {
		t.Fatal("errors.As() = false, want true")
	}

	err = bizErrInst.Errorf("HelloMoto2")
	assertContains(t, err.Error(), "cause=[HelloMoto1]")
	assertContains(t, err.Error(), "chain=[HelloMoto1|HelloMoto2]")
}

var testError = NewBuilder[int64]().
	SetCode(59144).
	SetMessage("internal error").
	SetStable(false).
	Build()

func testBizErrorStack1(stackLevel int) BizError {
	bizErr := testBizErrorStack2(stackLevel)
	if stackLevel == 1 {
		return bizErr.WithMessageAndStack("Stack1")
	}
	return bizErr
}

func testBizErrorStack2(stackLevel int) BizError {
	bizErr := testBizErrorStack3(stackLevel)
	if stackLevel == 2 {
		return bizErr.WithMessageAndStack("Stack2")
	}
	return bizErr
}

func testBizErrorStack3(stackLevel int) BizError {
	bizErr := testError
	if stackLevel == 3 {
		return testError.WithMessageAndStack("Stack3")
	}
	return bizErr
}

func assertMessage(t *testing.T, err BizError, want string) {
	t.Helper()
	if got := err.Message(); got != want {
		t.Fatalf("Message() = %q, want %q", got, want)
	}
}

func assertHasStack(t *testing.T, err BizError) {
	t.Helper()
	if err.Stack() == "" {
		t.Fatal("Stack() is empty")
	}
}

func assertContains(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("%q does not contain %q", got, want)
	}
}
