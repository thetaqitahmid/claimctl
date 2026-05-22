import { createApi } from "@reduxjs/toolkit/query/react";
import { clearCredentials, setCredentials } from "../slices/authSlice";
import { baseQueryWithReauth } from "./baseQuery";

interface credentialType {
  email: string;
  password: string;
}

export interface User {
  id: string;
  email: string;
  name: string;
  role: string;
  lastLogin?: string;
  status: string;
  createdAt?: string;
  updatedAt?: string;
  authProvider?: string;
}

export interface CreateUserRequest {
  name: string;
  email: string;
  password?: string;
  role: string;

  status: string;
}

interface authResponse {
  user: User;
}

export const authApiSlice = createApi({
  reducerPath: "auth",
  tagTypes: ["User"],
  baseQuery: baseQueryWithReauth,
  endpoints: (builder) => ({
    login: builder.mutation<authResponse, credentialType>({
      query: (credentials: credentialType) => ({
        url: "/login",
        method: "POST",
        body: credentials,
      }),
      invalidatesTags: ["User"],
    }),
    loginLDAP: builder.mutation<authResponse, credentialType>({
      query: (credentials: credentialType) => ({
        url: "/auth/ldap",
        method: "POST",
        body: credentials,
      }),
      invalidatesTags: ["User"],
    }),
    logout: builder.mutation<void, void>({
      query: () => ({
        url: "/logout",
        method: "POST",
      }),
      invalidatesTags: ["User"],
      async onQueryStarted(_, { dispatch, queryFulfilled }) {
        try {
          await queryFulfilled;
          dispatch(clearCredentials());
        } catch (error) {
          console.error(error);
        }
      },
    }),
    getMe: builder.query<authResponse, void>({
      query: () => "/me",
      providesTags: ["User"],
      async onQueryStarted(_, { dispatch, queryFulfilled }) {
        try {
          const {
            data: { user },
          } = await queryFulfilled;
          dispatch(
            setCredentials({
              email: user.email,
              user: user.name,
              role: user.role,
              authProvider: user.authProvider ?? null,
            })
          );
        } catch (error) {
          console.error(error);
        }
      },
    }),
  }),
});

export const { useLoginMutation, useLogoutMutation, useGetMeQuery, useLoginLDAPMutation } =
  authApiSlice;
