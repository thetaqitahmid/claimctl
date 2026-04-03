import { createApi, fetchBaseQuery } from "@reduxjs/toolkit/query/react";
import { Resource, ResourceWithStatus, HealthConfig, HealthStatusData } from "../../types";
import { BACKEND_URL } from "../../config";

export const resourcesApiSlice = createApi({
  reducerPath: "resources",
  baseQuery: fetchBaseQuery({
    baseUrl: `${BACKEND_URL}/api`,
    credentials: "include",
  }),
  tagTypes: ["Resource", "HealthConfig", "HealthStatus"],
  endpoints: (builder) => {
    return {
      getResources: builder.query<ResourceWithStatus[], void>({
        query: () => "/resources/with-status",
        providesTags: ["Resource"],
      }),
      getResourceWithStatus: builder.query<ResourceWithStatus, string>({
        query: (id) => `/resources/${id}/with-status`,
        providesTags: ["Resource", "HealthStatus"],
      }),
      updateResource: builder.mutation<
        Resource,
        Partial<Resource> & Pick<Resource, "id">
      >({
        query: ({ id, ...patch }) => ({
          url: `/resources/${id}`,
          method: "PATCH",
          body: patch,
        }),
        invalidatesTags: ["Resource"],
      }),
      deleteResource: builder.mutation<Resource, string>({
        query: (id) => ({
          url: `/resources/${id}`,
          method: "DELETE",
        }),
        invalidatesTags: ["Resource"],
      }),
      createResource: builder.mutation<Resource, Partial<Resource>>({
        query: (newResource) => ({
          url: "/resources",
          method: "POST",
          body: newResource,
        }),
        invalidatesTags: ["Resource"],
      }),

      // Health Check endpoints
      getHealthConfig: builder.query<HealthConfig, string>({
        query: (resourceId) => `/resources/${resourceId}/health/config`,
        providesTags: ["HealthConfig"],
      }),
      updateHealthConfig: builder.mutation<HealthConfig, Partial<HealthConfig> & { resourceId: string }>({
        query: ({ resourceId, ...config }) => ({
          url: `/resources/${resourceId}/health/config`,
          method: "PUT",
          body: config,
        }),
        invalidatesTags: ["HealthConfig", "Resource"],
      }),
      deleteHealthConfig: builder.mutation<void, string>({
        query: (resourceId) => ({
          url: `/resources/${resourceId}/health/config`,
          method: "DELETE",
        }),
        invalidatesTags: ["HealthConfig", "Resource"],
      }),
      getHealthStatus: builder.query<HealthStatusData, string>({
        query: (resourceId) => `/resources/${resourceId}/health/status`,
        providesTags: ["HealthStatus"],
      }),
      getHealthHistory: builder.query<HealthStatusData[], { resourceId: string; limit?: number }>({
        query: ({ resourceId, limit = 10 }) => `/resources/${resourceId}/health/history?limit=${limit}`,
      }),
      triggerHealthCheck: builder.mutation<{ message: string }, string>({
        query: (resourceId) => ({
          url: `/resources/${resourceId}/health/check`,
          method: "POST",
        }),
        invalidatesTags: ["HealthStatus"],
      }),

      // Maintenance endpoints
      setMaintenanceMode: builder.mutation<
        Resource,
        { resourceId: string; isUnderMaintenance: boolean; reason?: string }
      >({
        query: ({ resourceId, isUnderMaintenance, reason }) => ({
          url: `/resources/${resourceId}/maintenance`,
          method: "PUT",
          body: { is_under_maintenance: isUnderMaintenance, reason },
        }),
        invalidatesTags: ["Resource"],
      }),
      getMaintenanceHistory: builder.query<
        { id: string; resourceId: string; previousState: boolean; newState: boolean; changedBy: string; changedAt: number; reason?: string; changedByEmail: string }[],
        string
      >({
        query: (resourceId) => `/resources/${resourceId}/maintenance/history`,
      }),
    };
  },
});

export const {
  useGetResourcesQuery,
  useGetResourceWithStatusQuery,
  useUpdateResourceMutation,
  useDeleteResourceMutation,
  useCreateResourceMutation,
  useGetHealthConfigQuery,
  useUpdateHealthConfigMutation,
  useDeleteHealthConfigMutation,
  useGetHealthStatusQuery,
  useGetHealthHistoryQuery,
  useTriggerHealthCheckMutation,
  useSetMaintenanceModeMutation,
  useGetMaintenanceHistoryQuery,
} = resourcesApiSlice;
