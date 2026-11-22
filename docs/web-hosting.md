# Web hosting review

Terraform manages a deployed public CloudFront HTTPS distribution backed by a private S3
bucket. CloudFront origin access control signs requests, and the bucket policy
allows reads only from this distribution. API responses still go directly to API
Gateway and are not cached by the website distribution. The app shell and assets
use CloudFront's disabled-cache policy and S3 `no-store`; this favors safe, fresh
updates over minimizing requests. A CloudFront Function rewrites only
`/auth/callback` to the app shell, leaving the browser URL and OAuth query intact.

The distribution's generated HTTPS origin is added to the Cognito client's
callback/logout URLs and API CORS origins. The localhost entries remain for local
development. The app is publicly readable; Cognito and Go enforce API access.
This does not add invitations, game features, or login for nonmembers.

## Build and review

The build uses public settings from the ignored `web/.env.local`; see
[web setup](../web/README.md). These values are included in the JavaScript and
must never contain credentials, passwords, tokens, or private invitation links.
Build before validation/plan and do not rebuild or change `web/dist` between
reviewing a saved plan and applying it. Terraform manages each built object;
a missing build is rejected during planning to avoid an accidental empty site.

```bash
cd web
npm ci
npm test
npm run build
cd ..
terraform -chdir=infra fmt -check
terraform -chdir=infra validate
```

A plan must be explicitly authorized for the task. Review the S3 bucket and
policy, CloudFront distribution, function and response headers, site objects,
API CORS change, and Cognito app-client callback/logout updates. Existing
resource addresses and table data must not be replaced or destroyed. Apply and
bucket-policy changes require separate explicit authorization after review.
Never use `-auto-approve`. Treat saved plans as private and do not commit them.

Once applied, `terraform -chdir=infra output -raw web_url` returns the public
website address. Confirming sign-in and league operations against AWS requires
explicit live-test authorization. Rollback requires building a known previous
revision and reviewing a new plan; do not delete the bucket to roll back a release.
Changing the generated CloudFront domain changes the registered origin and URLs.

## Cost

S3 storage and requests, CloudFront requests and data transfer, and CloudFront
Function invocations can incur usage charges. Disabling cache increases S3 and
CloudFront origin requests. Actual costs depend on traffic, region and account
eligibility; no free-tier amount or monthly total is assumed. Check current
[AWS S3 pricing](https://aws.amazon.com/s3/pricing/) and
[CloudFront pricing](https://aws.amazon.com/cloudfront/pricing/) before approval.
