# Podium Picks: first playable milestone

Status: agreed implementation direction, November 15, 2026. Endpoint contracts
below are targets, not a claim that handlers or deployed resources exist.

## Scope and current implementation

One private league, one season, one race, and a full round played by friends.
Retain one Go Lambda behind the existing API Gateway HTTP API and use DynamoDB.
The browser is the TypeScript/Vite app in `web/`; static hosting is a later task.

Implemented in source:

- `lambda/league.go`: authenticated organizer creates a league and membership in
  one conditional transaction; GET uses consistent reads and membership checks.
- `lambda/prediction.go` and `scoring.go`: pure podium validation and non-stacking
  scoring; no prediction HTTP handlers or persistence yet.
- `lambda/membership.go`: membership model and consistent-read access check.
  GET league requires JWT identity and membership.
- `infra/dynamodb.tf`: league and membership tables; no membership write/list handlers.
- `lambda/auth.go`: tested identity extraction from API Gateway JWT context;
  requires an access token and nonblank, unpadded sub. Returns a generic error
  mapped to 401 by both league handlers.
- `infra/auth.tf`: Cognito and a JWT authorizer attached to the league routes.
- `web/`: buildable placeholder; no sign-in, callback, or API integration yet.

Only POST /leagues and GET /leagues/{leagueId} are routed. JWT authorization and
membership permissions have been applied; a follow-up plan found no changes.
Live authorization tests remain outstanding. With an empty initial organizer
sub, creation fails closed until a verified organizer account is configured.

## Identity and access policy

Friends use Cognito email/password sign-up with email verification, then hosted
login via authorization code + S256 PKCE. A public app client has no secret.
Request `openid email podium-picks/play`; send the access token as a Bearer token.
API Gateway will validate issuer, client audience, expiry, and the play scope.
Go must require access-token claims, a nonempty `sub`, and league authorization.
Never accept user identity or organizer role from request JSON or arbitrary headers.

The user pool permits account registration; a Cognito account alone does not
permit joining or reading a league. Membership requires an invitation.

Use Cognito `sub` as the stable application user ID; email is neither a key nor
an authorization rule. One configured organizer sub can create the first league;
thereafter the league's server-derived ownerId authorizes organizer actions.
Creation atomically adds organizer membership. Existing ownerId values are not
proof of authenticated ownership: create a fresh league or separately review an
explicit ownership migration. Do not silently adopt legacy records.

| Caller | Allowed operations |
| --- | --- |
| Anonymous | View web shell and Cognito login only; API returns 401 |
| Signed-in nonmember | Redeem a valid invitation; no league/race/pick/results access |
| Member | Read league, members, races, own picks, published results, standings; save own picks before lock; read others' picks after lock |
| Organizer (also member) | All member actions; create invitations and races; publish results |
| Configured initial organizer | Create the milestone league |

All future API routes require JWT and `podium-picks/play`. Membership and owner
checks remain in Go, not in client UI, CORS, or OAuth scopes. Replace existing
AWS_IAM application routes only after Go authorization tests pass. Retain IAM
for Terraform, Lambda execution, and the separate inspector role; no IAM fallback
route may bypass JWT/membership rules. Friends never receive AWS credentials.

## Contract conventions

JSON uses camelCase; podium wire shape is `{"p1":"VER","p2":"NOR","p3":"LEC"}`.
Introduce explicit request/response types or JSON tags before exposing the existing
Go Podium struct. Exactly three nonblank, distinct eligible IDs are required;
IDs match the race's driver list exactly, without case conversion or trimming.

Use bounded single-object JSON bodies with unknown fields rejected, following
existing handlers' 16 KiB body limit. IDs in URLs must be 1–128 ASCII characters
from letters, digits, `_`, and `-` for the new game entities; use that restriction
when creating the new league too, without rewriting legacy league IDs. Season
is a four-digit year. Clients choose a raceId for idempotent creation; the backend
creates invitation tokens. Display names
are trimmed, nonblank, and at most 80 Unicode code points; emails are never listed.

Common statuses: 400 invalid input, 401 missing/invalid authentication, 403 missing
membership/organizer permission, 404 missing resource after access checks, 409
state conflict, 413 oversized body, 500 generic service failure. Fail closed when
authorization storage fails. Return JSON `{"message":"..."}` errors without SDK
error details. No request bodies, tokens, invitation links, or picks in logs.

Protected responses use `Cache-Control: no-store`; do not cache private API data
in CloudFront, service workers, or browser persistent storage.

## Target endpoints

Paths are relative to the existing API stage URL. `L` means leagueId, `S` season,
and `R` raceId below; actual route parameters use those full names.

