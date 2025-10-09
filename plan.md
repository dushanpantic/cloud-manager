# Implementation Plan: Azure Managed Redis Support

## Goal
Extend the project with support for Azure Managed Redis (new resource type) leveraging existing Azure Cache for Redis implementation patterns (client + control plane + runtime reconcilers).

## Assumptions
- "Azure Managed Redis" is a distinct Azure offering (preview GA) separate from the existing standard Azure Cache for Redis used today in `redisinstance`.
- We will introduce a new Custom Resource (CR) rather than overloading the existing `RedisInstance` kind to avoid breaking changes and to keep provider-specific fields clean.
- Naming convention will follow existing patterns: control plane CRD under `api/cloud-control/v1beta1` and runtime plane under `api/cloud-resources/v1beta1` (mirroring existing redis resources if any). Working name: `AzureManagedRedisInstance` (plural `azuremanagedredisinstances`) unless simplified to `AzureManagedRedis` (choose shorter if consistent with repo conventions after review).
- Reconcile flow will resemble current Azure `redisinstance` flow: create resource, create private endpoint + DNS zone group, wait for availability, update status, handle modifications, deletion ordering.
- Feature toggle may be required (check existing feature flag pattern in `config/featureToggles`). We'll introduce `AzureManagedRedis` flag defaulting to disabled until stabilized.

## High-Level Steps
1. CRD & API types
2. Generated deepcopy / manifests update
3. Client abstraction for Azure Managed Redis
4. Control plane reconciler implementation
5. Runtime (SKR) reconciler (if required for agent-plane syncing or provisioning steps) – mirror existing structure
6. Wiring into manager startup (controllers registration)
7. RBAC additions
8. Feature flag integration
9. Status & conditions conventions
10. Unit / integration tests (state machine actions)
11. Documentation (README, user docs, samples)
12. Makefile / codegen / manifests refresh

## Detailed Steps

### 1. Define API Types
Status: DONE (initial implementation complete)
- Control-plane type `azuremanagedredisinstance_types.go` updated: spec now mirrors Azure portion of existing `RedisInstance` (fields: `remoteRef`, `ipRange`, `scope`, `sku` (using existing `AzureRedisSKU`), optional `redisConfiguration`, `redisVersion`, `shardCount`).
- Status fields added: `state`, `id`, `primaryEndpoint`, `readEndpoint`, `authString`, `caCert`, `conditions`.
- Helper methods implemented (`ScopeRef`, `CloneForPatchStatus`, state setters) consistent with other control-plane resources.
- Runtime-plane type simplified to projection: spec only `ipRange`; status mirrors endpoints/credentials (`id`, `primaryEndpoint`, `readEndpoint`, `authString`, `caCert`, `state`, `conditions`).
- Unused/unsupported concepts intentionally excluded (persistence, TLS config, maintenance window, networking toggle, tags) because Managed Redis offering version targeted either doesn't expose or we defer them. TLS assumed always-on.
- Printcolumns & resource category markers added for control-plane CR.
Remaining (Minor):
- Consider adding short names if desired (e.g., `amr`), not yet defined.
- Revisit validation for shardCount immutability once update semantics clarified.

### 2. Code Generation & Manifests
Status: PARTIALLY DONE (spec/status updated; CRDs need regeneration commit)
- Samples updated with realistic control-plane spec (includes remoteRef, ipRange, scope, sku, redisVersion, basic redisConfiguration) and minimal runtime spec.
- `make generate` executed; `make manifests` was started but previously interrupted manually—run again to ensure CRDs in `config/crd/bases` reflect new schema.
Remaining:
- Let `make manifests` complete (no interruption) and commit regenerated CRDs.
- Optionally add a dedicated shorter sample name if needed.
- Decide on CA bundle injection patches later (not required now).

