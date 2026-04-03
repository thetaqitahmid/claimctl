import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react';

export interface UserNotificationPreference {
    userId: string;
    eventType: string;
    channel: string;
    enabled: boolean;
}

export interface UserChannelConfig {
    slack_destination: string;
    teams_webhook_url: string;
    notification_email?: string;
}

export const preferencesApi = createApi({
    reducerPath: 'preferencesApi',
    baseQuery: fetchBaseQuery({
        baseUrl: '/api',
        prepareHeaders: (headers) => {
            // JWT is handled by HTTP-only cookie, but if we need auth header fallback:
            return headers;
        },
    }),
    tagTypes: ['Preferences', 'UserConfig'],
    endpoints: (builder) => ({
        getPreferences: builder.query<UserNotificationPreference[], void>({
            query: () => '/me/preferences',
            providesTags: ['Preferences'],
        }),
        updatePreference: builder.mutation<UserNotificationPreference, Partial<UserNotificationPreference>>({
            query: (preference) => ({
                url: '/me/preferences',
                method: 'PUT',
                body: preference,
            }),
            invalidatesTags: ['Preferences'],
        }),
        getUserChannelConfig: builder.query<UserChannelConfig, void>({
             query: () => '/me', // Fallback if fields are there?
             transformResponse: (response: { user?: { slack_destination?: string; teams_webhook_url?: string; notification_email?: string } }) => ({
                 slack_destination: response.user?.slack_destination || '',
                 teams_webhook_url: response.user?.teams_webhook_url || '',
                 notification_email: response.user?.notification_email || ''
             }),
             providesTags: ['UserConfig']
        }),

        updateChannelConfig: builder.mutation<unknown, UserChannelConfig>({
            query: (config) => ({
                url: '/me/channel-config',
                method: 'PUT',
                body: config,
            }),
            invalidatesTags: ['UserConfig'],
        }),
        testUserEmailConfig: builder.mutation<{ message: string }, void>({
            query: () => ({
                url: '/me/test-email',
                method: 'POST',
            }),
        }),
    }),
});

export const {
    useGetPreferencesQuery,
    useUpdatePreferenceMutation,
    useUpdateChannelConfigMutation,
    useGetUserChannelConfigQuery,
    useTestUserEmailConfigMutation,
} = preferencesApi;
