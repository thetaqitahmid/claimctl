import { createApi } from "@reduxjs/toolkit/query/react";
import { baseQueryWithReauth } from "./baseQuery";

export interface Secret {
  id: string;
  key: string;
  value: string;
  description: string;
  createdAt: number;
  updatedAt: number;
}

export const secretsApi = createApi({
  reducerPath: "secretsApi",
  baseQuery: baseQueryWithReauth,
  tagTypes: ["Secret"],
  endpoints: (builder) => ({
    getSecrets: builder.query<Secret[], void>({
      query: () => "/secrets",
      providesTags: ["Secret"],
    }),
    createSecret: builder.mutation<
      Secret,
      { key: string; value: string; description: string }
    >({
      query: (body) => ({ url: "/secrets", method: "POST", body }),
      invalidatesTags: ["Secret"],
    }),
    updateSecret: builder.mutation<
      Secret,
      { id: string; key: string; value: string; description: string }
    >({
      query: ({ id, ...body }) => ({
        url: `/secrets/${id}`,
        method: "PUT",
        body,
      }),
      invalidatesTags: ["Secret"],
    }),
    deleteSecret: builder.mutation<void, string>({
      query: (id) => ({ url: `/secrets/${id}`, method: "DELETE" }),
      invalidatesTags: ["Secret"],
    }),
  }),
});

export const {
  useGetSecretsQuery,
  useCreateSecretMutation,
  useUpdateSecretMutation,
  useDeleteSecretMutation,
} = secretsApi;
