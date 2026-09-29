# kangaroo

Shared utility packages and tools for wgdl.

## Packages

- [`bizerr`](./bizerr): typed business errors (code + stack + cause)
- [`callmeta`](./callmeta): shared trace context and business correlation for RPC, HTTP, messages, and persistent tasks.
- [`env`](./env): read platform env vars `XX_WG_SERVICE_NAME` / `XX_WG_ENV` / `XX_WG_REGION`
- [`configcenter`](./configcenter): `GetConfig` / `GetSecrets`（按当前 ServiceName/Env/Region 拉取）
