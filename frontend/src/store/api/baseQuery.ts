import {
  fetchBaseQuery,
  BaseQueryFn,
  FetchArgs,
  FetchBaseQueryError,
} from "@reduxjs/toolkit/query/react";
import { clearCredentials } from "../slices/authSlice";
import { BACKEND_URL } from "../../config";

const rawBaseQuery = fetchBaseQuery({
  baseUrl: `${BACKEND_URL}/api`,
  credentials: "include",
});

let csrfToken: string | undefined;
let csrfBootstrap: Promise<string | undefined> | undefined;

const csrfBaseQuery: typeof rawBaseQuery = async (args, api, extraOptions) => {
  const request = typeof args === "string" ? { url: args } : args;
  const unsafe = !["GET", "HEAD", "OPTIONS"].includes(
    (request.method ?? "GET").toUpperCase()
  );
  if (!unsafe) return rawBaseQuery(args, api, extraOptions);

  const bootstrap = async () => {
    csrfBootstrap ??= Promise.resolve(rawBaseQuery("/auth/csrf", api, extraOptions))
      .then((result) => {
        const data = result.data as { csrfToken?: string } | undefined;
        csrfToken = data?.csrfToken;
        return csrfToken;
      })
      .finally(() => { csrfBootstrap = undefined; });
    return csrfBootstrap;
  };
  if (!csrfToken) await bootstrap();
  const send = () => {
    const headers = new Headers();
    if (request.headers instanceof Headers) {
      request.headers.forEach((value, key) => headers.set(key, value));
    } else {
      const entries = Array.isArray(request.headers)
        ? request.headers
        : Object.entries(request.headers ?? {});
      for (const [key, value] of entries) {
        if (value !== undefined) headers.set(key, value);
      }
    }
    if (csrfToken) headers.set("X-CSRF-Token", csrfToken);
    return rawBaseQuery({ ...request, headers }, api, extraOptions);
  };
  let result = await send();
  const errorData = result.error?.data as { code?: string } | undefined;
  if (result.error?.status === 403 && errorData?.code === "CSRF_INVALID") {
    csrfToken = undefined;
    await bootstrap();
    result = await send();
  }
  return result;
};

export const baseQueryWithReauth: BaseQueryFn<
  string | FetchArgs,
  unknown,
  FetchBaseQueryError
> = async (args, api, extraOptions) => {
  let result = await csrfBaseQuery(args, api, extraOptions);

  if (result.error?.status === 401) {
    // Attempt silent refresh
    const refreshResult = await csrfBaseQuery(
      { url: "/auth/refresh", method: "POST" },
      api,
      extraOptions
    );

    if (refreshResult.error) {
      (api.dispatch)(clearCredentials());
    } else {
      // Retry the original request
      result = await csrfBaseQuery(args, api, extraOptions);
    }
  }

  return result;
};
