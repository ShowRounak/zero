# Provider catalog definitions

Provider catalog entries are TOML files in `src/providers/catalog/definitions`.
Use one definition file for each first-party provider, gateway, or local
endpoint.

Global model metadata belongs in `src/providers/catalog/models`; gateway and
local definitions should normally reference those models with `globalModelId`
instead of adding new model files.

## File selection

| What you're adding | Files to touch |
|--------------------|----------------|
| First-party provider with no new global models | `definitions/<provider>.toml` |
| First-party provider with owned global models | `definitions/<provider>.toml` and `models/<family>.toml` |
| Gateway or hosted proxy | `definitions/<gateway>.toml` |
| Local endpoint such as Ollama | `definitions/<local>.toml` |

## Top-level provider fields

Every provider definition must include `id`, `name`, `kind`, `description`,
`baseURL`, and `defaultModel`.

Declare credential env vars once with top-level `credentialEnvVars`. The catalog
normalizer copies that value into setup, validation, and preset metadata for
runtime consumers.

| Field | Required | Accepted values or behavior |
|-------|----------|-----------------------------|
| `id` | Yes | Stable provider ID. |
| `name` | Yes | Human-readable provider name. |
| `kind` | Yes | `provider`, `gateway`, or `localhost`. |
| `description` | Yes | Short provider description. |
| `baseURL` | Yes | Provider API base URL. |
| `defaultModel` | Yes | Model used when the user does not choose one. |
| `category` | No | `local`, `hosted`, or `aggregating`. |
| `vendorId` | No | Inherits another provider's `transportConfig` when this definition omits one. |
| `isFirstParty` | No | Adds owned global models from `models/*.toml` to this provider's model list. |
| `supportsModelRouting` | No | Metadata for providers that can route multiple model families. |
| `apiKeyLabel` / `apiKeyPlaceholder` | No | UI labels for credential entry. |
| `apiKeyRequired` | No | `false` for local/no-auth providers; otherwise defaults from `setup.requiresAuth`. |
| `credentialEnvVars` | No | Canonical env vars that may contain provider credentials. |

## Setup

`setup` describes how `/provider` should ask for credentials.

| Field | Required | Accepted values or behavior |
|-------|----------|-----------------------------|
| `requiresAuth` | Yes when `setup` is present | `true` for credentialed providers; `false` for no-auth local endpoints. |
| `authMode` | Yes when `setup` is present | `api-key`, `oauth`, `adc`, `token`, or `none`. |
| `credentialEnvVars` | No | Normally omit this and use top-level `credentialEnvVars`. |
| `setupPrompt` | No | Optional credential setup prompt text. |

## Validation

`validation` currently supports credential-env checks.

| Field | Required | Accepted values or behavior |
|-------|----------|-----------------------------|
| `kind` | Yes when `validation` is present | Currently `credential-env`. |
| `credentialEnvVars` | No | Normally omit this and use top-level `credentialEnvVars`. |
| `missingCredentialMessage` | No | Message shown when required credentials are missing. |
| `matchBaseUrlHosts` | No | Hostnames used to match an existing provider config. |

## Transport config

Keep `transportConfig`, `transportConfig.authHeader`, and
`transportConfig.headers` in one `[transportConfig]` block with dotted keys.

`transportConfig.kind` accepts `openai-compatible` and
`anthropic-compatible`. OpenAI-compatible providers are supported at runtime.
Anthropic-compatible descriptors are recognized by the catalog, but runtime
creation currently rejects them until that transport is implemented.

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

## Model catalog and discovery

Keep `catalog` and `catalog.discovery` in one `[catalog]` block with dotted
`discovery.*` keys.

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

## Usage and presets

`usage` is provider metadata. Set `supported = false` for providers where usage
accounting should not be expected. `silentlyIgnore = true` suppresses usage
noise for providers such as local runtimes.

| Usage field | Required | Accepted values or behavior |
|-------------|----------|-----------------------------|
| `supported` | Yes when `usage` is present | Whether usage accounting is expected for this provider. |
| `silentlyIgnore` | No | Suppress usage-accounting noise when unsupported. |

Keep `preset` and `preset.badge` in one `[preset]` block with dotted
`badge.*` keys.

| Preset field | Required | Accepted values or behavior |
|--------------|----------|-----------------------------|
| `id` | Yes when `preset` is present | Stable preset ID. |
| `description` | Yes when `preset` is present | Preset description. |
| `label` | No | Optional UI label. |
| `apiKeyEnvVars` | No | Normally omit this and use top-level `credentialEnvVars`. |
| `fallbackBaseUrl` | No | Base URL used when config does not provide one. |
| `fallbackModel` | No | Model used when config does not provide one. |
| `badge.text` / `badge.color` | No | Optional display badge metadata. |
