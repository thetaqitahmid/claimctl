import { configureStore } from "@reduxjs/toolkit";
import { resourcesApiSlice } from "./api/resources";
import { authApiSlice } from "./api/auth";
import { spacesApiSlice } from "./api/spaces";
import authSlice from "./slices/authSlice";
import resourcesUiSlice from "./slices/resourcesUiSlice";
import userApiSlice from "./api/users";
import { reservationsApiSlice } from "./api/reservations";
import { settingsApi } from "./api/settings";
import { preferencesApi } from "./api/preferences";
import { tokensApiSlice } from "./api/tokens";
import { groupsApiSlice } from "./api/groups";
import {
  useDispatch,
  useSelector,
  TypedUseSelectorHook,
  useStore,
} from "react-redux";

export const store = configureStore({
  reducer: {
    [resourcesApiSlice.reducerPath]: resourcesApiSlice.reducer,
    [authApiSlice.reducerPath]: authApiSlice.reducer,
    [userApiSlice.reducerPath]: userApiSlice.reducer,
    [reservationsApiSlice.reducerPath]: reservationsApiSlice.reducer,
    [spacesApiSlice.reducerPath]: spacesApiSlice.reducer,
    [settingsApi.reducerPath]: settingsApi.reducer,
    [preferencesApi.reducerPath]: preferencesApi.reducer,
    [tokensApiSlice.reducerPath]: tokensApiSlice.reducer,
    [groupsApiSlice.reducerPath]: groupsApiSlice.reducer,
    authSlice: authSlice,
    resourcesUiSlice: resourcesUiSlice,
  },
  middleware: (getDefaultMiddleware) => {
    return getDefaultMiddleware()
      .concat(resourcesApiSlice.middleware)
      .concat(authApiSlice.middleware)
      .concat(userApiSlice.middleware)
      .concat(reservationsApiSlice.middleware)
      .concat(spacesApiSlice.middleware)
      .concat(settingsApi.middleware)
      .concat(preferencesApi.middleware)
      .concat(tokensApiSlice.middleware)
      .concat(groupsApiSlice.middleware);
  },
});

export type RootState = ReturnType<typeof store.getState>;
export type AppStore = typeof store;
export type AppDispatch = typeof store.dispatch;

// Export hooks for usage in functional components
export const useAppDispatch: () => AppDispatch = useDispatch;
export const useAppSelector: TypedUseSelectorHook<RootState> = useSelector;
export const useAppStore: () => AppStore = useStore;
