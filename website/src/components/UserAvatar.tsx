import { useAuth } from "../hooks/useAuth";
import { Avatar, AvatarImage } from "./ui/avatar";

export const UserAvatar = () => {
  const { user } = useAuth();
  return (
    <div title={user?.name ?? undefined} className="text-sm text-white">
      <Avatar>
        <AvatarImage src={user?.avatar} />
      </Avatar>
    </div>
  );
};
