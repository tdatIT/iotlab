/**
 * HTTP client for the iotlab-service API.
 *
 * Every endpoint answers with the shared envelope `{ code, message, data }`;
 * `code === 0` means success. The browser calls the backend directly at
 * VITE_API_BASE_URL (baked in at build time), so the backend must allow this
 * app's origin via CORS.
 */
const API_BASE_URL = (
  import.meta.env.VITE_API_BASE_URL ?? "http://localhost:3000/api/v1"
).replace(/\/$/, "");

/** The service rate-limits at ~10 req/s; retry 429s a few times with backoff. */
const MAX_RETRIES = 3;
const RETRY_BASE_MS = 400;

interface Envelope<T> {
  code: number;
  message: string;
  data: T;
}

export class ApiError extends Error {
  readonly status: number;
  readonly code: number | undefined;

  constructor(message: string, status: number, code?: number) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }
}

function delay(ms: number, signal?: AbortSignal): Promise<void> {
  return new Promise((resolve, reject) => {
    const timer = setTimeout(resolve, ms);
    signal?.addEventListener("abort", () => {
      clearTimeout(timer);
      reject(signal.reason);
    });
  });
}

export async function apiGet<T>(
  path: string,
  { signal }: { signal?: AbortSignal } = {},
): Promise<T> {
  for (let attempt = 0; ; attempt++) {
    const res = await fetch(`${API_BASE_URL}${path}`, {
      headers: { Accept: "application/json" },
      signal,
    });

    if (res.status === 429 && attempt < MAX_RETRIES) {
      await delay(RETRY_BASE_MS * 2 ** attempt, signal);
      continue;
    }

    let body: Envelope<T> | undefined;
    try {
      body = (await res.json()) as Envelope<T>;
    } catch {
      body = undefined;
    }

    if (!res.ok || !body || body.code !== 0) {
      throw new ApiError(
        body?.message || res.statusText || "Request failed",
        res.status,
        body?.code,
      );
    }
    return body.data;
  }
}
