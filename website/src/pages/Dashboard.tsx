import { UserAvatar } from "@/components/UserAvatar";
import { useAuth } from "../hooks/useAuth";

const Dashboard = () => {
  const { user } = useAuth();
  console.log(user);
  return (
    <div>
     <UserAvatar/>
    </div>
  );
};

export default Dashboard;
