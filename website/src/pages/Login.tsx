import { Link } from "react-router-dom";

const Login = () => {
  return (
    <div className="flex min-h-screen items-center justify-center p-4">
      {/* Login Card */}
      <div className="w-full max-w-md rounded-xl border border-zinc-800 bg-zinc-950 p-8 shadow-2xl">
        {/* Header */}
        <div className="mb-8 text-center">
          <h1 className="text-2xl font-bold text-white tracking-tight">
            Login into the orcs
          </h1>
          <p className="mt-2 text-sm text-zinc-400">
            Enter your credentials to enter the horde.
          </p>
        </div>

        {/* Form */}
        <form className="space-y-6">
          <div>
            <label
              htmlFor="email"
              className="block text-sm font-medium text-zinc-300 mb-2"
            >
              Email Address
            </label>
            <input
              type="email"
              id="email"
              placeholder="name@clan.com"
              className="w-full rounded-lg border border-zinc-700 px-4 py-2.5 text-white placeholder-zinc-500 outline-none transition focus:border-zinc-600 focus:ring-1 focus:ring-zinc-600"
            />
          </div>

          <div>
            <label
              htmlFor="password"
              className="block text-sm font-medium text-zinc-300 mb-2"
            >
              Password
            </label>
            <input
              type="password"
              id="password"
              placeholder="••••••••"
              className="w-full rounded-lg border border-zinc-700 px-4 py-2.5 text-white placeholder-zinc-500 outline-none transition focus:border-zinc-600 focus:ring-1 focus:ring-zinc-600"
            />
          </div>

          <button
            type="submit"
            className="w-full rounded-lg bg-zinc-900 px-4 py-2.5 font-semibold text-white transition hover:bg-zinc-800 active:transform active:scale-[0.98]"
          >
            Sign In
          </button>
        </form>

        {/* Footer */}
        <div className="mt-6 text-center text-sm">
          <span className="text-zinc-500">New here? </span>
          <Link
            to="/signup"
            className="font-medium text-zinc-400 hover:text-zinc-400 underline-offset-4 hover:underline"
          >
            Join the ranks
          </Link>
        </div>
      </div>
    </div>
  );
};

export default Login;
