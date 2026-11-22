import type { AppConfig } from "./auth";

export interface League {
  leagueId: string;
  name: string;
  ownerId: string;
  createdAt: string;
}

export async function leagueRequest<T>(
  config: AppConfig,
  token: string,
  path: string,
  options: RequestInit = {},
): Promise<T> {
  const response = await fetch(`${config.apiUrl}${path}`, {
    ...options,
    headers: {
      Authorization: `Bearer ${token}`,
      ...(options.body ? { "Content-Type": "application/json" } : {}),
    },
    cache: "no-store",
  });
  if (!response.ok) {
    const messages: Record<number, string> = {
      400: "Check the details and try again.",
      401: "Your session has expired. Please sign in again.",
      403: "You do not have access to this league or action.",
      404: "League not found.",
      409: "That league ID is already in use.",
      413: "The request is too large.",
    };
    throw new Error(
      messages[response.status] ?? "Something went wrong. Please try again.",
    );
  }
  return (await response.json()) as T;
}
