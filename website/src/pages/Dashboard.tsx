import { UserAvatar } from "@/components/UserAvatar";
import { useAuth } from "../hooks/useAuth";

const Dashboard = () => {
  const { user } = useAuth();
  console.log(user);
  return (
    <div className="mx-auto max-w-7xl pt-10" >
      <div className="flex justify-between row">
        <div className="text-2xl">{user?.name}</div>
        <div>
          <UserAvatar />
        </div>
      </div>

      <div>
        <div>
          <button>deploy you project</button>
        </div>
      </div>
    </div>
  );
};

export default Dashboard;