| Method/path | Permission | Input and success | Important conflicts |
| --- | --- | --- | --- |
| POST /leagues | Initial organizer | `{leagueId,name,displayName}` → 201 `{leagueId}`; derive ownerId and create membership | Existing ID → 409; body ownerId → 400 |
| GET /leagues/{L} | Member | 200 `{leagueId,name,ownerId,createdAt}` | Nonmember → 403 |
| GET /leagues/{L}/members | Member | 200 `{members:[{memberId,displayName}]}` | Must include all query pages |
| POST /leagues/{L}/invitations | Organizer | `{}` → 201 `{invitationToken,expiresAt}`; return raw token only at creation | Token expiry seven days |
| POST /invitations/redeem | Signed-in user | `{invitationToken,displayName}` → 200 `{leagueId,memberId}` | Invalid/expired/used by another user → 409 |
| POST /leagues/{L}/seasons/{S}/races | Organizer | `{raceId,name,deadlineAt,eligibleDrivers}` → 201 `{raceId}` | Identical retry → 200; changed body for same raceId → 409 |
| GET /leagues/{L}/seasons/{S}/races | Member | 200 `{races:[...]}` with IDs, names, deadlines, driver choices, publication state | No pick values |
| GET /leagues/{L}/seasons/{S}/races/{R} | Member | 200 race metadata and publication state | Unknown race → 404 |
| PUT /leagues/{L}/seasons/{S}/races/{R}/prediction | Member | Podium → 200 `{podium,updatedAt}`; create or replace own row | At/after lock → 409 |
| GET /leagues/{L}/seasons/{S}/races/{R}/prediction | Member | 200 own `{podium,updatedAt}` | No saved prediction → 404 |
| GET /leagues/{L}/seasons/{S}/races/{R}/predictions | Member after lock | 200 `{predictions:[{memberId,podium}]}` | Before lock → 403, with no other picks or submission counts |
| PUT /leagues/{L}/seasons/{S}/races/{R}/result | Organizer after lock | Podium → 200 `{podium,publishedAt,publishedBy}` | Different already-published result → 409 |
| GET /leagues/{L}/seasons/{S}/races/{R}/result | Member | 200 published result | Unpublished → 404 |
| GET /leagues/{L}/seasons/{S}/races/{R}/scores | Member | 200 `{scores:[{memberId,points}]}` | Unpublished → 409 |
| GET /leagues/{L}/seasons/{S}/standings | Member | 200 `{standings:[{memberId,displayName,points,rank}],publishedRounds}` | Before any result: zero points, no hidden picks |

Race creation uses a client-chosen raceId retained across retries. A conditional
write creates the record once; an identical retry returns that record with 200,
and a different request for the same ID returns 409. Thus an ambiguous timeout
cannot create a second race. No public league discovery is needed for this milestone;
invitation redemption supplies leagueId, which the UI can retain as navigation
state without treating it as authorization.

## Race state, privacy, and scoring

- Store deadlines as UTC instants with explicit timezone. Require a future
  deadline and at least three unique eligible drivers when creating the race.
- Freeze deadline and eligible drivers at creation for this milestone.
- An injected backend clock is authoritative: editing is allowed only when
  `now < deadlineAt`. At equality the race is locked. Reevaluate time immediately
  before each write attempt and on each retry; do not use browser timestamps.
- This defines acceptance at the server's write-decision time, not the eventual
  storage commit instant. A write already accepted just before lock may finish
  just afterward. Never delay an accepted request in a queue for later execution.
- A transaction checks current membership and unchanged race metadata together
  with the prediction write. Same-member concurrent edits use last successful
  write wins, with one row per member/race.
- Before lock, only the owning member can read their own prediction through the
  application, including for the organizer. At lock, members can read all picks,
  even if no official result is published yet. AWS administrators can technically
  inspect DynamoDB; this is application privacy, not encryption from the operator.
- Publishing requires membership plus owner authorization and a locked race.
  Validate the official podium against the same frozen eligible driver list.
- First publication conditionally saves one result on race metadata. An identical
  retry returns the original publication record; a different result returns 409.
  On conditional conflict, reread consistently and compare before responding.
- Each prediction position earns 5 for exact position, otherwise 2 if its driver
  is elsewhere on the official podium, otherwise 0. Scores never stack. All exact
  = 15; a cyclic permutation = 6; one exact plus two displaced = 9.
- Compute scores on read from predictions and published results. Sum each race
  once for season totals; never increment a mutable season total on publication.
  Missing predictions earn zero, and unpublished races contribute nothing.
- Equal totals share competition rank (1, 1, 3); use memberId for stable ordering
  within ties. All members appear, including those without submissions.
- No result replacement, race rescheduling, membership removal, or organizer
  transfer UI in this milestone. Revisit these with explicit correction policies.

