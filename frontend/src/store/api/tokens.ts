import { createApi, fetchBaseQuery } from "@reduxjs/toolkit/query/react";
import { BACKEND_URL } from "../../config";

export interface ApiToken {
  id: string;
  name: string;
  token?: string; // Only present upon creation
  createdAt: string;
  expiresAt?: string;
  lastUsedAt?: string;
}

export interface CreateTokenRequest {
  name: string;
  expires_in?: string; // "30d", "1y", etc.
}

export interface CreateTokenResponse {
  token: string;
  id: string;
  name: string;
  createdAt: string;
  expiresAt?: string;
}

export const tokensApiSlice = createApi({
  reducerPath: "tokensApi",
  tagTypes: ["ApiToken"],
  baseQuery: fetchBaseQuery({
    baseUrl: `${BACKEND_URL}/api`,
    credentials: "include",
  }),
  endpoints: (builder) => ({
    getTokens: builder.query<ApiToken[], void>({
      query: () => "/tokens",
      providesTags: ["ApiToken"],
    }),
    createToken: builder.mutation<CreateTokenResponse, CreateTokenRequest>({
      query: (body) => ({
        url: "/tokens",
        method: "POST",
        body,
      }),
      invalidatesTags: ["ApiToken"],
    }),
    revokeToken: builder.mutation<void, string>({
      query: (id) => ({
        url: `/tokens/${id}`,
        method: "DELETE",
      }),
      invalidatesTags: ["ApiToken"],
    }),
  }),
});

export const {
  useGetTokensQuery,
  useCreateTokenMutation,
  useRevokeTokenMutation,
} = tokensApiSlice;
