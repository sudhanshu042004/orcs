import { Github } from "lucide-react";
import { apiUrl } from "../utils/contants";

const Login = () => {
  const handleGithubLogin = async () => {
   return window.location.href = `${apiUrl + 'login'}`
  };

  return (
    <div className="flex min-h-screen items-center justify-center p-4 bg-black">
      {/* Login Card */}
      <div className="w-full max-w-sm rounded-2xl border border-zinc-800 bg-zinc-950 p-10 shadow-2xl text-center">
        
        {/* Header */}
        <div className="mb-10">
          <h1 className="text-2xl font-bold text-white tracking-tight">
            Login into the orcs
          </h1>
          <p className="mt-2 text-sm text-zinc-500">
            One-click entry to the horde.
          </p>
        </div>

        {/* GitHub Button */}
        <button
          onClick={handleGithubLogin}
          className="group cursor-pointer relative flex w-full items-center justify-center gap-3 rounded-lg bg-white px-4 py-3 text-sm font-bold text-black transition-all hover:bg-zinc-200 active:scale-[0.98]"
        >
          <Github className="h-5 w-5" />
          Continue with GitHub
        </button>

        {/* Footer */}
        <div className="mt-8 pt-6 border-t border-zinc-900">
          <p className="text-xs text-zinc-600 leading-relaxed">
            By joining, you agree to our <br />
            <span className="text-zinc-500 underline underline-offset-2 cursor-pointer hover:text-zinc-400">Code of Honor</span> and <span className="text-zinc-500 underline underline-offset-2 cursor-pointer hover:text-zinc-400">Terms</span>.
          </p>
        </div>
      </div>
    </div>
  );
};

export default Login;