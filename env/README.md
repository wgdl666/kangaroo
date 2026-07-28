# env

`env` reads the three wgdl platform process environment variables:

| Variable | Meaning | Example |
|----------|---------|---------|
| `XX_WG_PSM` | Product / Service Module | `wg.mirror.hub` |
| `XX_WG_ENV` | Deployment environment (`prod` or `ppe_*`) | `ppe_mirror_zby` |
| `XX_WG_REGION` | Deploy region | `CN` |

No runtime dependencies.

## Install

```sh
go get github.com/wgdl666/kangaroo/env
```

## Usage

```go
package main

import (
	"fmt"
	"log"

	"github.com/wgdl666/kangaroo/env"
)

func main() {
	v, err := env.Load()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(v.PSM, v.Env, v.Region)
	fmt.Println(v.IsPPE(), v.IsProd())
}
```

## API

- `Load()` reads and validates the three variables.
- `MustLoad()` panics if validation fails.
- `Vars.Validate()` / `IsProd()` / `IsPPE()` helpers.
- Constants: `KeyPSM`, `KeyEnv`, `KeyRegion`, `EnvProd`, `EnvPPEPrefix`.
