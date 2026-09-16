// Package env reads wgdl platform process environment variables (XX_WG_*).
//
// These identify the running service, deployment environment,
// and region — used to address Config Center and branch regional logic.
package env

import (
	"fmt"
	"os"
	"strings"
)

const (
	KeyServiceName = "XX_WG_SERVICE_NAME"
	KeyEnv         = "XX_WG_ENV"
	KeyRegion      = "XX_WG_REGION"

	EnvProd = "prod"
	EnvDev  = "dev"
	// EnvPPEPrefix marks preview environments; each preview has its own suffix.
	EnvPPEPrefix = "ppe_"

	RegionCN = "CN"
	RegionSG = "SG"
	RegionUS = "US"

	ServiceHub        = "wghub"
	ServiceUserCenter = "user-center"
	ServiceUserMemory = "user-memory"
	ServiceModelHub   = "modelhub"
	ServiceWardrober  = "wardrober"
	ServiceOps        = "wgops"
)

// Package-level platform env, populated by MustInit.
var (
	ServiceName string // XX_WG_SERVICE_NAME
	Env         string // XX_WG_ENV: prod | dev | ppe_*
	Region      string // XX_WG_REGION: CN | SG | US
)

var allowedServices = map[string]struct{}{
	ServiceHub:        {},
	ServiceUserCenter: {},
	ServiceUserMemory: {},
	ServiceModelHub:   {},
	ServiceWardrober:  {},
	ServiceOps:        {},
}

// MustInit reads XX_WG_SERVICE_NAME / XX_WG_ENV / XX_WG_REGION into package globals.
// Panics if any value is missing or invalid.
func MustInit() {
	serviceName := strings.TrimSpace(os.Getenv(KeyServiceName))
	environment := strings.TrimSpace(os.Getenv(KeyEnv))
	region := strings.TrimSpace(os.Getenv(KeyRegion))
	if err := validate(serviceName, environment, region); err != nil {
		panic(err)
	}
	ServiceName = serviceName
	Env = environment
	Region = region
}

// Validate checks package globals against ServiceName / Env / Region conventions.
func Validate() error {
	return validate(ServiceName, Env, Region)
}

func validate(serviceName, environment, region string) error {
	if serviceName == "" {
		return fmt.Errorf("%s is required", KeyServiceName)
	}
	if _, ok := allowedServices[serviceName]; !ok {
		return fmt.Errorf("%s %q is not a known service", KeyServiceName, serviceName)
	}
	if environment == "" {
		return fmt.Errorf("%s is required", KeyEnv)
	}
	if environment != EnvProd && environment != EnvDev && !strings.HasPrefix(environment, EnvPPEPrefix) {
		return fmt.Errorf("%s must be %q, %q, or start with %q (got %q)", KeyEnv, EnvProd, EnvDev, EnvPPEPrefix, environment)
	}
	if region == "" {
		return fmt.Errorf("%s is required", KeyRegion)
	}
	if region != RegionCN && region != RegionSG && region != RegionUS {
		return fmt.Errorf("%s must be %q, %q, or %q (got %q)", KeyRegion, RegionCN, RegionSG, RegionUS, region)
	}
	return nil
}

func IsProd() bool { return Env == EnvProd }
func IsDev() bool  { return Env == EnvDev }

// IsPPE reports whether Env is a preview (ppe_*). Multiple previews may exist.
func IsPPE() bool { return strings.HasPrefix(Env, EnvPPEPrefix) }

func IsCN() bool { return Region == RegionCN }
func IsSG() bool { return Region == RegionSG }
func IsUS() bool { return Region == RegionUS }
