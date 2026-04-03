import { createApi, fetchBaseQuery } from "@reduxjs/toolkit/query/react";
import { UserGroup, GroupMember } from "../../types";
import { BACKEND_URL } from "../../config";

export const groupsApiSlice = createApi({
  reducerPath: "groupsApi",
  baseQuery: fetchBaseQuery({
    baseUrl: `${BACKEND_URL}/api`,
    credentials: "include",
  }),
  tagTypes: ["Group", "GroupMember"],
  endpoints: (builder) => {
    return {
      getGroups: builder.query<UserGroup[], void>({
        query: () => "/groups",
        providesTags: ["Group"],
      }),
      getGroup: builder.query<UserGroup, string>({
        query: (id) => `/groups/${id}`,
        providesTags: ["Group"],
      }),
      createGroup: builder.mutation<UserGroup, Partial<UserGroup>>({
        query: (newGroup) => ({
          url: "/groups",
          method: "POST",
          body: newGroup,
        }),
        invalidatesTags: ["Group"],
      }),
      updateGroup: builder.mutation<UserGroup, Partial<UserGroup> & { id: string }>({
        query: ({ id, ...patch }) => ({
          url: `/groups/${id}`,
          method: "PATCH",
          body: patch,
        }),
        invalidatesTags: ["Group"],
      }),
      deleteGroup: builder.mutation<void, string>({
        query: (id) => ({
          url: `/groups/${id}`,
          method: "DELETE",
        }),
        invalidatesTags: ["Group"],
      }),
      // Members
      getGroupMembers: builder.query<GroupMember[], string>({
        query: (groupId) => `/groups/${groupId}/members`,
        providesTags: ["GroupMember"],
      }),
      addUserToGroup: builder.mutation<void, { groupId: string; userId: string }>({
        query: ({ groupId, userId }) => ({
          url: `/groups/${groupId}/members`,
          method: "POST",
          body: { userId },
        }),
        invalidatesTags: ["GroupMember"],
      }),
      removeUserFromGroup: builder.mutation<void, { groupId: string; userId: string }>({
        query: ({ groupId, userId }) => ({
          url: `/groups/${groupId}/members/${userId}`,
          method: "DELETE",
        }),
        invalidatesTags: ["GroupMember"],
      }),
    };
  },
});

export const {
  useGetGroupsQuery,
  useGetGroupQuery,
  useCreateGroupMutation,
  useUpdateGroupMutation,
  useDeleteGroupMutation,
  useGetGroupMembersQuery,
  useAddUserToGroupMutation,
  useRemoveUserFromGroupMutation,
} = groupsApiSlice;
