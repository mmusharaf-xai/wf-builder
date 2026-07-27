/**
 * Client for the Go workflow-builder backend.
 * Browser and server both talk to NEXT_PUBLIC_API_URL / API_URL (default :8080).
 */

const DEFAULT_API_URL = "http://localhost:8080";

export function getApiBaseUrl(): string {
  const fromEnv =
    process.env.NEXT_PUBLIC_API_URL ||
    process.env.API_URL ||
    DEFAULT_API_URL;
  return fromEnv.replace(/\/$/, "");
}

/** Absolute URL for a backend path, e.g. apiUrl("/api/workflows") */
export function apiUrl(path: string): string {
  const base = getApiBaseUrl();
  const normalized = path.startsWith("/") ? path : `/${path}`;
  return `${base}${normalized}`;
}

export type ApiFetchOptions = RequestInit & {
  /** Skip JSON Content-Type (e.g. for FormData) */
  skipJsonContentType?: boolean;
};

/**
 * fetch() against the Go API with JSON content-type by default when a body is present.
 */
export async function apiFetch(
  path: string,
  init: ApiFetchOptions = {}
): Promise<Response> {
  const { skipJsonContentType, headers: initHeaders, ...rest } = init;
  const headers = new Headers(initHeaders);

  if (rest.body != null && !skipJsonContentType && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  return fetch(apiUrl(path), {
    ...rest,
    headers,
  });
}
