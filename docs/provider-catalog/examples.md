# Provider catalog examples

Top-level providers self-publish model families, such as OpenAI with
GPT/o-series, Anthropic with Claude/Opus, Google with Gemini, and DeepSeek with
DeepSeek models. Gateways and local endpoints normally reference those global
models with `globalModelId`.

## Gateway with discovery and curated models

```toml
id = "my-gateway"
name = "My Gateway"
kind = "gateway"
description = "OpenAI-compatible model gateway"
category = "aggregating"
baseURL = "https://api.example.com/v1"
defaultModel = "vendor/model-name"
supportsModelRouting = true
apiKeyLabel = "My Gateway API key"
apiKeyPlaceholder = "gw_live_..."
apiKeyRequired = true
credentialEnvVars = ["MY_GATEWAY_API_KEY"]
costCurrency = "USD"

[setup]
requiresAuth = true
authMode = "api-key"

[validation]
kind = "credential-env"
missingCredentialMessage = "An API key is required."
matchBaseUrlHosts = ["api.example.com"]

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
allowManualRefresh = true
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

[[catalog.models]]
globalModelId = "claude-sonnet-4"
apiName = "anthropic/claude-sonnet-4"
contextWindow = 200000
maxOutputTokens = 64000
cost.inputPerMillion = 3.00
cost.outputPerMillion = 15.00

[preset]
id = "my-gateway"
description = "My Gateway hosted models"
fallbackBaseUrl = "https://api.example.com/v1"
fallbackModel = "vendor/model-name"
badge.text = "HOSTED"
badge.color = "success"
```

## Local Ollama-style provider

```toml
id = "ollama"
name = "Ollama"
kind = "localhost"
description = "Local models via Ollama"
category = "local"
baseURL = "http://localhost:11434/v1"
defaultModel = "llama3.2"
apiKeyRequired = false

[setup]
requiresAuth = false
authMode = "none"

[startup]
autoDetectable = true
probeReadiness = "openai-compatible-models"

[transportConfig]
kind = "openai-compatible"

[catalog]
source = "dynamic"
discoveryCacheTtl = "5m"
discoveryRefreshMode = "on-open"
allowManualRefresh = true
discovery.kind = "ollama"
discovery.requiresAuth = false

[[catalog.models]]
id = "llama3.2"
name = "Llama 3.2"
tier = "local"

[[catalog.models]]
id = "mistral"
name = "Mistral"
tier = "local"
```

## First-party provider

```toml
id = "openai"
name = "OpenAI"
kind = "provider"
description = "OpenAI direct API"
category = "hosted"
baseURL = "https://api.openai.com/v1"
defaultModel = "gpt-4o"
isFirstParty = true
apiKeyRequired = true
credentialEnvVars = ["OPENAI_API_KEY"]
costCurrency = "USD"

[setup]
requiresAuth = true
authMode = "api-key"

[transportConfig]
kind = "openai-compatible"

[catalog]
source = "hybrid"
discoveryCacheTtl = "1h"
discoveryRefreshMode = "manual"
allowManualRefresh = true
discovery.kind = "openai-compatible"
discovery.requiresAuth = true
discovery.path = "/models"

[[catalog.models]]
globalModelId = "gpt-4o"
cost.inputPerMillion = 2.50
cost.outputPerMillion = 10.00

[[catalog.models]]
globalModelId = "gpt-4o-mini"
cost.inputPerMillion = 0.15
cost.outputPerMillion = 0.60
```
