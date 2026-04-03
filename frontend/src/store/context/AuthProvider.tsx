import { useEffect, ReactNode } from "react";

import { useAppSelector } from "../store";
import { useGetMeQuery } from "../api/auth";
import { useNavigate, useLocation } from "react-router-dom";

type AuthProviderProps = {
  children: ReactNode;
};

function AuthProvider({ children }: AuthProviderProps) {
  const publicPaths = ["/login"];
  const { isLoading, isError } = useGetMeQuery();
  const stateUser = useAppSelector((state) => state.authSlice.user);
  const navigate = useNavigate();
  const location = useLocation();
  const skipAuthCheck = publicPaths.includes(location.pathname);

  useEffect(() => {
    if (skipAuthCheck) return;

    if (isError || (!isLoading && !stateUser)) {
      console.error(
        "AuthProvider: user=",
        stateUser,
        "isError=",
        isError,
        "path=",
        location.pathname
      );
      navigate("/login", { replace: true });
    }
  }, [isError, isLoading, location, stateUser, navigate, skipAuthCheck]);

  if (isLoading) {
    return <div>Loading...</div>;
  }

  return children;
}

export default AuthProvider;