### 3. Azure Client Abstraction
Status: DONE (scaffolded initial client)
- Implemented new client in `pkg/kcp/provider/azure/client/clientManagedRedis.go` (mirrors existing Redis client; adjust SDK package if Managed Redis uses a different service like `armredisenterprise`).
- Exposes: CreateManagedRedis, GetManagedRedis, UpdateManagedRedis, DeleteManagedRedis, GetManagedRedisAccessKeys.
- Follow-up: Introduce a dedicated aggregated client provider (similar to `redisinstance/client/client.go`) once controller package is added.

### 4. State Object & Factory
Status: PARTIALLY DONE
- Added `pkg/kcp/managedredisinstance/types/state.go` defining managed redis state interface.
- Added `pkg/kcp/provider/azure/managedredisinstance/state.go` with concrete State struct (client plumbing, resource group naming) and `StateFactory` implementation.
- Added `pkg/kcp/provider/azure/managedredisinstance/new.go` skeleton action chain (only finalizer + Stop for now).
Remaining:
- Implement concrete action files (load, create, wait, modify, delete, updateStatus, updateStatusId) mirroring `provider/azure/redisinstance/*` adjusted for Managed Redis client.
- Wire controller to use `managedredisinstance.New` once actions exist (currently placeholder).

### 5. Reconciler (Control Plane)
Status: PARTIALLY DONE
- Action chain skeleton implemented in `pkg/kcp/provider/azure/managedredisinstance/new.go` now composes: add finalizer, load, create, waitAvailable, updateStatus, delete flow with wait + finalizer removal.
- Implemented action files: `loadManagedRedis.go`, `createManagedRedis.go`, `waitManagedRedisAvailable.go`, `updateStatus.go`, `deleteManagedRedis.go`, `waitManagedRedisDeleted.go`.
Remaining:
- Implement modify action (if any mutable fields supported) – currently skipped.
- Add status ID update step (separate from load) if needed; currently ID set during load.
- Integrate endpoints/auth retrieval once Azure Managed Redis client supports keys / connection strings.
- Enhance wait logic with exponential backoff or provisioning state differentiation.
- Add private endpoint / DNS actions if required by architecture (not yet implemented for Managed offering).

### 6. Runtime Plane Reconciler
Status: SCAFFOLD ONLY
- Stub controller exists: `internal/controller/cloud-resources/azuremanagedredisinstance_controller.go`.
Remaining:
- Determine runtime responsibilities (secret distribution, status mirroring) and implement.
- Possibly add feature gating if runtime controller should activate only when needed.
- If runtime plane requires a projection or secret distribution:
  - Create `pkg/skr/azuremanagedredisinstance` with state + reconciler using patterns from `pkg/skr/azureredisinstance` (copy & adapt).
  - Sync endpoints, auth details to runtime CR status or secrets.
  - Ensure finalizers & cleanup replicate existing logic.

### 7. Wire into Aggregating Redis Reconciler (If Applicable)
Status: BASIC WIRING DONE
- `cmd/main.go` registers both control-plane and runtime-plane reconcilers.
Remaining:
- Add feature flag gating around registration.
- Add logger naming consistency & custom options (e.g., max concurrent reconciles) if needed.
- If there is an aggregate reconciler for multiple providers (like `pkg/kcp/redisinstance/reconciler.go`), decide whether Managed Redis is a separate Kind (recommended) thus needing its own dedicated controller registration in `cmd/main.go` rather than altering existing multi-provider switch.
- Register controller in manager setup with appropriate predicates & feature gate checks.

### 8. RBAC
Status: INITIAL DONE
- Generated RBAC for CR access (CRUD + status + finalizers) for both api groups.
Remaining:
- Evaluate need for Secrets, ConfigMaps, or other resource verbs once controller logic defined.
- Ensure Azure credential Secret access (if per-scope secrets introduced) added.
- Add RBAC markers to new controller files (get;list;watch;create;update;patch;delete) for the new CRD.
- Update `config/rbac` with generated role rules for managed redis Azure resources (if using specific Azure secret objects, ensure permissions). Run codegen to refresh aggregated ClusterRole.

