import { ReactNode } from "react";
import { Navigate, Outlet } from "react-router-dom";
import { useAppSelector, RootState } from "../../store/store";

interface ProtectedRouteProps {
  allowedRoles?: string[];
  children?: ReactNode;
}

const ProtectedRoute = ({ allowedRoles = [] }: ProtectedRouteProps) => {
  const user = useAppSelector((state: RootState) => state.authSlice.user);
  const role = useAppSelector((state: RootState) => state.authSlice.role);

  if (!user) {
    // If somehow reached without user context, navigate to login
    return <Navigate to="/login" replace />;
  }

  if (allowedRoles.length > 0 && (!role || !allowedRoles.includes(role))) {
    // User does not have the required role
    return <Navigate to="/" replace />;
  }

  // Render the child routes
  return <Outlet />;
};

export default ProtectedRoute;
