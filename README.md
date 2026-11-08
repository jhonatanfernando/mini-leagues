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
go install github.com/aws/aws-lambda-go/cmd/build-lambda-zip@latest
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

### Deploy with Terraform

After packaging, run from the repository root:
Create `infra/dev.s3.tfbackend` from its `.example` file with the actual bucket
and account values if this is a fresh checkout.

```bash
AWS_PROFILE=terraform-dev terraform -chdir=infra init -backend-config=dev.s3.tfbackend
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
