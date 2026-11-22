# Podium Picks (mini-leagues)

A learning project using Go Lambda, API Gateway HTTP API, DynamoDB, and Terraform.
The source API creates and retrieves leagues with JWT-based authorization; the
the Terraform JWT route transition has been applied.

## Podium Picks milestone and sign-in

The [milestone contract](docs/podium-picks-milestone.md) defines target endpoints,
permissions, DynamoDB access patterns, deadline/privacy rules, and acceptance
checks. It distinguishes implemented behavior from future work.

[Cognito setup](docs/cognito-setup.md) describes the new authentication foundation:
email sign-in, classic hosted UI with code + PKCE, a public client, application
scope, and a JWT authorizer attached to the league routes.
Go league reads enforce JWT identity and membership, and creation requires the
configured organizer. Browser login remains a follow-up;
creating a Cognito account alone must never grant access to league data.

## Repository structure

```text
lambda/   Go Lambda API and domain rules (including podium validation/scoring)
web/      TypeScript + Vite browser application
infra/    Terraform application infrastructure; bootstrap/ manages state storage
scripts/  Offline Go checks and Lambda packaging
docs/     Plans and operational runbooks
```

The web app and Lambda build independently. Keep the existing Go module and
Terraform resource addresses in place while the product takes shape; no workspace
manager or additional backend layer is needed. The current API implements league
creation/retrieval; browser sign-in and game endpoints remain future tasks.

## Web development

Install Node.js 24 LTS and npm in your development environment. From the root:

```bash
cd web
npm ci
npm run dev
```

For verification and a static production build, run `npm run build` from `web/`.
Output goes to ignored `web/dist/`. See [web setup](web/README.md) for details.
The initial page is a scaffold, with no login or API requests. Friends cannot
use the application until browser integration and the backend transition are complete.

## Local checks

Use Go 1.26.8 (matching `lambda/go.mod`). Linux/WSL scripts also require Bash,
`zip`, and Python 3. Terraform requires 1.17.0 or newer. No AWS credentials are
needed for application checks or packaging; the first run may download Go modules.

From the repository root:

```bash
scripts/check
scripts/package
```

`check` verifies formatting, runs offline tests, vet and module verification, and
builds Linux amd64 `bootstrap`. `package` builds and verifies the ZIP's root entry,
executable permissions, integrity, and amd64 ELF header. Both disable CGO because
this application needs no C libraries. If your Go cache is read-only, set
`GOCACHE` to a writable directory when running these commands.

## API contract and configuration

`LEAGUES_TABLE_NAME` is required at startup. `MEMBERS_TABLE_NAME` and
`INITIAL_ORGANIZER_SUB` are required for league creation; missing configuration
fails closed. The initial organizer value is the exact Cognito access-token sub.
AWS region and credentials follow the SDK default configuration chain.

Terraform configures JWT routes, membership table configuration, and scoped
permissions. The transition was applied and a follow-up plan found no changes.
No live AWS authorization tests have been run. Set `initial_organizer_sub` to a verified user pool
account’s exact sub after signup; the empty default disables league creation.

- `POST /leagues` requires a verified access-token identity matching the configured
  initial organizer (401 without identity, 403 for another user). It accepts
  `{"leagueId":"weekend","name":"Weekend league","displayName":"Alex"}`.
  The server derives `ownerId` from the token and atomically creates the league
  and organizer membership. Success returns 201 with `{"leagueId":"weekend"}`;
  a conditional conflict returns 409. Client-supplied `ownerId` is rejected.
- `GET /leagues/{leagueId}` requires access-token identity and membership; it
  returns 401 or 403 otherwise. Members receive league details, or 404 if absent.
- Protected responses have `Cache-Control: no-store`; service failures return
  generic 500 messages. Missing membership configuration also fails closed.

POST accepts one UTF-8 JSON object, at most 16 KiB, rejecting unknown fields and
trailing values. The league ID must be 1–128 ASCII letters, digits, `_`, or `-`;
name is trimmed and limited to 200 Unicode code points, and displayName is trimmed
and limited to 80. Invalid input returns 400, or 413 for oversized bodies.
GET retains its existing legacy ID handling. Existing records are not migrated.

No AWS services are added. Once deployed, creation incurs DynamoDB transactional
write usage for two records; membership reads incur strongly consistent read usage.
No membership invitation or listing endpoint exists yet.

## Build and package the Lambda

The Lambda runs on Linux with the `x86_64` architecture. Build a Linux executable named `bootstrap`, then package it at:

```text
infra/artifacts/function.zip
```

### WSL / Linux

Requirements: Go and `zip`.

On Ubuntu, install `zip` if needed:

```bash
sudo apt-get update
sudo apt-get install -y zip
```

From the repository root:

```bash
mkdir -p infra/artifacts

(
  set -e
  cd lambda

  go mod download

  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bootstrap .
  chmod +x bootstrap

  rm -f ../infra/artifacts/function.zip
  zip -j ../infra/artifacts/function.zip bootstrap
)
```

This creates the ZIP directly in the Terraform artifacts folder; no manual copy is needed.

### Windows / PowerShell

Requirements: Go installed on Windows.

Install AWS's Lambda ZIP utility, which packages the executable with the permissions required by Lambda:

