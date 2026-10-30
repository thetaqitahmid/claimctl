import { beforeEach, afterEach, describe, expect, it, vi } from "vitest";
import type { BaseQueryApi } from "@reduxjs/toolkit/query";

const api = {
  signal: new AbortController().signal,
  dispatch: vi.fn(),
  getState: () => ({}),
  extra: undefined,
  endpoint: "test",
  type: "mutation",
  abort: vi.fn(),
} as BaseQueryApi;

const jsonResponse = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });

beforeEach(() => {
  vi.resetModules();
  vi.clearAllMocks();
});
afterEach(() => vi.unstubAllGlobals());

describe("baseQuery CSRF integration", () => {
  it("bootstraps a challenge and sends cookie credentials plus header", async () => {
    const requests: Request[] = [];
    vi.stubGlobal("fetch", vi.fn(async (request: Request) => {
      requests.push(request);
      return request.url.endsWith("/auth/csrf")
        ? jsonResponse({ csrfToken: "challenge" })
        : jsonResponse({ ok: true });
    }));
    const { baseQueryWithReauth } = await import("./baseQuery");
    const result = await baseQueryWithReauth(
      { url: "/login", method: "POST", body: { email: "user" } }, api, {}
    );
    expect(result.data).toEqual({ ok: true });
    expect(requests).toHaveLength(2);
    expect(requests[0].method).toBe("GET");
    expect(requests[1].credentials).toBe("include");
    expect(requests[1].headers.get("X-CSRF-Token")).toBe("challenge");
  });

  it("rebootstraps only typed pre-handler CSRF rejection", async () => {
    let challenge = 0;
    let mutation = 0;
    const headers: (string | null)[] = [];
    vi.stubGlobal("fetch", vi.fn(async (request: Request) => {
      if (request.url.endsWith("/auth/csrf")) {
        return jsonResponse({ csrfToken: `token${++challenge}` });
      }
      headers.push(request.headers.get("X-CSRF-Token"));
      return ++mutation === 1
        ? jsonResponse({ code: "CSRF_INVALID" }, 403)
        : jsonResponse({ ok: true });
    }));
    const { baseQueryWithReauth } = await import("./baseQuery");
    await baseQueryWithReauth({ url: "/reservations", method: "POST" }, api, {});
    expect(headers).toEqual(["token1", "token2"]);
  });

  it("does not replay a mutation after an unrelated forbidden response", async () => {
    const fetch = vi.fn(async (request: Request) => request.url.endsWith("/auth/csrf")
      ? jsonResponse({ csrfToken: "token" })
      : jsonResponse({ error: "Access denied" }, 403));
    vi.stubGlobal("fetch", fetch);
    const { baseQueryWithReauth } = await import("./baseQuery");
    const result = await baseQueryWithReauth({ url: "/reservations", method: "POST" }, api, {});
    expect(result.error?.status).toBe(403);
    expect(fetch).toHaveBeenCalledTimes(2);
  });

  it("includes the challenge header on refresh and retried mutations", async () => {
    let mutations = 0;
    const requests: Request[] = [];
    vi.stubGlobal("fetch", vi.fn(async (request: Request) => {
      requests.push(request);
      if (request.url.endsWith("/auth/csrf")) return jsonResponse({ csrfToken: "token" });
      if (request.url.endsWith("/auth/refresh")) return jsonResponse({ ok: true });
      return ++mutations === 1 ? jsonResponse({ error: "Unauthorized" }, 401) : jsonResponse({ ok: true });
    }));
    const { baseQueryWithReauth } = await import("./baseQuery");
    await baseQueryWithReauth({ url: "/reservations", method: "POST" }, api, {});
    expect(requests.map((request) => new URL(request.url).pathname)).toEqual([
      "/api/auth/csrf", "/api/reservations", "/api/auth/refresh", "/api/reservations",
    ]);
    expect(requests.slice(1).every((request) => request.headers.get("X-CSRF-Token") === "token")).toBe(true);
  });
});
