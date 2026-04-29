import { createApi } from "@reduxjs/toolkit/query/react";
import { User, CreateUserRequest } from "./auth";
import { baseQueryWithReauth } from "./baseQuery";

export const userApiSlice = createApi({
  reducerPath: "userApi",
  baseQuery: baseQueryWithReauth,
  tagTypes: ["User"],
  endpoints: (builder) => {
    return {
      getUsers: builder.query<User[], void>({
        query: () => "/users",
        providesTags: ["User"],
      }),
      getUserById: builder.query<User, string>({
        query: (id) => `/users/${id}`,
      }),
      createUser: builder.mutation<void, CreateUserRequest>({
        query: (newUser) => ({
          url: "/users",
          method: "POST",
          body: newUser,
        }),
        invalidatesTags: ["User"],
      }),
      updateUser: builder.mutation<void, Partial<User>>({
        query: (updatedUser) => ({
          url: `/users/${updatedUser.id}`,
          method: "PATCH",
          body: updatedUser,
        }),
        invalidatesTags: ["User"],
      }),
      deleteUser: builder.mutation<void, string>({
        query: (id) => ({
          url: `/users/${id}`,
          method: "DELETE",
        }),
        invalidatesTags: ["User"],
      }),
      changePassword: builder.mutation<void, { current_password: string; new_password: string }>({
        query: (body) => ({
          url: "/user/password",
          method: "POST",
          body,
        }),
      }),
    };
  },
});

export const {
  useGetUsersQuery,
  useGetUserByIdQuery,
  useCreateUserMutation,
  useUpdateUserMutation,
  useDeleteUserMutation,
  useChangePasswordMutation,
} = userApiSlice;
export default userApiSlice;