```powershell
go install github.com/aws/aws-lambda-go/cmd/build-lambda-zip@v1.55.1
```

From the repository root:

```powershell
New-Item -ItemType Directory -Force -Path ./infra/artifacts | Out-Null

Push-Location ./lambda

try {
    go mod download
    if ($LASTEXITCODE -ne 0) {
        throw "Dependency download failed."
    }

    $env:CGO_ENABLED = "0"
    $env:GOOS = "linux"
    $env:GOARCH = "amd64"

    go build -o bootstrap .
    if ($LASTEXITCODE -ne 0) {
        throw "Lambda build failed."
    }

    # Run the packaging utility on Windows.
    Remove-Item Env:GOOS -ErrorAction SilentlyContinue
    Remove-Item Env:GOARCH -ErrorAction SilentlyContinue

    $goBin = go env GOBIN
    if ([string]::IsNullOrWhiteSpace($goBin)) {
        $goBin = Join-Path (go env GOPATH) "bin"
    }

    $zipTool = Join-Path $goBin "build-lambda-zip.exe"

    Remove-Item ../infra/artifacts/function.zip -ErrorAction SilentlyContinue

    & $zipTool -o ../infra/artifacts/function.zip bootstrap
    if ($LASTEXITCODE -ne 0) {
        throw "Lambda packaging failed."
    }
}
finally {
    Remove-Item Env:CGO_ENABLED -ErrorAction SilentlyContinue
    Remove-Item Env:GOOS -ErrorAction SilentlyContinue
    Remove-Item Env:GOARCH -ErrorAction SilentlyContinue

    Pop-Location
}
```

Use the Lambda ZIP utility instead of `Compress-Archive` to preserve the executable permissions needed on Linux.

### Terraform state setup

Both Terraform roots now use the S3 backend. Follow the
[S3 state runbook](docs/s3-state-migration.md) for initialization and recovery.
The backend bucket is managed separately in `infra/bootstrap/`; the existing
learning bucket and application resource addresses are preserved.

### Application Terraform configuration

Copy `infra/terraform.tfvars.example` to `infra/terraform.tfvars` and set
`account_id` to your intended 12-digit account ID. The provider's account guard
is independent of the backend guard; configure both for the same account.
Neither file contains credentials. Select credentials through the AWS credential
chain, for example `export AWS_PROFILE=terraform-dev` for your entire shell
session, so initialization and subsequent commands use the same profile.
CI should use role-based credentials instead of a local profile.

| Setting | Default | Purpose |
| --- | --- | --- |
| `account_id` | Required | Reject provider access to other accounts |
| `region` | `us-east-2` | Existing deployment region |
| `environment` | `dev` | Only supported environment in this root |
| `owner` | `jhonatan.oliveira` | Common owner tag |
| `log_retention_days` | `7` | Finite Lambda log retention |
| `lambda_memory_mb` | `128` | Lambda memory |
| `lambda_timeout_seconds` | `10` | Lambda timeout |
| `api_throttling_burst_limit` | `5` | API burst target |
| `api_throttling_rate_limit` | `2` | API steady-state requests/second target |

Common tags are `Project`, `Environment`, `Owner`, and `ManagedBy`. Existing
resource-specific tags remain; the learning bucket's environment tag changes
from `Development` to `dev` for consistency. Default names and Terraform resource
addresses are preserved. Changing region can create replacements and requires a
separate migration review. Another environment needs separate state and names;
this configuration intentionally rejects environment values other than `dev`.

No services are added. Defaults retain current capacity and retention; increasing
memory, timeout, retention, or permitted traffic can increase usage costs.
Both on-demand DynamoDB tables and their keys remain unchanged. PITR and deletion
protection await a retention/recovery decision. The inspector still trusts only
`AccountFullAccessRole` through its existing condition and retains scoped project
read access; deciding whether that role is still needed requires a separate review.

Validate locally with `terraform -chdir=infra fmt -check` and
`terraform -chdir=infra validate`. Validation needs the initialized provider but
does not query live AWS. A separately authorized plan must confirm intended tag
updates and no unexplained creates, replacements, or deletions before apply.

### Deploy with Terraform

After packaging, run from the repository root:
Create `infra/dev.s3.tfbackend` from its `.example` file with the actual bucket
and account values if this is a fresh checkout.

```bash
export AWS_PROFILE=terraform-dev
terraform -chdir=infra init -backend-config=dev.s3.tfbackend
terraform -chdir=infra validate
terraform -chdir=infra plan
terraform -chdir=infra apply
```

Run these commands in an environment with valid AWS credentials.
Terraform plans require explicit task authorization. Apply requires separate
authorization after reviewing the plan; the commands above are not approval.

Terraform detects changes to the ZIP through `source_code_hash` and updates the Lambda.

### Notes

- Build with `go build ... .`, rather than `go build main.go`, to include all Go files in the package.
- The ZIP must contain `bootstrap` at its root.
- Keep `bootstrap` and `infra/artifacts/` excluded from Git.
- Windows and WSL have separate Go installations and AWS configurations.
- Run Terraform consistently in one environment. If switching between Windows and WSL, remove only `infra/.terraform/` and run `terraform init` again to install the correct providers. Preserve the Terraform state and lock file.
