import { createApi } from "@reduxjs/toolkit/query/react";
import { baseQueryWithReauth } from "./baseQuery";

export interface Webhook {
  id: string;
  name: string;
  url: string;
  method: string;
  headers: string | Record<string, string>;
  template: string;
  description: string;
  signingSecret?: string;
}

export interface WebhookLog {
  id: string;
  event: string;
  statusCode: number;
  requestBody: string;
  responseBody: string;
  durationMs: number;
  createdAt: string;
}

export interface AssignedWebhook {
  id: string;
  name: string;
  events: string[];
}

export const webhooksApi = createApi({
  reducerPath: "webhooksApi",
  baseQuery: baseQueryWithReauth,
  tagTypes: ["Webhook", "WebhookLog", "ResourceWebhook"],
  endpoints: (builder) => ({
    listWebhooks: builder.query<Webhook[], void>({
      query: () => "/webhooks",
      providesTags: ["Webhook"],
    }),
    createWebhook: builder.mutation<
      Webhook,
      Partial<Webhook> & Pick<Webhook, "name" | "url" | "method">
    >({
      query: (body) => ({ url: "/webhooks", method: "POST", body }),
      invalidatesTags: ["Webhook"],
    }),
    updateWebhook: builder.mutation<
      Webhook,
      Partial<Webhook> & Pick<Webhook, "id">
    >({
      query: ({ id, ...body }) => ({
        url: `/webhooks/${id}`,
        method: "PUT",
        body,
      }),
      invalidatesTags: ["Webhook"],
    }),
    deleteWebhook: builder.mutation<void, string>({
      query: (id) => ({ url: `/webhooks/${id}`, method: "DELETE" }),
      invalidatesTags: ["Webhook"],
    }),
    getWebhookLogs: builder.query<WebhookLog[], string>({
      query: (webhookId) => `/webhooks/${webhookId}/logs`,
      providesTags: ["WebhookLog"],
    }),
    getResourceWebhooks: builder.query<AssignedWebhook[], string>({
      query: (resourceId) => `/resources/${resourceId}/webhooks`,
      providesTags: ["ResourceWebhook"],
    }),
    addResourceWebhook: builder.mutation<
      void,
      { resourceId: string; webhook_id: string; events: string[] }
    >({
      query: ({ resourceId, ...body }) => ({
        url: `/resources/${resourceId}/webhooks`,
        method: "POST",
        body,
      }),
      invalidatesTags: ["ResourceWebhook"],
    }),
    removeResourceWebhook: builder.mutation<
      void,
      { resourceId: string; webhookId: string }
    >({
      query: ({ resourceId, webhookId }) => ({
        url: `/resources/${resourceId}/webhooks/${webhookId}`,
        method: "DELETE",
      }),
      invalidatesTags: ["ResourceWebhook"],
    }),
  }),
});

export const {
  useListWebhooksQuery,
  useCreateWebhookMutation,
  useUpdateWebhookMutation,
  useDeleteWebhookMutation,
  useGetWebhookLogsQuery,
  useGetResourceWebhooksQuery,
  useAddResourceWebhookMutation,
  useRemoveResourceWebhookMutation,
} = webhooksApi;
