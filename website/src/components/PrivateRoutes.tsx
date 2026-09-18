import { Navigate, Outlet, useLocation } from "react-router-dom";
import { useAuth } from "../hooks/useAuth";

const PrivateRoutes = () => {
  const { user, loading } = useAuth();
  const location = useLocation();

  if (loading) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-blue-500"></div>
      </div>
    );
  }

  if (user) {
    return <Outlet />;
  }

  // remember where they were headed so login can send them back there
  const callback = location.pathname + location.search;

  return (
    <Navigate to={`/login?callback=${encodeURIComponent(callback)}`} replace />
  );
};

export default PrivateRoutes;
