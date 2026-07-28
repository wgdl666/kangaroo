package env

import (
	"os"
	"testing"
)

func withEnv(t *testing.T, psm, environment, region string) {
	t.Helper()
	t.Setenv(KeyPSM, psm)
	t.Setenv(KeyEnv, environment)
	t.Setenv(KeyRegion, region)
}

func TestLoadOK(t *testing.T) {
	withEnv(t, "wg.mirror.hub", "ppe_mirror_zby", "CN")
	v, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if v.PSM != "wg.mirror.hub" || v.Env != "ppe_mirror_zby" || v.Region != "CN" {
		t.Fatalf("got %+v", v)
	}
	if !v.IsPPE() || v.IsProd() {
		t.Fatalf("ppe flags: IsPPE=%v IsProd=%v", v.IsPPE(), v.IsProd())
	}
}

func TestLoadProd(t *testing.T) {
	withEnv(t, "wg.mirror.hub", "prod", "CN")
	v, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !v.IsProd() || v.IsPPE() {
		t.Fatalf("prod flags: IsProd=%v IsPPE=%v", v.IsProd(), v.IsPPE())
	}
}

func TestLoadMissing(t *testing.T) {
	os.Clearenv()
	if _, err := Load(); err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadInvalidEnv(t *testing.T) {
	withEnv(t, "wg.mirror.hub", "staging", "CN")
	if _, err := Load(); err == nil {
		t.Fatal("expected invalid env error")
	}
}

func TestMustLoadPanics(t *testing.T) {
	os.Clearenv()
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	_ = MustLoad()
}
