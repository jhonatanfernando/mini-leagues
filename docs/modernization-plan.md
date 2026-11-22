# Mini-leagues modernization plan

Status: implementation in progress, November 15, 2026.

## Podium Picks application milestone

The [playable milestone contract](podium-picks-milestone.md) now defines the game
API, permissions, storage, and acceptance gates. Pure podium validation/scoring
and the web scaffold exist. [Cognito foundation](cognito-setup.md) is deployed. The Go handlers require JWT
identity, membership for reads, and a configured organizer for creation. Terraform
JWT routes, membership configuration, and scoped permissions have been applied;
a follow-up plan found no changes. Live authorization tests remain outstanding.
The browser login remains a follow-up. This feature work does not mark outstanding operational phases complete.

## Progress audit (source only)

- Phase 1 implemented: small DynamoDB interface, offline handler/router/response
  tests, local check/package scripts, dependency classification, and API/setup
  documentation. The placeholder module path remains pending confirmed identity.
- Phase 2: `docs/s3-state-migration.md` records completed migration and remote
  verification of both roots. No post-migration plan or lock-contention test is
  recorded; these checks still require their own authorization.
- Phase 3 configuration prepared: validated settings, common provider tags,
  credential-chain authentication, provider account guard, and a local variable
  example. Default names/settings and all resource addresses are preserved.
  Formatting and offline validation pass; no plan or apply was run. An explicitly
  authorized plan must verify intended tag updates and absence of resource churn.
  Both tables and their recovery settings remain unchanged pending a retention
  decision. Inspector trust/read scope was reviewed in source and preserved;
  continued need for the role and caller-policy review remain outstanding.
- Phase 4 request contract implemented with tests written before enforcement:
  16 KiB decoded bodies (413), one UTF-8 JSON object, unknown-field rejection,
  trimmed 128-byte identifiers and 200-code-point names (400). README documents
  compatibility changes, decoder field matching, and unchanged success/conflict
  semantics. The developer-only access policy explicitly treats ownerId as
  metadata; authenticated ownership remains required before multiuser exposure.
  Request correlation is still pending with Phase 5 structured logging.
  Go checks and Linux ZIP verification pass; deployment has not been performed.
- Phase 5 remains: structured logging, access logs, alert destinations/thresholds,
  budget settings, and operational runbooks.
- Phase 6 remains: credential-free CI and reviewed artifact/plan delivery.

The findings table below records the original baseline, rather than current
completion. This update makes no live AWS or Terraform-state claims.

## Direction

Keep Go Lambda + API Gateway HTTP API + DynamoDB + Terraform. Establish safe state management, tests, repeatable builds, and operational visibility before adding features. Use small changes with a clear acceptance check; preserve existing Terraform addresses and deployed resource names.

Keep one development environment initially. Introduce another environment only when needed, with separate state and resource names. Avoid premature microservices, frameworks, a VPC/NAT gateway, or a generic repository layer.

## Findings from the repository

| Area | Current state | Proposed improvement |
| --- | --- | --- |
| Terraform state | No backend declared; configuration defaults to local state | Dedicated S3 backend with locking and recovery procedures |
| Existing S3 bucket | `aws_s3_bucket.learning` is part of the app root | Preserve it; use a separate state bucket with its own lifecycle |
| Lambda | Four domain/startup/router/response files; client initialized once | Keep this layout; inject a small DynamoDB interface for tests |
| API | Create/get league, IAM authorization, conditional create, strong reads | Preserve behavior first, then define validation and ownership rules |
| Tests | No test files tracked | Focused offline handler, routing, and response tests |
| Dependencies | Go 1.26.8, AWS SDK v2, committed dependency locks | Align local/CI toolchains; tidy dependency classification |
| Terraform tooling | Requires Terraform >=1.17.0; AWS provider ~>6.0 | Record tested CLI version; retain provider lock and review upgrades |
| Configuration | Region, profile, names and settings embedded in Terraform | Parameterize environment settings without renaming existing resources |
| Security | Scoped execution role; separate inspector role | Preserve separation; review inspector trust and caller permissions |
| Operations | Seven-day Lambda logs, API throttling | Structured logs, API access logs, actionable alarms and cost alerts |
| Deployment | Manual packaging and Terraform commands | One documented check/build workflow, offline CI, reviewed deployments |

