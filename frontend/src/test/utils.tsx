import React, { ReactElement } from 'react';
import { render, RenderOptions } from '@testing-library/react';
import { Provider } from 'react-redux';
import { combineReducers, configureStore } from '@reduxjs/toolkit';
import { MemoryRouter } from 'react-router-dom';
import authSlice from '../store/slices/authSlice';
import resourcesUiSlice from '../store/slices/resourcesUiSlice';
import { resourcesApiSlice } from '../store/api/resources';
import { authApiSlice } from '../store/api/auth';
import { spacesApiSlice } from '../store/api/spaces';
import userApiSlice from '../store/api/users';
import { reservationsApiSlice } from '../store/api/reservations';
import { settingsApi } from '../store/api/settings';
import { preferencesApi } from '../store/api/preferences';
import { tokensApiSlice } from '../store/api/tokens';
import { groupsApiSlice } from '../store/api/groups';

// Combine reducers for test store
const rootReducer = combineReducers({
  [resourcesApiSlice.reducerPath]: resourcesApiSlice.reducer,
  [authApiSlice.reducerPath]: authApiSlice.reducer,
  [userApiSlice.reducerPath]: userApiSlice.reducer,
  [reservationsApiSlice.reducerPath]: reservationsApiSlice.reducer,
  [spacesApiSlice.reducerPath]: spacesApiSlice.reducer,
  [settingsApi.reducerPath]: settingsApi.reducer,
  [preferencesApi.reducerPath]: preferencesApi.reducer,
  [tokensApiSlice.reducerPath]: tokensApiSlice.reducer,
  [groupsApiSlice.reducerPath]: groupsApiSlice.reducer,
  authSlice,
  resourcesUiSlice,
});

type TestRootState = ReturnType<typeof rootReducer>;

function createTestStore(preloadedState?: Partial<TestRootState>) {
  return configureStore({
    reducer: rootReducer,
    middleware: (getDefaultMiddleware) =>
      getDefaultMiddleware().concat(
        resourcesApiSlice.middleware,
        authApiSlice.middleware,
        userApiSlice.middleware,
        reservationsApiSlice.middleware,
        spacesApiSlice.middleware,
        settingsApi.middleware,
        preferencesApi.middleware,
        tokensApiSlice.middleware,
        groupsApiSlice.middleware
      ),
    preloadedState,
  });
}

type AppStore = ReturnType<typeof createTestStore>;

interface ExtendedRenderOptions extends Omit<RenderOptions, 'queries'> {
  preloadedState?: Partial<TestRootState>;
  store?: AppStore;
  route?: string;
}

export function renderWithProviders(
  ui: ReactElement,
  {
    preloadedState = {},
    store = createTestStore(preloadedState),
    route = '/',
    ...renderOptions
  }: ExtendedRenderOptions = {}
) {
  function Wrapper({ children }: { children: React.ReactNode }) {
    return (
      <Provider store={store}>
        <MemoryRouter initialEntries={[route]}>
          {children}
        </MemoryRouter>
      </Provider>
    );
  }

  return {
    store,
    ...render(ui, { wrapper: Wrapper, ...renderOptions }),
  };
}

// Re-export everything from testing-library
// eslint-disable-next-line react-refresh/only-export-components
export * from '@testing-library/react';
export { renderWithProviders as render };
