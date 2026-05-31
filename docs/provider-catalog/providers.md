# Provider catalog definitions

Provider catalog entries are TOML files in `src/providers/catalog/definitions`.
Use one definition file for each API provider, gateway, or local endpoint.

A top-level provider is the company or service that self-publishes a model
family. Examples: OpenAI publishes GPT and o-series models, Anthropic publishes
Claude and Opus/Sonnet models, Google publishes Gemini models, and DeepSeek
publishes DeepSeek models.

Gateways and hosted proxies do not own those global model families. They expose
routes to models owned by top-level providers, so their catalog entries should
usually reference global models with `globalModelId`.

## Pick The Right File

| What you're adding | Files to touch | Brand/provider examples |
|--------------------|----------------|-------------------------|
| Top-level provider with no new global models | `definitions/<provider>.toml` | OpenAI after GPT models already exist, Anthropic after Claude models already exist |
| Top-level provider with owned global models | `definitions/<provider>.toml` and `models/<family>.toml` | OpenAI + GPT/o-series, Anthropic + Claude/Opus, Google + Gemini, DeepSeek + DeepSeek |
| Gateway or hosted proxy | `definitions/<gateway>.toml` | OpenRouter, OpenGateway, Groq |
| Local endpoint | `definitions/<local>.toml` | Ollama, LM Studio |

## Recommended Shape

Keep credential env vars at the provider top level:

```toml
credentialEnvVars = ["OPENAI_API_KEY"]
```

The catalog normalizer copies that value into setup, validation, and preset
metadata for runtime consumers. Do not repeat the same env var in every section.

Keep these nested objects inside one TOML block with dotted keys:

```toml
[transportConfig]
kind = "openai-compatible"
authHeader.name = "authorization"
headers.X-Trace-Source = "zero"

[catalog]
source = "hybrid"
discovery.kind = "openai-compatible"

[preset]
id = "my-provider"
badge.text = "HOSTED"
```

Use provider-level `costCurrency` for pricing currency. Individual provider
model routes can define numeric `cost.*` fields, but global model files should
not define pricing.

## Required Provider Fields

Every definition needs these top-level fields:

| Field | Purpose |
|-------|---------|
| `id` | Stable provider ID. |
| `name` | Human-readable provider name. |
| `kind` | `provider`, `gateway`, or `localhost`. |
| `description` | Short provider description. |
| `baseURL` | Provider API base URL. |
| `defaultModel` | Model used when the user does not choose one. |

## Common Optional Fields

| Field | Purpose |
|-------|---------|
| `category` | `local`, `hosted`, or `aggregating`. |
| `vendorId` | Inherit another provider's `transportConfig` when this definition omits one. |
| `isFirstParty` | Auto-add owned global models from `models/*.toml`; use this for top-level providers. |
| `supportsModelRouting` | Marks providers that can route multiple model families. |
| `apiKeyLabel` / `apiKeyPlaceholder` | UI labels for credential entry. |
| `apiKeyRequired` | `false` for local/no-auth providers; otherwise defaults from `setup.requiresAuth`. |
| `credentialEnvVars` | Canonical env vars that may contain provider credentials. |
| `costCurrency` | Currency code for provider-level model pricing, such as `USD`. |

## Setup

Use setup for credential flow metadata:

```toml
[setup]
requiresAuth = true
authMode = "api-key"
```

| Field | Required | Accepted values or behavior |
|-------|----------|-----------------------------|
| `requiresAuth` | Yes when `setup` is present | `true` for credentialed providers; `false` for no-auth local endpoints. |
| `authMode` | Yes when `setup` is present | `api-key`, `oauth`, `adc`, `token`, or `none`. |
| `credentialEnvVars` | No | Normally omit this and use top-level `credentialEnvVars`. |
| `setupPrompt` | No | Optional credential setup prompt text. |

## Validation

Use validation when Zero should check credentials or match configured provider
URLs:

```toml
[validation]
kind = "credential-env"
missingCredentialMessage = "An API key is required."
matchBaseUrlHosts = ["api.example.com"]
```

| Field | Required | Accepted values or behavior |
|-------|----------|-----------------------------|
| `kind` | Yes when `validation` is present | Currently `credential-env`. |
| `credentialEnvVars` | No | Normally omit this and use top-level `credentialEnvVars`. |
| `missingCredentialMessage` | No | Message shown when required credentials are missing. |
| `matchBaseUrlHosts` | No | Hostnames used to match an existing provider config. |