These are source-code findings, not confirmation of live AWS configuration or the contents/location of authoritative state. Local tools report Go 1.26.8 and Terraform 1.17.0.

## Phase 1 — Repeatable local checks and baseline tests

1. Add a small DynamoDB interface containing only `PutItem` and `GetItem`, matching SDK signatures. Keep client construction in `main.go` and pass request context through unchanged.
2. Write focused failing tests before behavior changes. Cover malformed JSON/base64, missing/blank fields, trimmed values, successful create/get, duplicate create (409), missing league (404), DynamoDB failures (500), item decode failures, unknown routes, and response serialization failures. Assert conditional writes, consistent reads, and context propagation using fakes.
3. Add documented `check` and `package` scripts or Make targets. Include formatting, tests, vet, Linux cross-build, and ZIP root/permission checks. Preserve the Windows instructions and pin the Windows packaging utility instead of using `@latest`.
4. Run `go mod tidy` and review the diff: directly imported modules are currently marked indirect. Retain `go.mod` and `go.sum`. Replace the placeholder module path only when the intended repository/module identity is confirmed.
5. Add an overview, prerequisites, API examples and configuration reference to README. Document signed requests because both routes require AWS IAM.

Acceptance: a clean checkout can check and package the application without AWS credentials; existing API behavior is covered and unchanged. Generated artifacts remain ignored.

## Phase 2 — Protect Terraform state in S3

### Backend design

Create a dedicated `infra/bootstrap/` Terraform root for backend resources, separate from the app root at `infra/`. The existing learning bucket stays at its current address. Keeping backend resources separate reduces the chance that application teardown removes its own state storage.

Configure the new bucket with:

- A unique account/region-qualified name, versioning, explicit server-side encryption, all S3 public-access blocks, and a policy requiring TLS.
- `force_destroy = false` and Terraform `prevent_destroy` protection. These are guardrails, not a substitute for restricted IAM and recovery documentation.
- S3-native state locking with `use_lockfile = true`. Do not add a DynamoDB lock table; HashiCorp documents DynamoDB locking as deprecated.
- Explicit keys such as `mini-leagues/dev/terraform.tfstate` and `mini-leagues/bootstrap/terraform.tfstate`.
- Least-privilege backend access: bucket listing constrained to the needed prefixes, read/write on state objects, and read/write/delete on the corresponding `.tflock` objects. The Lambda execution role gets no state access.

Start with SSE-S3 to avoid a customer-managed KMS key's additional cost and policy maintenance. Revisit KMS if a specific security requirement calls for it. Supply credentials through the AWS credential chain; keep credentials out of backend files and Terraform variables. Backend authentication is configured independently of the AWS provider.

