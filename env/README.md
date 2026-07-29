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

	"github.com/wgdl666/kangaroo/env"
)

func main() {
	env.MustInit()
	fmt.Println(env.PSM, env.Env, env.Region)
	fmt.Println(env.IsPPE(), env.IsProd())
}
```

## API

- `MustInit()` reads and validates the three variables into package globals; panics on failure.
- Globals: `PSM`, `Env`, `Region`.
- `Validate()` / `IsProd()` / `IsPPE()` helpers.
- Constants: `KeyPSM`, `KeyEnv`, `KeyRegion`, `EnvProd`, `EnvPPEPrefix`.
