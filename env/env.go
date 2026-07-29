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
)

// Package-level platform env, populated by MustInit.
var (
	PSM    string // XX_WG_PSM, e.g. wg.mirror.hub
	Env    string // XX_WG_ENV: prod | ppe_*
	Region string // XX_WG_REGION, e.g. CN
)

// Vars holds the three platform environment variables.
type Vars struct {
	PSM    string // XX_WG_PSM, e.g. wg.mirror.hub
	Env    string // XX_WG_ENV: prod | ppe_*
	Region string // XX_WG_REGION, e.g. CN
}

// MustInit reads XX_WG_PSM / XX_WG_ENV / XX_WG_REGION into package globals.
// Panics if any value is missing or invalid.
func MustInit() {
	v := Vars{
		PSM:    strings.TrimSpace(os.Getenv(KeyPSM)),
		Env:    strings.TrimSpace(os.Getenv(KeyEnv)),
		Region: strings.TrimSpace(os.Getenv(KeyRegion)),
	}
	if err := v.Validate(); err != nil {
		panic(err)
	}
	PSM = v.PSM
	Env = v.Env
	Region = v.Region
}

// Validate checks PSM / Env / Region conventions.
func (v Vars) Validate() error {
	if v.PSM == "" {
		return fmt.Errorf("%s is required", KeyPSM)
	}
	if v.Env == "" {
		return fmt.Errorf("%s is required", KeyEnv)
	}
	if v.Env != EnvProd && !strings.HasPrefix(v.Env, EnvPPEPrefix) {
		return fmt.Errorf("%s must be %q or start with %q (got %q)", KeyEnv, EnvProd, EnvPPEPrefix, v.Env)
	}
	if v.Region == "" {
		return fmt.Errorf("%s is required", KeyRegion)
	}
	return nil
}

// IsProd reports whether Env is the production environment.
func IsProd() bool { return Env == EnvProd }

// IsPPE reports whether Env is an online test (ppe_*) environment.
func IsPPE() bool { return strings.HasPrefix(Env, EnvPPEPrefix) }

// IsProd reports whether Env is the production environment.
func (v Vars) IsProd() bool { return v.Env == EnvProd }

// IsPPE reports whether Env is an online test (ppe_*) environment.
func (v Vars) IsPPE() bool { return strings.HasPrefix(v.Env, EnvPPEPrefix) }
