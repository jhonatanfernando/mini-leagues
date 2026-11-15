# Podium Picks web

A static TypeScript application built with Vite. There is no UI framework,
backend server, authentication, or API integration yet. Go remains the authority
for validation, deadlines, visibility, and scoring; the browser will call the API.

Use Node.js 24 LTS and its bundled npm, installed in the same environment as the
checkout. In WSL, use Linux Node/npm rather than the Windows installation.

From `web/`:

```bash
npm ci
npm run dev
```

Open the localhost URL Vite prints. The development server binds to loopback.

```bash
npm run check   # strict TypeScript checking
npm run build   # type-check and create dist/
npm run preview # serve the production build locally
```

Keep `package-lock.json` committed; `node_modules/` and `dist/` are generated.
`dist/` will eventually be uploaded to the static website host. No hosting
resources are provisioned by these commands. Use `index.html` as the entry point
and `src/main.ts` for application startup. Add feature modules under `src/` when
needed rather than creating empty folders now.

Future API URL and Cognito client settings are public browser configuration.
Never put AWS credentials, client secrets, or invitation secrets in build-time
configuration. All `VITE_` environment variables are exposed to the browser.
