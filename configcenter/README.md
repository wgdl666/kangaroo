# configcenter

Go SDK for pulling published configs from **Config Center**.

Implements the client-side conventions from wgDevLab `XX_WG_ENV.md`:

1. Address bundles with `XX_WG_PSM` + `XX_WG_ENV`
2. If `ppe_*` has no published config → fall back to same PSM's `prod`
3. Poll every **5s** with `If-None-Match` / ETag
4. Optional SDK heartbeat (`POST .../sdk/heartbeat`)

## Install

```sh
go get github.com/wgdl666/kangaroo/configcenter
```

## Environment

| Variable | Required | Meaning |
|----------|----------|---------|
| `XX_WG_PSM` | yes* | Product/Service Module |
| `XX_WG_ENV` | yes* | `prod` or `ppe_*` |
| `XX_WG_REGION` | yes* | Region (e.g. `CN`) |
| `XX_WG_CONFIG_CENTER_URL` | yes† | Config Center base URL |
| `XX_WG_CONFIG_CENTER_TOKEN` | no | SDK bearer (`CONFIG_CENTER_SDK_TOKEN`) |
| `XX_WG_INSTANCE_ID` | no | Heartbeat instance id (default: hostname) |

\* via [`kangaroo/env`](../env) globals (`env.MustInit`) unless you pass `Options.Vars`  
† or `Options.BaseURL`

## Usage

```go
package main

import (
	"context"
	"log"

	"github.com/wgdl666/kangaroo/configcenter"
)

func main() {
	c, err := configcenter.New(configcenter.Options{
		// BaseURL / Token / Vars can also come from XX_WG_* env vars.
		OnUpdate: func(s configcenter.Snapshot) {
			log.Printf("config gen=%d fallback=%v", s.Bundle.Generation, s.Fallback)
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	if err := c.Start(context.Background()); err != nil {
		log.Fatal(err)
	}
	defer c.Stop()

	snap, _ := c.Current()
	region, _ := snap.Setting("xx_wg.region")
	log.Println("region", region)
}
```

## API

- `New(Options)` / `Start` / `Stop` / `Fetch` / `Current`
- `Snapshot.Setting("a.b")` / `Snapshot.Secret(key)`
- `DefaultPollInterval` = 5s
