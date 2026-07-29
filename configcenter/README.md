# configcenter

Config Center 客户端：只提供两个泛型拉取接口。

- `GetConfig[T]` → 应用配置（settings）解码为 `T`
- `GetSecrets[T]` → 密钥配置（secrets）解码为 `T`

每次拉取自动使用当前 [`kangaroo/env`](../env) 的 `PSM` / `Env` / `Region`；`ppe_*` 无配置时回退同一 PSM 的 `prod`。若 bundle 含 `xx_wg.region`，会与 `env.Region` 校验一致。

## Install

```sh
go get github.com/wgdl666/kangaroo/configcenter
```

## Usage

```go
package main

import (
	"context"
	"log"

	"github.com/wgdl666/kangaroo/configcenter"
	"github.com/wgdl666/kangaroo/env"
)

type AppConfig struct {
	Port int `json:"port"`
	XXWG struct {
		Region string `json:"region"`
	} `json:"xx_wg"`
}

type AppSecrets struct {
	PostgresDSN string `json:"POSTGRES_DSN"`
	S3AccessKey string `json:"S3_ACCESS_KEY"`
}

func main() {
	env.MustInit()
	configcenter.MustInit(configcenter.Options{
		BaseURL: "http://127.0.0.1:8096",
	})

	cfg, err := configcenter.GetConfig[AppConfig](context.Background())
	if err != nil {
		log.Fatal(err)
	}
	secrets, err := configcenter.GetSecrets[AppSecrets](context.Background())
	if err != nil {
		log.Fatal(err)
	}
	log.Println(cfg.Port, secrets.PostgresDSN)
}
```

也可用 `GetConfig[map[string]any]` / `GetSecrets[map[string]string]`。

## API

| Func | 说明 |
|------|------|
| `MustInit(Options)` | 设置 Config Center 地址（及可选 Token）；失败 panic |
| `GetConfig[T](ctx)` | 拉取并解码应用配置 |
| `GetSecrets[T](ctx)` | 拉取并解码密钥配置 |
