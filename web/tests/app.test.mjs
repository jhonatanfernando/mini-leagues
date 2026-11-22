import test from "node:test";
import assert from "node:assert/strict";
import { loadConfig, completeCallback } from "../src/auth.ts";
import { leagueRequest } from "../src/api.ts";

const env = {
  VITE_COGNITO_ISSUER: "https://issuer.example/pool/",
  VITE_COGNITO_CLIENT_ID: "client",
  VITE_COGNITO_DOMAIN: "https://login.example/",
  VITE_API_URL: "https://api.example/dev/",
};
const config = loadConfig(env);

test("configuration requires HTTPS endpoints and normalizes trailing slashes", () => {
  assert.equal(config.apiUrl, "https://api.example/dev");
  assert.equal(
    loadConfig({ ...env, VITE_API_URL: "http://api.example" }),
    null,
  );
  assert.equal(loadConfig({ ...env, VITE_COGNITO_CLIENT_ID: "" }), null);
  assert.equal(loadConfig({ ...env, VITE_COGNITO_ISSUER: "not a url" }), null);
});

test("callback consumes the full response and clears the URL on success and failure", async () => {
  let cleanups = 0;
  globalThis.window = {
    location: {
      href: "http://localhost:5173/auth/callback?code=private&state=state",
    },
    history: {
      replaceState: (_state, _title, url) => {
        assert.equal(url, "/");
        cleanups++;
      },
    },
  };
  globalThis.document = { title: "Podium Picks" };
  const user = { access_token: "token" };
  assert.equal(
    await completeCallback({
      signinRedirectCallback: async (url) => {
        assert.match(url, /code=/);
        return user;
      },
    }),
    user,
  );
  await assert.rejects(
    completeCallback({
      signinRedirectCallback: async () => {
        throw new Error("invalid state");
      },
    }),
  );
  assert.equal(cleanups, 2);
});

test("league API sends the access token and JSON and maps failures without leaking details", async () => {
  const calls = [];
  globalThis.fetch = async (url, options) => {
    calls.push({ url, options });
    return new Response('{"leagueId":"grid"}', { status: 201 });
  };
  assert.deepEqual(
    await leagueRequest(config, "secret", "/leagues", {
      method: "POST",
      body: '{"leagueId":"grid"}',
    }),
    { leagueId: "grid" },
  );
  assert.equal(calls[0].url, "https://api.example/dev/leagues");
  assert.equal(calls[0].options.headers.Authorization, "Bearer secret");
  assert.equal(calls[0].options.headers["Content-Type"], "application/json");
  assert.equal(calls[0].options.cache, "no-store");
  for (const [status, expected] of [
    [401, /sign in again/],
    [403, /do not have access/],
    [409, /already in use/],
    [500, /Something went wrong/],
  ]) {
    globalThis.fetch = async () =>
      new Response("private service detail", { status });
    await assert.rejects(leagueRequest(config, "secret", "/leagues"), expected);
  }
});

test("OIDC client rejects callbacks with unknown or replayed state", async () => {
  const { makeAuth } = await import("../src/auth.ts");
  const storage = new Map();
  globalThis.window = {
    location: { origin: "http://localhost:5173" },
    sessionStorage: {
      get length() {
        return storage.size;
      },
      key: (index) => [...storage.keys()][index] ?? null,
      getItem: (key) => storage.get(key) ?? null,
      setItem: (key, value) => {
        storage.set(key, value);
      },
      removeItem: (key) => {
        storage.delete(key);
      },
      clear: () => storage.clear(),
    },
  };
  const manager = makeAuth(config);
  await assert.rejects(
    manager.signinRedirectCallback(
      "http://localhost:5173/auth/callback?code=abc&state=unknown",
    ),
  );
  await assert.rejects(
    manager.signinRedirectCallback(
      "http://localhost:5173/auth/callback?code=abc&state=unknown",
    ),
  );
});

test("local sign-in uses the registered localhost origin", async () => {
  const { isSupportedLocalOrigin } = await import("../src/auth.ts");
  assert.equal(isSupportedLocalOrigin("http://localhost:5173"), true);
  assert.equal(isSupportedLocalOrigin("http://127.0.0.1:5173"), false);
  assert.equal(isSupportedLocalOrigin("http://localhost:5174"), false);
  assert.equal(isSupportedLocalOrigin("https://app.example"), true);
});

test("CloudFront serves the app shell only for the OAuth callback", async () => {
  const { readFile } = await import("node:fs/promises");
  const { runInNewContext } = await import("node:vm");
  const source = await readFile(new URL("../../infra/web_callback.js", import.meta.url), "utf8");
  const handler = runInNewContext(`${source}\nhandler`);
  const callback = { request: { uri: "/auth/callback", querystring: { code: { value: "example" } } } };
  assert.equal(handler(callback).uri, "/index.html");
  assert.equal(callback.request.querystring.code.value, "example");
  assert.equal(handler({ request: { uri: "/assets/missing.js" } }).uri, "/assets/missing.js");
});