## Transport

OpenAI-compatible transports are supported at runtime. Anthropic-compatible
descriptors are recognized by the catalog, but runtime creation currently
rejects them until that transport is implemented.

```toml
[transportConfig]
kind = "openai-compatible"
maxTokensField = "max_completion_tokens"
removeBodyFields = ["store", "stream_options"]
authHeader.name = "authorization"
authHeader.scheme = "bearer"
```

| Field | Required | Accepted values or behavior |
|-------|----------|-----------------------------|
| `kind` | Yes when `transportConfig` is present | `openai-compatible` or `anthropic-compatible`. |
| `headers.*` | No | Static headers. Values beginning with `$` are read from `process.env`; unset env vars are omitted. |
| `authHeader.name` | No | Header name used for API key auth. Defaults to `authorization` during discovery. |
| `authHeader.scheme` | No | `bearer` sends `Bearer <key>`; `raw` sends the key value directly. Defaults to `bearer`. |
| `maxTokensField` | No | `max_tokens` or `max_completion_tokens`. |
| `removeBodyFields` | No | Request body fields stripped before sending to the provider. |
| `endpointPath` | No | Optional transport endpoint path metadata. |

For runtime OpenAI-compatible calls, the OpenAI SDK supplies its normal
`Authorization` header unless a custom raw or non-Authorization `authHeader` is
configured. For model discovery, Zero builds fetch headers directly and sends
the configured auth header when discovery requires auth.

## Catalog And Discovery

Use `catalog` for static model routes, dynamic discovery, or both:

```toml
[catalog]
source = "hybrid"
discoveryCacheTtl = "1h"
discoveryRefreshMode = "manual"
allowManualRefresh = true
discovery.kind = "openai-compatible"
discovery.requiresAuth = true
discovery.path = "/models"
```

| Field | Required | Accepted values or behavior |
|-------|----------|-----------------------------|
| `source` | Yes when `catalog` is present | `static`, `dynamic`, or `hybrid`. |
| `models` | No | Static `[[catalog.models]]` entries. These are returned before discovered models. |
| `discovery.kind` | No | `openai-compatible`, `ollama`, or `custom`; only OpenAI-compatible and Ollama discovery execute today. |
| `discovery.path` | No | Discovery path appended to `baseURL`; defaults to `/models`. |
| `discovery.requiresAuth` | No | `false` skips auth headers and cache key partitioning by API key. |
| `discoveryCacheTtl` | No | Milliseconds as a number, or a string ending in `m`, `h`, or `d`; defaults to `1h`. |
| `discoveryRefreshMode` | No | `manual`, `on-open`, `background-if-stale`, or `startup` metadata. |
| `allowManualRefresh` | No | Metadata for UI refresh affordances. |

OpenAI-compatible discovery accepts either `{ data: [{ id: "..." }] }` or a raw
array of model IDs/objects. Ollama discovery calls `/api/tags` on the base URL
without the `/v1` suffix and reads `models[].name`.

## Usage And Presets

Use `usage` for usage-accounting metadata:

```toml
[usage]
supported = false
silentlyIgnore = true
```

| Usage field | Required | Accepted values or behavior |
|-------------|----------|-----------------------------|
| `supported` | Yes when `usage` is present | Whether usage accounting is expected for this provider. |
| `silentlyIgnore` | No | Suppress usage-accounting noise when unsupported. |

Use `preset` for setup fallbacks and display metadata:

```toml
[preset]
id = "my-provider"
description = "My Provider hosted models"
fallbackBaseUrl = "https://api.example.com/v1"
fallbackModel = "vendor/model-name"
badge.text = "HOSTED"
badge.color = "success"
```

| Preset field | Required | Accepted values or behavior |
|--------------|----------|-----------------------------|
| `id` | Yes when `preset` is present | Stable preset ID. |
| `description` | Yes when `preset` is present | Preset description. |
| `label` | No | Optional UI label. |
| `apiKeyEnvVars` | No | Normally omit this and use top-level `credentialEnvVars`. |
| `fallbackBaseUrl` | No | Base URL used when config does not provide one. |
| `fallbackModel` | No | Model used when config does not provide one. |
| `badge.text` / `badge.color` | No | Optional display badge metadata. |
