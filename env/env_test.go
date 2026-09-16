package env

import (
	"os"
	"testing"
)

func withEnv(t *testing.T, serviceName, environment, region string) {
	t.Helper()
	t.Setenv(KeyServiceName, serviceName)
	t.Setenv(KeyEnv, environment)
	t.Setenv(KeyRegion, region)
}

func resetGlobals() {
	ServiceName, Env, Region = "", "", ""
}

func TestMustInitPPE(t *testing.T) {
	resetGlobals()
	withEnv(t, ServiceHub, "ppe_exhibition", "US")
	MustInit()
	if ServiceName != ServiceHub || Env != "ppe_exhibition" || Region != RegionUS {
		t.Fatalf("got ServiceName=%q Env=%q Region=%q", ServiceName, Env, Region)
	}
	if !IsPPE() || IsProd() || IsDev() || !IsUS() {
		t.Fatalf("flags IsPPE=%v IsProd=%v IsDev=%v IsUS=%v", IsPPE(), IsProd(), IsDev(), IsUS())
	}
}

func TestMustInitDev(t *testing.T) {
	resetGlobals()
	withEnv(t, ServiceHub, EnvDev, "CN")
	MustInit()
	if !IsDev() || IsPPE() || IsProd() || !IsCN() {
		t.Fatalf("dev flags: IsDev=%v IsPPE=%v IsProd=%v IsCN=%v", IsDev(), IsPPE(), IsProd(), IsCN())
	}
}

func TestMustInitProd(t *testing.T) {
	resetGlobals()
	withEnv(t, ServiceHub, EnvProd, "CN")
	MustInit()
	if !IsProd() || IsPPE() || IsDev() {
		t.Fatalf("prod flags: IsProd=%v IsPPE=%v IsDev=%v", IsProd(), IsPPE(), IsDev())
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
	withEnv(t, ServiceHub, "staging", "CN")
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	MustInit()
}

func TestMustInitUnknownServicePanics(t *testing.T) {
	resetGlobals()
	withEnv(t, "wg.mirror.hub", EnvDev, "CN")
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	MustInit()
}

func TestMustInitSG(t *testing.T) {
	resetGlobals()
	withEnv(t, ServiceHub, EnvDev, RegionSG)
	MustInit()
	if Region != RegionSG || !IsSG() || IsCN() {
		t.Fatalf("region flags: Region=%q IsSG=%v IsCN=%v", Region, IsSG(), IsCN())
	}
}

func TestMustInitInvalidRegionPanics(t *testing.T) {
	resetGlobals()
	withEnv(t, ServiceHub, EnvProd, "EU")
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	MustInit()
}
