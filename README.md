# Mini-leagues

A learning project using Go Lambda, API Gateway HTTP API, DynamoDB, and Terraform.
The API creates and retrieves leagues; both routes require AWS IAM authorization.

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

`LEAGUES_TABLE_NAME` is required at startup. Terraform supplies the table name;
AWS region and credentials follow the SDK's default configuration chain.

- `POST /leagues` accepts `leagueId`, `name`, and `ownerId` strings. Values are
  trimmed and must be nonblank. Success is 201 with `{"leagueId":"weekend"}`;
  duplicate IDs return 409. Invalid JSON/base64 or missing fields return 400.
- `GET /leagues/{leagueId}` returns 200 with `leagueId`, `name`, `ownerId`, and
  a UTC RFC3339 `createdAt`; absent IDs return 404 and blank IDs return 400.
- Service or item-decoding failures return generic 500 messages.

Current compatibility behavior accepts unknown JSON fields and imposes no
application length limits. IAM authenticates the caller, but `ownerId` is supplied
by the client and reads do not enforce ownership. Keep this for developer use;
multiuser access needs an explicit ownership policy first.

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
