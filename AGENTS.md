# Project context

Mini-leagues is a personal learning project for Go, AWS, Terraform,
and eventually a Go MCP server.

Explain meaningful design decisions briefly so I can learn.
Prefer small, reviewable changes.

# Repository

- lambda/: Go Lambda application.
- infra/: Terraform infrastructure.
- README.md: setup, build, packaging, and deployment instructions.
- Read the current code before assuming implementation details.

# Go conventions

- Group related operations in domain files.
- Keep league models and handlers together in league.go.
- Keep routing, startup, and JSON response helpers separate.
- Do not create a file for every endpoint.
- Pass request context to AWS SDK calls.
- Return useful HTTP statuses without exposing internal errors.
- Avoid unnecessary dependencies and abstractions.

# Testing and verification

- For behavior changes, write a focused failing test first when practical.
- Test validation, successful behavior, conflicts, and service failures.
- Use mocks or fakes for unit tests; do not call live AWS.
- Prefer small interfaces for the AWS operations needed by handlers.
- From lambda/, run:
  - gofmt on changed Go files
  - go test ./...
  - go vet ./...
  - CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bootstrap .
- For Terraform changes, run:
  - terraform -chdir=infra fmt -check
  - terraform -chdir=infra validate
- Report checks that failed or could not run.

# AWS and Terraform

- This uses my personal AWS account.
- Preserve existing Terraform state and resource addresses.
- Never print credentials, tokens, or Terraform state contents.
- Never commit state, credentials, bootstrap, or generated ZIP files.
- Keep .terraform.lock.hcl, go.mod, and go.sum committed.
- Scope IAM permissions to required actions and resources.
- Explain expected cost implications when adding AWS services.
- Do not run live AWS tests without explicit authorization.
- Terraform plan requires explicit authorization for the task.
- Terraform apply, destroy, state changes, account creation, and IAM
  mutations require separate explicit authorization after review.
- Do not use -auto-approve.

# Workflow

- Inspect the repository and propose a brief plan for substantial changes.
- Implement only the requested scope.
- Update documentation when commands or API behavior change.
- Before finishing, summarize behavior, checks, and remaining limitations.
- Do not commit or push unless requested.
- Use one agent by default.
- Use parallel agents only when requested, with separate ownership.