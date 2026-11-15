# Podium Picks (mini-leagues)

A learning project using Go Lambda, API Gateway HTTP API, DynamoDB, and Terraform.
The API creates and retrieves leagues; both routes require AWS IAM authorization.

## Podium Picks milestone and sign-in

The [milestone contract](docs/podium-picks-milestone.md) defines target endpoints,
permissions, DynamoDB access patterns, deadline/privacy rules, and acceptance
checks. It distinguishes implemented behavior from future work.

[Cognito setup](docs/cognito-setup.md) describes the new authentication foundation:
email sign-in, classic hosted UI with code + PKCE, a public client, application
scope, and an unattached JWT authorizer. Existing API routes still use AWS_IAM.
Go league reads now enforce JWT identity and membership. Browser login and
organizer creation are separate follow-ups;
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
The initial page is a scaffold, with no login or API requests. Existing AWS_IAM
routes cannot yet be used by friends; Cognito/JWT integration is planned.

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

`LEAGUES_TABLE_NAME` is required at startup. `MEMBERS_TABLE_NAME` configures
membership reads; a missing value fails protected reads closed with 500. AWS region
and credentials follow the SDK's default configuration chain.

Deployment staging: the GET handler now requires verified JWT access-token context
and a matching membership row. Terraform still configures AWS_IAM routes and does
not yet supply MEMBERS_TABLE_NAME or membership read permissions. Do not deploy
this intermediate artifact independently: finish organizer membership creation and
the reviewed JWT/configuration/IAM transition first. IAM-only GET requests receive
401 with this version, even when signed correctly. The curl GET example below is
historical for the earlier IAM handler and will need Bearer-token replacement at
that transition.

- `POST /leagues` accepts `leagueId`, `name`, and `ownerId` strings. Values are
  trimmed and must be nonblank. Success is 201 with `{"leagueId":"weekend"}`;
  duplicate IDs return 409. Invalid JSON/base64 or missing fields return 400.
- `GET /leagues/{leagueId}` requires JWT access-token identity (401 otherwise)
  and membership (403 otherwise). Members receive 200 with `leagueId`, `name`,
  `ownerId`, and UTC RFC3339 `createdAt`; missing leagues return 404 after the
  membership check. Blank/overlong IDs return 400 after authentication.
- Membership storage or decoding failures return generic 500 responses without
  reading league data. All JSON responses include `Cache-Control: no-store`.
- Service or item-decoding failures return generic 500 messages.

Request validation (Phase 4 compatibility changes):

- POST accepts exactly one UTF-8 JSON object. Unknown fields, trailing JSON values,
  arrays, malformed JSON, and a top-level `null` return 400 (`Invalid JSON`).
  Missing, null, or blank required fields return the existing required-fields error.
  Field matching follows Go's JSON decoder (case-insensitive matches are accepted;
  duplicate fields use the last value). Use the documented camelCase field names.
- The decoded POST body may contain at most 16 KiB (16,384 bytes), including JSON
  syntax and whitespace. Larger bodies return 413 (`Request body is too large`).
  For base64 requests, the encoded string is also bounded to 21,848 bytes before
  decoding; send standard base64 without inserted line breaks. Invalid base64
  within this bound returns the existing 400 (`Invalid request body`).
- After trimming surrounding whitespace, `leagueId` and `ownerId` are limited to
  128 UTF-8 bytes each. The same league ID limit applies to GET. POST returns 400
  (`leagueId and ownerId must be at most 128 bytes`); GET returns 400
  (`leagueId must be at most 128 bytes`). No new identifier character set is imposed.
- Trimmed names may contain at most 200 Unicode code points, rather than bytes;
  longer names return 400 (`name must be at most 200 characters`). A visual symbol
  containing combining marks can count as multiple code points.

Previously accepted unknown fields and overlong inputs are now rejected. Existing
stored leagues are not rewritten; GET requests for pre-existing IDs longer than
128 bytes are also rejected. Review any existing callers/data before deployment.
Successful response shapes, conditional-create conflicts, and consistent reads
are unchanged.

League reads use Cognito sub as the membership memberId and require a consistent
read of the existing `(leagueId, memberId)` key. There is no ownerId or email bypass.
League creation still trusts client-supplied ownerId; organizer-only creation and
atomic organizer membership are the next task. No membership write/list endpoints
exist yet. The app is not ready for multiuser exposure.

No AWS services are added by membership lookup. Once deployed, an authorized league
read adds one strongly consistent DynamoDB membership read, with corresponding
on-demand read usage charges.

For signed requests, set `API_URL` to the deployed stage URL, `AWS_REGION` to its
region, and supply temporary credentials through your normal credential process.
The caller needs `execute-api:Invoke` on the intended routes. For example, with
curl's SigV4 support (these requests access live AWS; run only when authorized):

```bash
curl --aws-sigv4 "aws:amz:${AWS_REGION}:execute-api" \
  --user "${AWS_ACCESS_KEY_ID}:${AWS_SECRET_ACCESS_KEY}" \
  -H "x-amz-security-token: ${AWS_SESSION_TOKEN}" \
  -H 'Content-Type: application/json' \
  --data '{"leagueId":"weekend","name":"Weekend league","ownerId":"developer"}' \
  "${API_URL}/leagues"

curl --aws-sigv4 "aws:amz:${AWS_REGION}:execute-api" \
  --user "${AWS_ACCESS_KEY_ID}:${AWS_SECRET_ACCESS_KEY}" \
  -H "x-amz-security-token: ${AWS_SESSION_TOKEN}" \
  "${API_URL}/leagues/weekend"
```

Avoid shell tracing and sharing credential-bearing command output. The session
header is required for temporary credentials; omit it for credentials without a
session token. AWS profiles are not automatically read by curl.

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
