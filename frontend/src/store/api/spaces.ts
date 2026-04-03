import { createApi, fetchBaseQuery } from "@reduxjs/toolkit/query/react";
import { Space } from "../../types";
import { BACKEND_URL } from "../../config";

export const spacesApiSlice = createApi({
  reducerPath: "spaces",
  baseQuery: fetchBaseQuery({
    baseUrl: `${BACKEND_URL}/api`,
    credentials: "include",
  }),
  tagTypes: ["Space"],
  endpoints: (builder) => {
    return {
      getSpaces: builder.query<Space[], void>({
        query: () => "/spaces",
        providesTags: ["Space"],
      }),
      getSpace: builder.query<Space, string>({
        query: (id) => `/spaces/${id}`,
        providesTags: ["Space"],
      }),
      createSpace: builder.mutation<Space, Partial<Space>>({
        query: (newSpace) => ({
          url: "/spaces",
          method: "POST",
          body: newSpace,
        }),
        invalidatesTags: ["Space"],
      }),
      updateSpace: builder.mutation<Space, Partial<Space> & Pick<Space, "id">>({
        query: ({ id, ...patch }) => ({
          url: `/spaces/${id}`,
          method: "PATCH",
          body: patch,
        }),
        invalidatesTags: ["Space"],
      }),
      deleteSpace: builder.mutation<void, string>({
        query: (id) => ({
          url: `/spaces/${id}`,
          method: "DELETE",
        }),
        invalidatesTags: ["Space"],
      }),
      // Permissions
      getSpacePermissions: builder.query<import("../../types").SpacePermission[], string>({
        query: (spaceId) => `/spaces/${spaceId}/permissions`,
        providesTags: ["Space"], // Re-fetch on space changes or permission updates
      }),
      addSpacePermission: builder.mutation<void, { spaceId: string; groupId?: string; userId?: string }>({
        query: ({ spaceId, ...body }) => ({
          url: `/spaces/${spaceId}/permissions`,
          method: "POST",
          body,
        }),
        invalidatesTags: ["Space"],
      }),
      removeSpacePermission: builder.mutation<void, { spaceId: string; groupId?: string; userId?: string }>({
        query: ({ spaceId, ...body }) => ({
          url: `/spaces/${spaceId}/permissions`,
          method: "DELETE",
          body,
        }),
        invalidatesTags: ["Space"],
      }),
    };
  },
});

export const {
  useGetSpacesQuery,
  useGetSpaceQuery,
  useCreateSpaceMutation,
  useUpdateSpaceMutation,
  useDeleteSpaceMutation,
  useGetSpacePermissionsQuery,
  useAddSpacePermissionMutation,
  useRemoveSpacePermissionMutation,
} = spacesApiSlice;