HashiCorp recommends versioning for recovery and documents native locking and its permissions in the [S3 backend reference](https://developer.hashicorp.com/terraform/language/backend/s3).

### Migration runbook to prepare before execution

1. Identify the authoritative existing state and workspace privately, confirm account/region, and stop concurrent Terraform operations. Do not infer that this checkout holds the latest state. Do not print state or credentials.
2. Create restricted-access backups of the authoritative app state and existing backend configuration outside Git. Record a recovery procedure and retain backups until migration is verified.
3. Prepare and review the bootstrap configuration and a separately authorized plan. Obtain separate authorization before creating the bucket or changing IAM. Initially bootstrap uses local state; protect that temporary state as carefully as the app state.
4. After backend creation, prepare partial S3 backend configuration for the application. With explicit state-migration authorization, run `terraform -chdir=infra init -migrate-state` with the reviewed backend parameters. Do not substitute `-reconfigure`, force-copy, or manual object uploads for a state migration.
5. Verify expected resource addresses and state lineage/serial privately, remote persistence, versioning and locking. With separate plan authorization, confirm the app plan has no migration-induced creates, replacements or destroys. Build the required Lambda ZIP first so its hash can be evaluated.
6. Migrate bootstrap's own state to its separate S3 key under the same approval discipline. Document that backend recovery still depends on retaining access to this bucket.
7. Document recovery using a known-good S3 version under a maintenance lock/window. Never blindly overwrite remote state with an old local copy. Retain local backups securely until verification completes; never commit them.

Do not combine migration with resource renames, provider upgrades, or moving application resources between roots. A failed migration requires reconciling which backend is authoritative before any further writes.

Acceptance: application and bootstrap state persist remotely, locking is enabled, recovery is documented, and reviewed plans show no unintended resource churn.

## Phase 3 — Terraform configuration and infrastructure safeguards

1. Add `variables.tf`, `locals.tf`, backend configuration examples, and consistent project/environment/owner/managed-by tags. Keep `infra/` as the application root. File organization alone does not require modules or change resource addresses.
2. Remove the hardcoded provider profile; select local credentials via `AWS_PROFILE=terraform-dev` and use role-based credentials in CI. Parameterize region with the existing `us-east-2` default and add an account guard to prevent deployments into the wrong account.
3. Parameterize settings that have a practical use: environment, log retention, Lambda memory/timeout, and API throttles. Defaults must preserve existing physical names and behavior.
4. Keep both on-demand DynamoDB tables and their keys. Decide retention/recovery requirements before enabling point-in-time recovery and deletion protection. Do not remove the currently unused members table without checking its data and intended use under separate authorization.
5. Review inspector-role trust, its dependency on `AccountFullAccessRole`, and whether it is still needed. Keep inspector, application caller, Lambda execution, and deployment permissions separate. Do not broaden access as part of cleanup.
6. Preserve `create_league` Terraform labels even though the function now handles multiple routes. If names are later changed for clarity, use reviewed `moved` blocks for address changes; physical AWS renames can still require replacement and should be a separate decision.

Acceptance: format/validation pass; an explicitly authorized plan shows only intended updates. No unexplained replacements or deletions.

## Phase 4 — API contract and application hardening

Retain this structure initially:

```text
lambda/
  main.go          # configuration, dependencies, Lambda startup
  league.go        # league models and handlers
  league_test.go   # domain handler tests and DynamoDB fakes
  router.go
  router_test.go
  response.go
  response_test.go
```

Keep application wiring and its small DynamoDB interface near startup. Add a configuration file only when configuration grows enough to justify it. Extract reusable domain code into `internal/` when a second entry point, such as MCP, actually needs it; at that point `cmd/api` and `cmd/mcp` may become useful. Do not introduce a service/repository stack solely to reorganize two handlers.

Separate contract changes from structural refactoring:

- Specify identifier/name limits, accepted JSON shape, oversized-body handling and unknown-field policy; add tests before enforcement and document compatibility changes.
- Decide ownership semantics. IAM authorization authenticates access to AWS but the current handler trusts client-supplied `ownerId` and does not check league ownership on reads. Before multiple users or real personal data, map authenticated principals to application users and enforce an explicit access policy. Keep IAM auth for current developer use; consider JWT/Cognito when a browser app requires user login.
- Preserve the conditional create and 409 contract initially. It prevents overwrites but does not make retries return the original successful response; introduce an idempotency contract only when needed.
- Keep stable error responses and server-side diagnostic detail. Add request correlation without logging bodies, credentials or authorization headers.

Acceptance: documented request/response examples match tests; ownership limitations are explicit and resolved before multiuser exposure. No unannounced API changes.

## Phase 5 — Observability, reliability and cost controls

1. Use standard-library `log/slog` JSON logging with API/Lambda request IDs, route, outcome, duration and useful error context. Preserve initialization-time SDK client reuse, as recommended in [AWS Lambda best practices](https://docs.aws.amazon.com/lambda/latest/dg/best-practices.html).
2. Add API Gateway access logs with explicit retention and no request bodies/tokens. Keep a documented short retention default for dev.
3. Add a minimal set of actionable alerts: Lambda errors/throttles, API 5xx, and duration near timeout. Choose thresholds and notification destination before creating alarms; document how to investigate them.
4. Add a budget alert and billing review routine. Budgets and API throttles are alerts/controls, not a hard spending cap. Consider reserved concurrency only after evaluating its account-quota and availability effects.
5. Measure before changing Lambda architecture/memory or read consistency. Keep x86_64, 128 MB and strong reads initially; benchmark changes separately with explicit live-test authorization.

Acceptance: an operator can correlate a failed request across API and Lambda logs, knows who receives alerts, and has a short troubleshooting/rollback runbook.

## Phase 6 — CI and reviewed delivery

1. Add credential-free pull-request CI: Go formatting/tests/vet, Linux build/package verification, `go mod verify`, dependency vulnerability checking, and Terraform formatting/validation. Terraform initialization for validation uses `-backend=false`; provide the expected local ZIP before validation. Provider downloads require network access, but CI must not query live AWS.
2. Pin workflow actions to reviewed commit SHAs and tool versions; keep provider/module lock files committed. Use the same build commands locally and in CI. Review dependency/tool updates in small changes.
3. Keep AWS-connected planning separate from offline CI. Only run Terraform plans when explicitly authorized for the task; do not give forked/untrusted PRs AWS credentials. Treat saved plans as sensitive artifacts with restricted access and short retention.
4. If GitHub Actions is selected for deployment, use OIDC with repository/branch/environment-constrained trust and least-privilege roles, rather than stored AWS access keys. Creating its IAM configuration requires separate authorization after review.
5. Serialize deployment jobs per state/environment. Build once, identify the artifact by commit/checksum, review the saved plan and apply that exact plan only after separate authorization. Keep artifacts available for reproducible rollback; do not rebuild between plan and apply.
6. Keep Terraform responsible for code and infrastructure deployment initially. Do not also introduce a direct Lambda updater that competes with Terraform. Document rollback by selecting the prior artifact and reviewing a new plan; Lambda aliases/canaries can follow if release risk justifies them.

Acceptance: PR checks work without AWS access; deployments are traceable to reviewed source, artifact and plan, and cannot run concurrently against one state.

## Expected cost implications

| Addition | Expected implication |
| --- | --- |
| S3 state bucket/versioning | Small storage/request charges at this scale; retained versions accumulate |
| Native S3 locking | S3 request charges; no separate lock table |
| SSE-S3 | Avoids customer-managed KMS key and KMS request charges |
| Access logs/alarms | Additional CloudWatch ingestion, retention and alarm charges; keep volume/retention bounded |
| DynamoDB PITR/backups, if selected | Additional backup storage and recovery charges |
| OIDC/CI | CI usage and artifact retention may consume hosting allowances; no long-lived AWS keys needed |
| Additional environments | Duplicate resources and traffic can increase spending; add only when needed |

These are qualitative expectations, not a priced estimate or a free-tier guarantee. Check current regional prices when preparing each AWS change. Retain the existing serverless design; additional networking and always-on services are unnecessary for the current scope.

## Suggested review units and completion checks

Implement in order: (1) tests/interface, (2) local scripts/docs, (3) backend configuration/runbook, (4) separately approved backend creation and migration, (5) Terraform configuration safeguards, (6) API contract/security changes, (7) logs/alerts, (8) CI and reviewed delivery. Offline CI can move earlier once the local scripts are stable.

For Go changes, run gofmt, `go test ./...`, `go vet ./...`, and `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bootstrap .` from `lambda/`. For Terraform changes, run `terraform -chdir=infra fmt -check` and `terraform -chdir=infra validate`, plus equivalent checks for the bootstrap root. Report blocked checks rather than assuming success.

Planning does not authorize Terraform plans, live AWS tests, apply/destroy, IAM mutations or state migrations. Prepare reviewable configuration first, then obtain the separate authorizations required by AGENTS.md. Do not commit or push unless requested.

Completion means reproducible local/CI checks, protected remote state with a recovery procedure, tested/documented API behavior, scoped credentials, useful operational signals, and a reviewed deployment/rollback process. Production environments, MCP implementation, authentication-provider migration and major data-model changes remain separate feature decisions.
