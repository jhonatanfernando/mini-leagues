# S3 state setup and migration

The bootstrap plan was applied on November 8, 2026: six S3 resources created,
none changed or destroyed. Bucket `mini-leagues-tfstate-298032444404-us-east-2`
has verified versioning, SSE-S3 encryption, public-access blocks, disabled ACLs,
and a TLS-only policy. The user migrated both roots using `init -migrate-state`.
Both active backends were verified to use the expected bucket, separate keys,
encryption, native locking, account guard, and default workspace. Remote states
contain 16 application and 6 bootstrap managed resource instances; all resource
records and outputs match the protected pre-migration backups. Formatting and
validation passed for both roots. No post-migration plan or lock-contention test
has been run.

Remote lineage and serial metadata differ from the original local snapshots.
This is consistent with Terraform 1.17.0's first-write behavior for an empty S3
key: remote `PersistState` refreshes the missing snapshot before generating
initial metadata, even after migration has supplied metadata. Resource data was
verified separately and is unchanged. See the version-specific
[remote persistence implementation](https://github.com/hashicorp/terraform/blob/v1.17.0/internal/states/remote/state.go#L158)
and [migration implementation](https://github.com/hashicorp/terraform/blob/v1.17.0/internal/command/meta_backend_migrate.go#L427).
Use the remote state as authoritative; retain old backups for recovery rather
than pushing them back to restore old metadata.

The numbered setup steps below are retained as a historical/recovery runbook;
do not repeat the completed creation or migration steps. Commands are instructions,
not authorization.
Run from the repository root in the environment holding the authoritative state.

## Design and cost

`infra/bootstrap/` manages a dedicated bucket independently of the app root.
Application resources, including `aws_s3_bucket.learning`, keep their existing
addresses. The bucket uses versioning, SSE-S3 encryption, disabled ACLs, public
access blocks, TLS enforcement, `force_destroy = false`, and `prevent_destroy`.
These protections do not prevent a sufficiently privileged operator from deleting
it outside Terraform or changing the configuration.

The two roots use separate object keys and S3-native locking. No DynamoDB lock
table or customer-managed KMS key is needed. Expect small S3 storage and request
charges, including versions and lock operations; old versions are retained for
recovery and accumulate over time. There is no automatic version expiry.
See HashiCorp's [S3 backend documentation](https://developer.hashicorp.com/terraform/language/backend/s3)
for locking, credentials, and required permissions.

## 1. Prepare locally

Confirm which machine/directory has the authoritative app state; do not assume
this checkout has it. Confirm the expected AWS account and region privately.
This procedure assumes the existing app uses local state and the `default`
workspace. Stop if either assumption is false and adapt the migration before
proceeding. Stop concurrent Terraform operations throughout migration.

Copy `infra/bootstrap/terraform.tfvars.example` to
`infra/bootstrap/terraform.tfvars` and replace the account ID. The provider rejects
other accounts. Select credentials through the AWS credential chain:

```bash
export AWS_PROFILE=terraform-dev
terraform -chdir=infra/bootstrap init -backend=false -lockfile=readonly
terraform -chdir=infra/bootstrap fmt -check
terraform -chdir=infra/bootstrap validate
```

The committed bootstrap provider lock starts from the application's existing
lock, avoiding a provider upgrade. Never put credentials in Terraform files.
Keep bootstrap's temporary local state protected and outside Git.

## 2. Review the bucket plan, then create it

After explicit authorization to plan:

```bash
terraform -chdir=infra/bootstrap plan -out=bootstrap.tfplan
```

Expect six new S3 resources: bucket, public access block, ownership controls,
versioning, encryption configuration, and TLS-only bucket policy. Expect no
application changes or IAM identity-policy changes. The bucket policy is an
access-control change and must be included in the apply review.

Only after separate apply authorization for this reviewed plan:

```bash
terraform -chdir=infra/bootstrap apply bootstrap.tfplan
```

Record the `state_bucket_name` output. If the name already exists, stop and
investigate ownership; do not import or repurpose it without a separate review.
If creation partially fails, preserve bootstrap state and reconcile it before
retrying. Allow newly enabled versioning to propagate before writing state;
AWS recommends waiting 15 minutes after first enabling it.
See [S3 versioning guidance](https://docs.aws.amazon.com/AmazonS3/latest/userguide/manage-versioning-examples.html).

## 3. Review backend access

The identity performing migration and future Terraform operations needs backend
access independently of provider credentials. For each permitted key it needs:

| Scope | Actions |
| --- | --- |
| State bucket, limited to the relevant key prefixes | `s3:ListBucket` |
| Exact state object | `s3:GetObject`, `s3:PutObject` |
| Exact state object plus `.tflock` suffix | `s3:GetObject`, `s3:PutObject`, `s3:DeleteObject` |

Keys are `mini-leagues/dev/terraform.tfstate` and
`mini-leagues/bootstrap/terraform.tfstate`. Give app deployment identities only
the app key; bootstrap operators may need both. State deletion and version
recovery permissions are not needed for normal operations. Bucket management
requires additional S3 control-plane permissions for bootstrap operators.
No permissions are added to the Lambda role. If identity policies need changes,
prepare them for separate review/authorization before making IAM mutations.

## 4. Back up and migrate the application

Before activation, back up the authoritative app state, bootstrap local state,
and original backend configuration in a restricted directory outside Git. For
local state, copy the files without printing them; protect backups with owner-only
permissions. Do not paste state into chat or logs. Preserve backups until remote
persistence and recovery are verified.

Copy `infra/dev.s3.tfbackend.example` to `infra/dev.s3.tfbackend`, replacing the
bucket and account placeholders with the reviewed values. These local files are
ignored. Backend authentication and provider authentication independently use the AWS
credential chain. Export `AWS_PROFILE` for the session when using a local profile;
set the application's `account_id` variable and backend account guard to the same
intended account. The provider no longer embeds a profile name.

After explicit state-migration authorization, activate and migrate:

```bash
cp infra/backend.tf.example infra/backend.tf
terraform -chdir=infra init -migrate-state -backend-config=dev.s3.tfbackend
```

Review the interactive migration prompt. Do not use `-force-copy`,
`-reconfigure`, manual S3 uploads, or a new empty local state as substitutes.
Keep the activated `backend.tf` as source for eventual commit; keep credentials
and local backend parameters outside Git.

## 5. Migrate bootstrap state

Copy `infra/bootstrap/bootstrap.s3.tfbackend.example` to
`infra/bootstrap/bootstrap.s3.tfbackend` and replace the same placeholders. Back
up bootstrap's current local state again if it changed after the first backup.
With explicit authorization for this second state migration:

```bash
cp infra/bootstrap/backend.tf.example infra/bootstrap/backend.tf
terraform -chdir=infra/bootstrap init -migrate-state -backend-config=bootstrap.s3.tfbackend
```

The bucket must remain available to read its own management state. An app destroy
does not manage this bucket. Do not destroy the bootstrap root or delete its
configuration as routine application cleanup.

## 6. Verify before resuming work

With authorized read access, verify both objects and their versions exist in the
expected bucket, encryption/versioning/public access settings are correct, and
both initialized backend configurations have `use_lockfile = true`. Compare
resource addresses and state lineage/serial privately with the backups; never
print state contents. If testing lock contention, obtain explicit live-test
authorization and use a controlled procedure; configuration validation alone
does not prove locking works in AWS.

Build/package the Lambda using README before an app plan. With explicit plan
authorization for each root, review:

```bash
terraform -chdir=infra plan
terraform -chdir=infra/bootstrap plan
```

Migration itself should not create, replace or destroy managed resources. Stop
on unexpected changes and investigate drift or state selection. Do not apply to
make an unexplained plan disappear. Remove obsolete saved plans after review;
they can contain sensitive values.

For future checkouts, initialize each activated root with its local backend
parameter file using `init -backend-config=...`; do not use `-migrate-state` for
routine initialization. Continue using the default workspace with these keys.

## Recovery

If migration is interrupted, stop all writers and determine which backend has
the authoritative snapshot by privately comparing resource inventory,
lineage/serial and S3 versions. Preserve both copies. Do not rerun migration or
apply against an empty state until authority is established.

For state corruption, stop writers and select a known-good S3 object version.
Preserve the current version and local backups, review the resources changed
since the selected version, then prepare a separately authorized recovery/state
operation. Restoring an old state does not restore AWS resources or DynamoDB
data. Verify the restored state with an authorized plan before resuming writes.
If the entire bucket is lost, recover the bucket/access and protected state
backups under a reviewed disaster-recovery procedure; bucket versioning alone
does not protect against deletion of the bucket and all its versions.

Never force-unlock without confirming the owning operation has ended and obtaining
authorization for the state operation. `prevent_destroy` and state backups must
remain in place throughout recovery.
