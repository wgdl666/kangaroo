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

func resetGlobals() {
	PSM, Env, Region = "", "", ""
}

func TestMustInitOK(t *testing.T) {
	resetGlobals()
	withEnv(t, "wg.mirror.hub", "ppe_mirror_zby", "CN")
	MustInit()
	if PSM != "wg.mirror.hub" || Env != "ppe_mirror_zby" || Region != "CN" {
		t.Fatalf("got PSM=%q Env=%q Region=%q", PSM, Env, Region)
	}
	if !IsPPE() || IsProd() {
		t.Fatalf("ppe flags: IsPPE=%v IsProd=%v", IsPPE(), IsProd())
	}
}

func TestMustInitProd(t *testing.T) {
	resetGlobals()
	withEnv(t, "wg.mirror.hub", "prod", "CN")
	MustInit()
	if !IsProd() || IsPPE() {
		t.Fatalf("prod flags: IsProd=%v IsPPE=%v", IsProd(), IsPPE())
	}
}

func TestMustInitMissingPanics(t *testing.T) {
	resetGlobals()
	os.Clearenv()
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	MustInit()
}

func TestMustInitInvalidEnvPanics(t *testing.T) {
	resetGlobals()
	withEnv(t, "wg.mirror.hub", "staging", "CN")
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	MustInit()
}

func TestMustInitSG(t *testing.T) {
	resetGlobals()
	withEnv(t, "wg.mirror.hub", "prod", RegionSG)
	MustInit()
	if Region != RegionSG || !IsSG() || IsCN() {
		t.Fatalf("region flags: Region=%q IsSG=%v IsCN=%v", Region, IsSG(), IsCN())
	}
}

func TestMustInitInvalidRegionPanics(t *testing.T) {
	resetGlobals()
	withEnv(t, "wg.mirror.hub", "prod", "US")
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	MustInit()
}