### 9. Feature Flag
Status: NOT STARTED
- No flag entry for `AzureManagedRedis` yet.
Remaining:
- Add to `config/featureToggles/flag-schema.json` and default YAMLs.
- Wrap controller registration and possibly action segments.
- Add schema entry in `config/featureToggles/flag-schema.json` for `AzureManagedRedis` (boolean).
- Add default values in `featureToggles.yaml` (false) and `featureToggles.local.yaml` (true for local dev if desired).
- Wrap controller registration and action chain steps with `feature.FlagEnabled` checks (see existing pattern in `redisinstance` or others).

### 10. Status & Conditions
Status: NOT IMPLEMENTED
- Types contain empty status struct; no condition helpers yet.
Remaining:
- Add fields: State enum, endpoints, provisioning details, auth info (if safe), conditions.
- Implement status update actions in controller logic.
- Conditions: Ready, Error, Updating, Deleting.
- Implement helper functions to set / remove conditions (copy from existing RedisInstance logic) ensuring no drift in semantics.
- Populate status endpoints only when resource in succeeded / running state.

### 11. Testing
Status: SCAFFOLD ONLY
- Generated Ginkgo test files for controllers (basic creation/deletion existence test placeholders).
Remaining:
- Add unit tests for state factory, action chain, status transitions, error paths (mock Azure client).
- Consider envtest integration for full reconciliation happy path.
- Unit tests for:
  - State factory initialization with subscription & tenant IDs.
  - Action transitions: create, modify, delete flows (mock Azure client with interfaces).
  - Condition setting utilities.
- Integration (envtest) for CRD registration & basic reconciliation success path using a fake Azure client (no live cloud dependency) under `internal/api-tests` or appropriate test dir.
- Add test data samples.

### 12. Documentation
Status: NOT STARTED
- No README / user docs updates referencing Azure Managed Redis yet.
Remaining:
- Add CR spec documentation, examples, feature flag usage.
- Update `README.md` + user docs under `docs/user` explaining new CR:
  - Spec fields & defaults.
  - Feature flag enabling.
  - Example manifest.
  - Limitations (e.g., unsupported modifications initially).
- Add migration / compatibility note that existing RedisInstance remains unchanged.

### 13. Makefile & Codegen
Status: LIKELY OK / VERIFY
- No explicit Makefile changes observed; standard targets already handle new APIs via kubebuilder annotations.
Remaining:
- Run full build & tests after spec and logic changes; add any missing targets if specialized generation needed.
- Ensure any lists of APIs / controllers in `Makefile` scripts or generation targets include new types.
- Run: `make generate`, `make manifests`, `make build`, `make test-ff` to validate.

### 14. Quality & Release
Status: NOT STARTED
- No changelog or release notes updates.
Remaining:
- Update release artifacts once feature ready & behind flag.
- Verify `golangci-lint` passes (if configured).
- Run full test suite.
- Update `CHANGELOG` or release notes (if project pattern) describing new feature behind flag.

## Edge Cases & Considerations
- Partial creation: if Managed Redis created but private endpoint fails, ensure finalizer cleanup deletes remote resource.
- Modification semantics: some parameters may require recreation; decide to block with Condition status until manual intervention or implement blue/green (future scope).
- Credential rotation: if using default creds, note future enhancement for per-scope credentials.
- Rate limiting / throttling: reuse existing backoff patterns; ensure idempotency of create calls.
- Deletion ordering to avoid orphaned private endpoints / DNS groups.

## Follow-Up (Post-MVP)
- Support scaling operations (shard count changes) with safe update path.
- Parameter patch support with diff logic.
- Metrics & dashboards for Managed Redis usage.
- E2E test in CI with mocked Azure API or sandbox subscription.

## Acceptance Criteria
- New CRD applied successfully (kubectl get shows columns).
- Creating sample CR results in Managed Redis appearing in Azure (when real creds configured) or mocked path passes tests.
- Status transitions to Ready with endpoint info.
- Deleting CR cleans up Azure resources (observed via logs/tests).
- Feature can be toggled off to prevent controller registration.

