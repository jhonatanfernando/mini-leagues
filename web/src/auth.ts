import {
  InMemoryWebStorage,
  UserManager,
  WebStorageStateStore,
  type User,
} from "oidc-client-ts";

export interface AppConfig {
  issuer: string;
  clientId: string;
  domain: string;
  apiUrl: string;
}

export function loadConfig(
  env: Record<string, string | undefined>,
): AppConfig | null {
  const issuer = env.VITE_COGNITO_ISSUER?.replace(/\/$/, "");
  const domain = env.VITE_COGNITO_DOMAIN?.replace(/\/$/, "");
  const apiUrl = env.VITE_API_URL?.replace(/\/$/, "");
  const clientId = env.VITE_COGNITO_CLIENT_ID;
  if (!issuer || !domain || !apiUrl || !clientId) return null;
  try {
    if (
      [issuer, domain, apiUrl].some(
        (value) => new URL(value).protocol !== "https:",
      )
    )
      return null;
  } catch {
    return null;
  }
  return { issuer, domain, apiUrl, clientId };
}

// Cognito's development callback is registered against this exact origin.
export function isSupportedLocalOrigin(origin: string): boolean {
  const { hostname } = new URL(origin);
  return hostname !== "localhost" && hostname !== "127.0.0.1" || origin === "http://localhost:5173";
}

export function makeAuth(config: AppConfig): UserManager {
  return new UserManager({
    authority: config.issuer,
    client_id: config.clientId,
    redirect_uri: `${window.location.origin}/auth/callback`,
    response_type: "code",
    scope: "openid email podium-picks/play",
    // Only the PKCE verifier and OAuth state survive the round trip, briefly.
    stateStore: new WebStorageStateStore({ store: window.sessionStorage }),
    userStore: new WebStorageStateStore({ store: new InMemoryWebStorage() }),
    automaticSilentRenew: false,
    monitorSession: false,
    loadUserInfo: false,
  });
}

export async function completeCallback(manager: UserManager): Promise<User> {
  try {
    // The library validates state, issuer and token response, and consumes state.
    return await manager.signinRedirectCallback(window.location.href);
  } finally {
    window.history.replaceState({}, document.title, "/");
  }
}

export async function signOut(
  manager: UserManager,
  config: AppConfig,
): Promise<void> {
  await manager.removeUser();
  const url = new URL(`${config.domain}/logout`);
  url.searchParams.set("client_id", config.clientId);
  url.searchParams.set("logout_uri", `${window.location.origin}/`);
  window.location.assign(url.toString());
}
