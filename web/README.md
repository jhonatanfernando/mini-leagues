# Podium Picks web

A TypeScript/Vite browser app for the existing league API. It signs in through
Cognito's hosted UI with authorization code and PKCE, lets the configured
organizer create a league, and lets a member look up a league by ID. Invitations,
races, predictions, results and standings are not implemented yet.

## Local setup

Use Node.js 24 LTS and its bundled npm. In WSL, use Linux Node/npm. Copy
`.env.example` to `.env.local` and fill in the **public** Cognito issuer, app
client ID, Cognito domain and API stage URL. The API URL ends at `/dev`, not
`/leagues`. Terraform's `web_auth` and `create_league_url` outputs contain these
public values. Never put passwords, AWS credentials, tokens, client secrets or
invitation secrets in any `VITE_` variable: they are embedded in the browser build.

```bash
cd web
cp .env.example .env.local
# Edit .env.local with your public settings.
npm ci
npm test
npm run dev
```

The dev server binds to `localhost:5173` and stops if that port is in use,
instead of choosing another port. Open `http://localhost:5173/` exactly. Cognito currently registers
`http://localhost:5173/auth/callback` and `http://localhost:5173/` for logout;
`127.0.0.1` is a different origin. CORS for localhost has been deployed; live
end-to-end verification remains outstanding.
Creating a league requires the configured organizer's Cognito account. Existing
league reads require an existing membership; no invitation UI exists yet.

The OIDC client generates and checks state and S256 PKCE for each login. The
short-lived redirect state and verifier are in session storage and consumed on
callback. Tokens stay in memory and are discarded on reload; sign in again when
needed. Sign-out clears memory and ends the Cognito browser session. A previously
issued access token may remain usable until expiry.

```bash
npm run check   # strict TypeScript checking
npm test        # offline auth/configuration and API tests
npm run build   # type-check and create dist/
npm run preview # serve the production build locally
```

Private S3 and CloudFront hosting is deployed; live end-to-end tests remain outstanding.
See [the hosting review](../docs/web-hosting.md) for build, review, deployment,
cost, and callback details.
Keep `package-lock.json` committed; `node_modules/` and `dist/` are generated.
