import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react';

export interface AppSetting {
  key: string;
  value: string;
  category: string;
  description: string;
  is_secret: boolean;
}

export const settingsApi = createApi({
  reducerPath: 'settingsApi',
  baseQuery: fetchBaseQuery({
    baseUrl: '/api',
    prepareHeaders: (headers) => {
      // Assuming headers are handled globally or via proxy, but if JWT is in cookie, we rely on browser.
      // If we needed to attach token:
      // const token = (getState() as RootState).auth.token;
      // if (token) headers.set('authorization', `Bearer ${token}`);
      return headers;
    },
  }),
  tagTypes: ['Settings'],
  endpoints: (builder) => ({
    getSettings: builder.query<AppSetting[], void>({
      query: () => '/admin/settings',
      providesTags: ['Settings'],
    }),
    updateSetting: builder.mutation<AppSetting, Partial<AppSetting>>({
      query: (setting) => ({
        url: '/admin/settings',
        method: 'PUT',
        body: setting,
      }),
      invalidatesTags: ['Settings'],
    }),
  }),
});

export const { useGetSettingsQuery, useUpdateSettingMutation } = settingsApi;
