import { createApi } from "@reduxjs/toolkit/query/react";
import { baseQueryWithReauth } from "./baseQuery";

export const backupApi = createApi({
  reducerPath: "backupApi",
  baseQuery: baseQueryWithReauth,
  endpoints: (builder) => ({
    createBackup: builder.mutation<Blob, void>({
      query: () => ({
        url: "/admin/backup",
        method: "GET",
        responseHandler: (response) => response.blob(),
      }),
    }),
    restoreBackup: builder.mutation<
      { message: string },
      { file: File; force?: boolean }
    >({
      query: ({ file, force }) => {
        const formData = new FormData();
        formData.append("file", file);
        return {
          url: `/admin/restore${force ? "?force=true" : ""}`,
          method: "POST",
          body: formData,
        };
      },
    }),
  }),
});

export const { useCreateBackupMutation, useRestoreBackupMutation } = backupApi;
