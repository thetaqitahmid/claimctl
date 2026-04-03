import { createApi, fetchBaseQuery } from "@reduxjs/toolkit/query/react";
import { BACKEND_URL } from "../../config";

export interface CreateReservationRequest {
  resourceId: string;
}

export const reservationsApiSlice = createApi({
  reducerPath: "reservations",
  baseQuery: fetchBaseQuery({
    baseUrl: `${BACKEND_URL}/api`,
    credentials: "include",
  }),
  tagTypes: ["Reservation"],
  endpoints: (builder) => {
    return {
      createReservation: builder.mutation<void, CreateReservationRequest>({
        query: (request) => ({
          url: "/reservations",
          method: "POST",
          body: request,
        }),
        invalidatesTags: ["Reservation"],
      }),
      createTimedReservation: builder.mutation<void, { resourceId: string; duration: string }>({
        query: (request) => ({
          url: "/reservations/timed",
          method: "POST",
          body: request,
        }),
        invalidatesTags: ["Reservation"],
      }),
      cancelReservation: builder.mutation<void, string>({
        query: (reservationId) => ({
          url: `/reservations/${reservationId}/cancel`,
          method: "PATCH",
        }),
        invalidatesTags: ["Reservation"],
      }),
      cancelAllReservations: builder.mutation<void, string>({
        query: (resourceId) => ({
          url: `/admin/resources/${resourceId}/reservations`,
          method: "DELETE",
        }),
        invalidatesTags: ["Reservation"],
      }),
      completeReservation: builder.mutation<void, string>({
        query: (reservationId) => ({
          url: `/reservations/${reservationId}/complete`,
          method: "PATCH",
        }),
        invalidatesTags: ["Reservation"],
      }),
      getUserReservations: builder.query<{ id: string; resourceId: string; status: string; queuePosition: number }[], void>({
        query: () => "/reservations",
        providesTags: ["Reservation"],
      }),
      getQueueForResource: builder.query<{ id: string; userId: string; status: string; queuePosition: number; startTime: number; createdAt: number; userName: string; userEmail: string; duration?: string }[], string>({
        query: (resourceId) => `/resources/${resourceId}/queue`,
        providesTags: ["Reservation"],
      }),
      getUserHistory: builder.query<{ id: string; resourceId: string; resourceName: string; reservationId: string | null; action: string; timestamp: number; details: string }[], void>({
        query: () => "/me/history",
      }),
      getResourceHistory: builder.query<{ id: string; resourceId: string; resourceName: string; reservationId: string | null; action: string; timestamp: number; details: string; userName: string }[], string>({
        query: (resourceId) => `/resources/${resourceId}/history`,
      }),
    };
  },
});

export const {
  useCreateReservationMutation,
  useCreateTimedReservationMutation,
  useCancelReservationMutation,
  useCancelAllReservationsMutation,
  useCompleteReservationMutation,
  useGetUserReservationsQuery,
  useGetQueueForResourceQuery,
  useGetUserHistoryQuery,
  useGetResourceHistoryQuery,
} = reservationsApiSlice;