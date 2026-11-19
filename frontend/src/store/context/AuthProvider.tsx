import { useEffect, ReactNode } from "react";

import { useAppSelector } from "../store";
import { useGetMeQuery, useGetSetupStatusQuery } from "../api/auth";
import { useNavigate, useLocation } from "react-router-dom";

type AuthProviderProps = {
  children: ReactNode;
};

function AuthProvider({ children }: AuthProviderProps) {
  const publicPaths = ["/login", "/setup"];
  const navigate = useNavigate();
  const location = useLocation();
  const {
    data: setupStatus,
    isLoading: setupLoading,
  } = useGetSetupStatusQuery();
  const needsSetup = setupStatus?.needsSetup === true;
  const onSetupPage = location.pathname === "/setup";
  const { isLoading, isError } = useGetMeQuery(undefined, {
    skip: setupLoading || needsSetup || onSetupPage,
  });
  const stateUser = useAppSelector((state) => state.authSlice.user);
  const skipAuthCheck = publicPaths.includes(location.pathname);

  useEffect(() => {
    if (setupLoading) return;

    if (needsSetup && !stateUser) {
      if (location.pathname !== "/setup") {
        navigate("/setup", { replace: true });
      }
      return;
    }

    if (skipAuthCheck) return;

    if (isError || (!isLoading && !stateUser)) {
      navigate("/login", { replace: true });
    }
  }, [
    isError,
    isLoading,
    location,
    stateUser,
    navigate,
    skipAuthCheck,
    needsSetup,
    setupLoading,
  ]);

  if (setupLoading || isLoading) {
    return <div>Loading...</div>;
  }

  return children;
}

export default AuthProvider;
