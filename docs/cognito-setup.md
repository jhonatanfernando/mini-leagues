# Cognito foundation for Podium Picks

The Cognito foundation is deployed with refresh-token rotation disabled for Lite.
The JWT route, membership configuration, and permission transition was applied;
a follow-up plan found no changes. Browser PKCE sign-in is implemented in source;
Localhost CORS is deployed; live authorization verification remains outstanding. See the
[milestone contract](podium-picks-milestone.md) for the remaining authorization work.

## What will be created

`infra/auth.tf` adds five resources:

1. `aws_cognito_user_pool.podium_picks`: email sign-in, email verification,
   case-insensitive usernames, 12-character minimum passwords with upper/lowercase,
   number and symbol requirements, verified-email recovery, deletion protection.
2. `aws_cognito_resource_server.podium_picks`: application scope
   `podium-picks/play`. This scope allows API access, not league membership.
3. `aws_cognito_user_pool_client.web`: public/no secret; code grant only;
   Cognito users only; access/ID tokens last 15 minutes, refresh tokens one day;
   refresh-token rotation disabled and token revocation enabled.
4. `aws_cognito_user_pool_domain.podium_picks`: account/region/dev-qualified prefix
   with classic hosted UI (version 1).
5. `aws_apigatewayv2_authorizer.podium_picks`: JWT issuer and client audience,
   reading the Authorization header; attached to the league routes.

The pool explicitly selects Lite rather than the default Essentials feature plan.
Lite includes the classic hosted UI; it does not include the newer managed-login
branding experience. No branding resource is necessary for version 1. Without rotation, a refresh
token remains reusable until expiration or revocation; keep it in memory and
retain the one-day lifetime.
See [AWS feature plans](https://docs.aws.amazon.com/cognito/latest/developerguide/cognito-sign-in-feature-plans.html).

Self-service account registration is enabled, but invitations will control league
access in the application. MFA is OFF for this development milestone; mandatory
TOTP and enrollment should be reviewed before wider use. No SMS, identity pool,
custom domain/certificate, SES configuration, paid threat protection, or new
Lambda IAM permissions are added. The built-in email sender is suitable only for
small development use; review its delivery quotas before inviting the group.

The browser must generate a fresh S256 PKCE challenge/verifier and OAuth state for
each login. `allowed_oauth_flows = ["code"]` alone does not enforce client PKCE;
that browser implementation is a required later task. Do not ship a flow without
PKCE/state validation. Use an established OIDC library rather than custom token
verification. Keep tokens in memory, never in localStorage; short-lived verifier/
state may use sessionStorage and must be removed after callback validation. Avoid
persisting refresh tokens for this milestone. Reload can sign in again through
the Cognito browser session. Logout must clear app memory and visit the hosted
logout endpoint; already issued access tokens can remain valid until expiry.

## Local preparation and checks

Use the current application root at `infra/`, region `us-east-2`, default workspace,
and the reviewed account ID/backend settings. Do not recreate or migrate state.

Optional values in ignored `infra/terraform.tfvars`:

```hcl
auth_callback_urls = ["http://localhost:5173/auth/callback"]
auth_logout_urls   = ["http://localhost:5173/"]
```

URLs match exactly, including the trailing slash. The defaults use localhost;
opening the app at 127.0.0.1 is not the same OAuth origin. Register exact HTTPS
URLs once hosting exists; do not use wildcards. The variables reject non-HTTPS
web destinations except localhost HTTP, fragments, and query strings. These
The browser callback is implemented and localhost CORS has been applied.
Live testing requires explicit authorization.

On an already initialized checkout, offline checks are:

```bash
terraform -chdir=infra fmt -check
terraform -chdir=infra validate
```

Validation checks installed provider schemas, not live domain availability,
account quotas, delivery, or login behavior. On a fresh checkout use the existing
README/runbook initialization procedure; do not change backend authority to make
validation pass.

## Plan and deployment review (separate authorization required)

Package the current Lambda artifact before planning. Use your intended AWS profile
through the credential chain. With explicit task authorization, review an app-root
Terraform plan. Review the route and Lambda changes alongside the public `web_auth` output.
There may also be previously unapplied changes; review every difference. Stop on
unexplained replacements/deletions. Existing table keys, Lambda names, backend, and inspector policy should not
change. The route authorizers, Lambda environment, and execution permissions
change as part of the prepared transition.

Obtain separate authorization after plan review before applying. Keep the plan
and state private. Deletion protection blocks routine user-pool teardown; any
future removal needs an explicit reviewed lifecycle decision.

After an approved deployment, `web_auth` outputs the pool ID, client ID, issuer,
domain, scopes, and registered URLs. These are public app configuration, not AWS
credentials. The output is not a functioning login button or permission to access
league data. Never copy state or tokens into frontend configuration or chat.

## Required follow-up before friends use it

- Identity extraction is implemented/tested in `lambda/auth.go`: it requires
  access-token context and a valid nonblank sub and returns errUnauthenticated
  on failure. Both league handlers map this to 401; GET requires membership and
  POST requires the configured initial organizer.
  The helper relies on gateway validation; it does not verify raw JWTs itself.
- Membership checks and atomic organizer creation are implemented. Legacy client
  ownerId cannot automatically become trusted ownership. The table and permission
  wiring has been applied. After the organizer registers and verifies email,
  configure their exact Cognito sub; empty keeps creation disabled.
- Verify the deployed JWT routes and required `podium-picks/play` scope with
  live requests only after obtaining explicit live-test authorization.
- Review and deploy the prepared browser login/callback/logout flow, PKCE and
  state checks, and public configuration.
- Test login and API calls against the deployed localhost CORS configuration only
  with explicit live-test permission; hosted origins require a separate review.

Live acceptance, when separately authorized: email sign-up requires verification,
code+PKCE exchange issues access tokens, ID tokens are not accepted by scoped API
routes, expired/wrong-client tokens fail, missing scope fails, and valid signed-in
nonmembers still receive 403 for league data. The deployed routes now require JWT access tokens and the play scope.

## Cost

Cognito charges depend on monthly active users, feature plan, and enabled features.
Lite is selected explicitly to keep this small password-based application simple.
Free-tier eligibility is not guaranteed. Built-in email delivery has limits;
separate SES or SMS delivery would introduce their own configuration and possible
charges and is not included. Consult [current Cognito pricing](https://aws.amazon.com/cognito/pricing/)
before approving deployment. This change adds no always-on compute or networking.
