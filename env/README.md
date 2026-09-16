# env

`env` reads the three wgdl platform process environment variables:

| Variable | Meaning | Example |
|----------|---------|---------|
| `XX_WG_SERVICE_NAME` | Service identity | `wghub` |
| `XX_WG_ENV` | `prod`, `dev`, or a preview `ppe_*` | `ppe_exhibition` |
| `XX_WG_REGION` | `CN`, `SG`, or `US` | `US` |

`ppe_` is the preview-class prefix. Multiple previews may exist (`ppe_exhibition`, `ppe_preview`). `IsPPE()` only means the process is in the preview class.

Known service names: `wghub`, `user-center`, `user-memory`, `modelhub`, `wardrober`, `wgops`.

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
	fmt.Println(env.ServiceName, env.Env, env.Region)
	fmt.Println(env.IsPPE(), env.IsProd(), env.IsDev(), env.IsCN(), env.IsSG(), env.IsUS())
}
```

## API

- `MustInit()` reads and validates the three variables into package globals; panics on failure.
- Globals: `ServiceName`, `Env`, `Region`.
- `Validate()` / `IsProd()` / `IsDev()` / `IsPPE()` / `IsCN()` / `IsSG()` / `IsUS()`.
