# zero

To install dependencies:

```bash
bun install
```

To run:

```bash
bun run dev
```

To list built-in provider catalog entries:

```bash
bun run src/index.ts providers catalog
```

## Provider catalog

Provider, gateway, and local endpoint catalog entries live in
`src/providers/catalog/definitions`. Global model metadata lives in
`src/providers/catalog/models`.

Minimal gateway example:

```toml
id = "my-gateway"
name = "My Gateway"
kind = "gateway"
description = "OpenAI-compatible gateway"
category = "aggregating"
baseURL = "https://api.example.com/v1"
defaultModel = "vendor/model-name"
apiKeyRequired = true
credentialEnvVars = ["MY_GATEWAY_API_KEY"]

[setup]
requiresAuth = true
authMode = "api-key"

[transportConfig]
kind = "openai-compatible"
maxTokensField = "max_completion_tokens"
removeBodyFields = ["store", "stream_options"]
authHeader.name = "authorization"
authHeader.scheme = "bearer"
headers.X-Trace-Source = "zero"
headers.X-Tenant = "$MY_GATEWAY_TENANT"

[catalog]
source = "hybrid"
discoveryCacheTtl = "1h"
discoveryRefreshMode = "manual"
discovery.kind = "openai-compatible"
discovery.requiresAuth = true
discovery.path = "/models"

[[catalog.models]]
globalModelId = "gpt-4o"
apiName = "openai/gpt-4o"
contextWindow = 32000
maxOutputTokens = 4096
cost.inputPerMillion = 2.50
cost.outputPerMillion = 10.00
cost.currency = "USD"
```

Detailed docs:

- [Provider definitions](docs/provider-catalog/providers.md)
- [Model catalog entries](docs/provider-catalog/models.md)
- [Examples](docs/provider-catalog/examples.md)

This project was created using `bun init` in bun v1.3.11. [Bun](https://bun.com) is a fast all-in-one JavaScript runtime.
