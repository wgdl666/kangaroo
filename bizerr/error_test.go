package bizerr

import (
	"errors"
	"strings"
	"testing"
)

func TestNewCarriesCodeMessageAndStack(t *testing.T) {
	err := New(50001, "internal error")
	if err.Code() != 50001 {
		t.Fatalf("Code() = %d, want 50001", err.Code())
	}
	if err.Message() != "internal error" {
		t.Fatalf("Message() = %q, want internal error", err.Message())
	}
	if err.Stack() == "" {
		t.Fatal("Stack() is empty")
	}
}

func TestErrorfStackStartsAtCaller(t *testing.T) {
	// 格式化错误的首帧必须指向实际业务调用点，不能暴露 bizerr 内部的 Errorf。
	err := Errorf(50001, "dependency %s", "failed")
	if !strings.Contains(err.Stack(), "TestErrorfStackStartsAtCaller") {
		t.Fatalf("Stack() = %q, want caller frame", err.Stack())
	}
	if strings.Contains(err.Stack(), "bizerr.Errorf") {
		t.Fatalf("Stack() = %q, contains internal Errorf frame", err.Stack())
	}
}

func TestWrapKeepsCause(t *testing.T) {
	cause := errors.New("db timeout")
	err := Wrap(50002, cause, "create order failed")
	if !errors.Is(err, cause) {
		t.Fatal("errors.Is(wrapped, cause) = false, want true")
	}
	if code, ok := Code(err); !ok || code != 50002 {
		t.Fatalf("Code() = %d, %v; want 50002, true", code, ok)
	}
	if Message(err) != "create order failed" {
		t.Fatalf("Message() = %q, want create order failed", Message(err))
	}
	if !strings.Contains(err.Error(), "db timeout") {
		t.Fatalf("Error() = %q, want cause text", err.Error())
	}
}
