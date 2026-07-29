// Package env reads wgdl platform process environment variables (XX_WG_*).
//
// These identify the running Product/Service Module, deployment environment,
// and region — used to address Config Center and branch regional logic.
package env

import (
	"fmt"
	"os"
	"strings"
)

const (
	KeyPSM    = "XX_WG_PSM"
	KeyEnv    = "XX_WG_ENV"
	KeyRegion = "XX_WG_REGION"

	// EnvProd is the sole production environment name.
	EnvProd = "prod"
	// EnvPPEPrefix marks online test (PPE) environments, e.g. ppe_mirror_zby.
	EnvPPEPrefix = "ppe_"

	RegionCN = "CN"
	RegionSG = "SG"
)

// Package-level platform env, populated by MustInit.
var (
	PSM    string // XX_WG_PSM, e.g. wg.mirror.hub
	Env    string // XX_WG_ENV: prod | ppe_*
	Region string // XX_WG_REGION: CN | SG
)

// MustInit reads XX_WG_PSM / XX_WG_ENV / XX_WG_REGION into package globals.
// Panics if any value is missing or invalid.
func MustInit() {
	psm := strings.TrimSpace(os.Getenv(KeyPSM))
	environment := strings.TrimSpace(os.Getenv(KeyEnv))
	region := strings.TrimSpace(os.Getenv(KeyRegion))
	if err := validate(psm, environment, region); err != nil {
		panic(err)
	}
	PSM = psm
	Env = environment
	Region = region
}

// Validate checks package globals against PSM / Env / Region conventions.
func Validate() error {
	return validate(PSM, Env, Region)
}

func validate(psm, environment, region string) error {
	if psm == "" {
		return fmt.Errorf("%s is required", KeyPSM)
	}
	if environment == "" {
		return fmt.Errorf("%s is required", KeyEnv)
	}
	if environment != EnvProd && !strings.HasPrefix(environment, EnvPPEPrefix) {
		return fmt.Errorf("%s must be %q or start with %q (got %q)", KeyEnv, EnvProd, EnvPPEPrefix, environment)
	}
	if region == "" {
		return fmt.Errorf("%s is required", KeyRegion)
	}
	if region != RegionCN && region != RegionSG {
		return fmt.Errorf("%s must be %q or %q (got %q)", KeyRegion, RegionCN, RegionSG, region)
	}
	return nil
}

// IsProd reports whether Env is the production environment.
func IsProd() bool { return Env == EnvProd }

// IsPPE reports whether Env is an online test (ppe_*) environment.
func IsPPE() bool { return strings.HasPrefix(Env, EnvPPEPrefix) }

// IsCN reports whether Region is CN.
func IsCN() bool { return Region == RegionCN }

// IsSG reports whether Region is SG.
func IsSG() bool { return Region == RegionSG }
