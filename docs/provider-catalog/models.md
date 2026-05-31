# Provider model catalog

Global model metadata lives in `src/providers/catalog/models`. Files use TOML
`[[model]]` entries. Global models describe identity, ownership, limits, and
capabilities only; all pricing belongs on provider `[[catalog.models]]` route
entries.

Only top-level providers should add global model metadata. A top-level provider
self-publishes the model family, such as OpenAI with GPT/o-series, Anthropic
with Claude/Opus, Google with Gemini, or DeepSeek with DeepSeek models.

Each model entry must stay as one TOML block. Nested model subtables such as
`[model.cost]`, `[model.capabilities]`, or `[catalog.models.cost]` are not
allowed; use dotted keys inside the `[[model]]` or `[[catalog.models]]` block.

## Global model example

Use one `[[model]]` block per global model owned by a top-level provider. Add
additional models by adding additional `[[model]]` blocks in the same family
file.

```toml
[[model]]
id = "gpt-4o"
name = "GPT-4o"
apiName = "gpt-4o"
ownerProviderId = "openai"
tier = "first-party"
description = "OpenAI flagship multimodal model"
classification = ["chat", "vision", "coding"]
contextWindow = 128000
maxOutputTokens = 16384
defaultTemperature = 0.7
temperatureRange.min = 0
temperatureRange.max = 2
capabilities.supportsVision = true
capabilities.supportsStreaming = true
capabilities.supportsFunctionCalling = true
capabilities.supportsJsonMode = true
capabilities.supportsReasoning = false
capabilities.supportsPreciseTokenCount = true
capabilities.supportsEmbeddings = false
capabilities.supportsTemperature = true
notes = "Optional maintainer note."

[[model]]
id = "gpt-4o-mini"
name = "GPT-4o Mini"
apiName = "gpt-4o-mini"
ownerProviderId = "openai"
tier = "first-party"
description = "OpenAI small multimodal model"
classification = ["chat", "vision"]
contextWindow = 128000
maxOutputTokens = 16384
capabilities.supportsVision = true
capabilities.supportsStreaming = true
capabilities.supportsFunctionCalling = true
capabilities.supportsJsonMode = true
capabilities.supportsReasoning = false
```

## Model fields

Global `[[model]]` entries and provider `[[catalog.models]]` entries share the
identity, capability, and limit fields below. Provider entries may also define
route-specific pricing with `cost.*`.

| Field | Required | Accepted values or behavior |
|-------|----------|-----------------------------|
| `id` | Yes for global models; no for provider entries with `globalModelId` | Stable catalog ID. |
| `apiName` | No | Exact model name sent to the API. Defaults to `id`; with `globalModelId`, defaults to the global model ID unless overridden. |
| `globalModelId` | No | Inherits metadata from a global model, then applies provider-entry overrides. |
| `ownerProviderId` | No | First-party provider that owns a global model. Used by `isFirstParty` providers. |
| `name` / `label` | No | Display metadata. |
| `default` / `hidden` | No | Optional selection metadata. |
| `classification` | No | Any of `chat`, `reasoning`, `vision`, and `coding`. |
| `tier` | No | `first-party`, `hosted`, `local`, or `community`. |
| `contextWindow` | No | Input/context token limit for this model or route. |
| `maxOutputTokens` | No | Output token limit for this model or route. |
| `defaultTemperature` | No | Suggested temperature default. |
| `temperatureRange.min` / `temperatureRange.max` | No | Supported temperature bounds. |
| `capabilities.*` | No | Feature flags listed below. |
| `transportOverrides.*` | No | Per-model transport overrides merged over provider `transportConfig`. |
| `notes` | No | Free-form maintainer note. |

## Provider model pricing and overrides

When a provider or gateway exposes a global model under a provider-specific
name, limit, or price, keep those route-specific values in that provider's
single `[[catalog.models]]` block. Put the shared pricing currency once at the
provider top level with `costCurrency`.

```toml
costCurrency = "USD"

[[catalog.models]]
globalModelId = "claude-sonnet-4"
apiName = "anthropic/claude-sonnet-4"
contextWindow = 200000
maxOutputTokens = 64000
capabilities.supportsReasoning = true
cost.inputPerMillion = 3.00
cost.outputPerMillion = 15.00
cost.cachePerMillion = 0.30
transportOverrides.maxTokensField = "max_tokens"
transportOverrides.removeBodyFields = ["store"]

[[catalog.models]]
globalModelId = "deepseek-v3"
apiName = "deepseek/deepseek-chat"
contextWindow = 65536
maxOutputTokens = 8192
cost.inputPerMillion = 0.14
cost.outputPerMillion = 0.28
```

`capabilities` flags are `supportsVision`, `supportsStreaming`,
`supportsFunctionCalling`, `supportsJsonMode`, `supportsReasoning`,
`supportsPreciseTokenCount`, `supportsEmbeddings`, and
`supportsTemperature`.

Provider model `cost` requires `inputPerMillion` and `outputPerMillion` when
present. `cachePerMillion` is optional. Do not put `cost` or `costCurrency` in
global model files.

`transportOverrides` accepts the same fields as `transportConfig`. Header
overrides are merged with provider headers, and `removeBodyFields` is unioned
with provider-level removals.
