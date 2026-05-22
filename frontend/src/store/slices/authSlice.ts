import { createSlice, PayloadAction } from "@reduxjs/toolkit";

interface CredentialsPayload {
  user: string | null;
  email: string | null;
  role: string | null;
  authProvider: string | null;
}

const initialState: CredentialsPayload = {
  user: null,
  email: null,
  role: null,
  authProvider: null,
};

const authSlice = createSlice({
  name: "auth",
  initialState,
  reducers: {
    setCredentials: (state, action: PayloadAction<CredentialsPayload>) => {
      const { user, email, role, authProvider } = action.payload;
      state.user = user;
      state.email = email;
      state.role = role;
      state.authProvider = authProvider;
    },
    clearCredentials: (state) => {
      state.user = null;
      state.email = null;
      state.role = null;
      state.authProvider = null;
    },
  },
});

export const { setCredentials, clearCredentials } = authSlice.actions;
export default authSlice.reducer;
