import { createApi } from '@reduxjs/toolkit/query/react';
import { baseQueryWithReauth } from './baseQuery';

export interface AppSetting {
  key: string;
  value: string;
  category: string;
  description: string;
  is_secret: boolean;
}

export const settingsApi = createApi({
  reducerPath: 'settingsApi',
  baseQuery: baseQueryWithReauth,
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