## Data and access patterns

Preserve existing Terraform addresses, table names, and primary keys:

| Table / record | Key | Attributes / access |
| --- | --- | --- |
| Existing mini_leagues | leagueId | name, ownerId=sub, createdAt; GetItem |
| Existing mini_league_members | leagueId + memberId=sub | displayName, joinedAt; GetItem for access; Query league for members |
| Proposed game table / race | pk=`LEAGUE#L#SEASON#S`, sk=`RACE#R#META` | Frozen deadline and eligible drivers; optional published result + publisher/time |
| Proposed game table / prediction | Same pk, sk=`RACE#R#PICK#sub` | Podium, updatedAt; GetItem for own; Query race prefix after lock |
| Proposed game table / invitation | pk=`INVITE#sha256(token)`, sk=`META` | leagueId, expiresAt, consumedBy; conditional transactional redemption |

The new game table is a future change, not part of auth.tf. Use on-demand billing,
no indexes initially, and consume every Query page. Query the season partition
for race metadata and scoring inputs; never use a table scan. Use consistent
reads for access decisions, saved picks, publication retries, and scoring.

Create league+organizer membership atomically. Redeem invitations with one
transaction that conditionally consumes the invite and creates membership.
Same-user redemption retry is harmless; another user cannot consume it. Invitations
are random high-entropy single-use bearer tokens, shared manually. Store only
hashes and enforce expiration in Go and write conditions; TTL deletion is not an
authorization mechanism. An existing member's redemption must not overwrite their
profile; define the idempotent return in the redemption handler tests.

Do not reinterpret or overwrite legacy membership data without checking its
intended identity scheme. No data migration is performed by this document.

## Ordered remaining work and acceptance gates

1. Pure validation/scoring: implemented; retain tests and Go checks.
2. Contract: this document; update it alongside each implemented endpoint.
3. Cognito foundation: configuration prepared; see [setup](cognito-setup.md).
4. Go identity helper: implemented and tested offline; missing/invalid trusted
   identity returns errUnauthenticated (handlers must map it to 401). Spoofed
   body/header/path/query IDs and IAM identity are ignored. Gateway remains
   responsible for signature, issuer, audience, expiry, and application scope.
5. Membership lookup: implemented/tested; consistent reads use (leagueId, sub),
   nonmembers receive 403, malformed/storage failures return generic 500, and no
   league read occurs before authorization. JSON responses use no-store.
6. Atomic organizer creation: implemented; JWT organizer identity, conditional
   transaction, duplicate/conflict/failure tests. Configuration wiring follows.
7. JWT route transition: applied; mocked configuration tests pass and a follow-up
   plan found no changes. Test tokens and membership only with explicit live-test
   authorization.
8. Game table: additive keys/config/permissions, no replacement of old tables.
9. Invitation creation: organizer only; raw token never stored/logged.
10. Redemption/members: atomic consume, expiry, replay, pagination tests.
11. Race handlers: organizer only, fixed UTC deadline and eligible drivers.
12. Prediction writes: duplicate drivers → 400; edit just before lock succeeds;
    edit exactly at/after lock → 409 with no write; recheck time on retry.
13. Prediction reads: owning member can read before lock; others cannot; organizer
    has no special early-read privilege; all members can read after lock.
14. Result publication: nonorganizer → 403; before lock → 409; identical second
    publication succeeds without changing publishedAt; different result → 409.
15. Scores/standings: manual fixtures verify 5/2/0, shared ranks, zero missing
    picks, multiple rounds, pagination, and identical totals after republishing.
16. Gateway game routes: require JWT plus play scope everywhere, exact permissions.
17. Browser login/join: PKCE and state validation, signup verification, safe token
    handling, invitation preserved across login without logging it.
18. Browser predictions: save/edit, deadline display, stale-form rejection.
19. Organizer/result/standings screens: direct unauthorized calls still fail.
20. Hosting/CORS: exact browser origins and callbacks, private S3/HTTPS CloudFront.
21. Authorized two-browser rehearsal: join → edit → lock/reveal → publish twice →
    verify round points and standings. Record actual results.

For each Go change, start with focused failing tests where practical, use fakes,
and run formatting, tests, vet, and the Linux build via `scripts/check`.
Infrastructure changes require fmt/validate, then explicitly authorized plan
review before deployment. Apply and IAM mutations require separate authorization
after review. Never combine approval of this document with permission to deploy.
Review plans at Cognito creation, JWT/permission transition, game-table creation,
route additions, and website/CORS deployment. Keep artifacts and existing state
addresses intact; never run live tests without explicit authorization.

## Later backlog

- SQS only when background work is justified.
- Automatic schedules/results, race changes, and result corrections.
- MCP after domain functions have a real second entry point.
