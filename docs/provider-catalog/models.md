# Provider model catalog

Global model metadata lives in `src/providers/catalog/models`. Files use TOML
`[[model]]` entries.

Each model entry must stay as one TOML block. Nested model subtables such as
`[model.cost]`, `[model.capabilities]`, or `[catalog.models.cost]` are not
allowed; use dotted keys inside the `[[model]]` or `[[catalog.models]]` block.

## Global model example

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
cost.inputPerMillion = 2.50
cost.outputPerMillion = 10.00
cost.cachePerMillion = 0.25
cost.currency = "USD"
notes = "Optional maintainer note."
```

## Model fields

Global `[[model]]` entries and provider `[[catalog.models]]` entries share the
same shape. Provider entries may define local models directly, or reference a
global model with `globalModelId` and override route-specific fields.

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
| `cost.inputPerMillion` / `cost.outputPerMillion` | Yes when `cost` is present | Pricing per one million input and output tokens. |
| `cost.cachePerMillion` / `cost.currency` | No | Optional cached-token price and currency code. |
| `transportOverrides.*` | No | Per-model transport overrides merged over provider `transportConfig`. |
| `notes` | No | Free-form maintainer note. |

## Gateway model overrides

When a gateway exposes a global model under a provider-specific name, limit, or
price, keep those route-specific values in that gateway's single
`[[catalog.models]]` block:

```toml
[[catalog.models]]
globalModelId = "claude-sonnet-4"
apiName = "anthropic/claude-sonnet-4"
contextWindow = 200000
maxOutputTokens = 64000
capabilities.supportsReasoning = true
cost.inputPerMillion = 3.00
cost.outputPerMillion = 15.00
cost.cachePerMillion = 0.30
cost.currency = "USD"
transportOverrides.maxTokensField = "max_tokens"
transportOverrides.removeBodyFields = ["store"]
```

`capabilities` flags are `supportsVision`, `supportsStreaming`,
`supportsFunctionCalling`, `supportsJsonMode`, `supportsReasoning`,
`supportsPreciseTokenCount`, `supportsEmbeddings`, and
`supportsTemperature`.

`cost` requires `inputPerMillion` and `outputPerMillion` when present.
`cachePerMillion` and `currency` are optional.

`transportOverrides` accepts the same fields as `transportConfig`. Header
overrides are merged with provider headers, and `removeBodyFields` is unioned
with provider-level removals.
