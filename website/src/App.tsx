import { BrowserRouter, Route, Routes } from "react-router-dom";
import Login from "./pages/Login";
import Landing from "./pages/Landing";
import Signup from "./pages/Signup";
import ProjectProgress from "./pages/ProjectProgress";
import DashboardLayout from "./components/DashboardLayout";
import Overview from "./pages/dashboard/Overview";
import Projects from "./pages/dashboard/Projects";
import Import from "./pages/dashboard/Import";
import Deploy from "./pages/dashboard/Deploy";
import Settings from "./pages/dashboard/Settings";
import Help from "./pages/dashboard/Help";
import { AuthProvider } from "./context/AuthContext";
import PrivateRoutes from "./components/PrivateRoutes";

const App = () => {
  return (
    <div className="font-open">
      <AuthProvider>
        <BrowserRouter>
          <Routes>
            <Route path="/" element={<Landing />} />
            <Route element={<Login />} path="/login" />
            <Route path="/signup" element={<Signup />} />

            <Route element={<PrivateRoutes />}>
              {/* Each dashboard tab is its own URL */}
              <Route path="/dashboard" element={<DashboardLayout />}>
                <Route index element={<Overview />} />
                <Route path="projects" element={<Projects />} />
                <Route path="import" element={<Import />} />
                <Route path="deploy" element={<Deploy />} />
                <Route path="settings" element={<Settings />} />
                <Route path="help" element={<Help />} />
              </Route>
              <Route path="/project/:id" element={<ProjectProgress />} />
            </Route>
          </Routes>
        </BrowserRouter>
      </AuthProvider>
    </div>
  );
};

export default App;
