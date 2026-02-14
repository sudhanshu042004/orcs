import { Link } from "react-router-dom";

const Signup = () => {
  return (
    <div className="flex min-h-screen items-center justify-center p-4">
      {/* Signup Card */}
      <div className="w-full max-w-md rounded-xl border border-zinc-800 bg-zinc-950 p-8 shadow-2xl">
        
        {/* Header */}
        <div className="mb-8 text-center">
          <h1 className="text-2xl font-bold text-white tracking-tight">
            Join the horde
          </h1>
          <p className="mt-2 text-sm text-zinc-400">
            Create your account to start your journey.
          </p>
        </div>

        {/* Form */}
        <form className="space-y-5">
          {/* Username Field */}
          <div>
            <label
              htmlFor="username"
              className="block text-sm font-medium text-zinc-300 mb-2"
            >
              Username
            </label>
            <input
              type="text"
              id="username"
              placeholder="@username"
              className="w-full rounded-lg border border-zinc-700 bg-transparent px-4 py-2.5 text-white placeholder-zinc-500 outline-none transition focus:border-zinc-600 focus:ring-1 focus:ring-zinc-600"
            />
          </div>

          {/* Email Field */}
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
              className="w-full rounded-lg border border-zinc-700 bg-transparent px-4 py-2.5 text-white placeholder-zinc-500 outline-none transition focus:border-zinc-600 focus:ring-1 focus:ring-zinc-600"
            />
          </div>

          {/* Password Field */}
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
              className="w-full rounded-lg border border-zinc-700 bg-transparent px-4 py-2.5 text-white placeholder-zinc-500 outline-none transition focus:border-zinc-600 focus:ring-1 focus:ring-zinc-600"
            />
          </div>

          {/* Submit Button */}
          <button
            type="submit"
            className="w-full mt-2 rounded-lg bg-zinc-100 px-4 py-2.5 font-semibold text-zinc-950 transition hover:bg-white active:transform active:scale-[0.98]"
          >
            Create Account
          </button>
        </form>

        {/* Footer */}
        <div className="mt-6 text-center text-sm">
          <span className="text-zinc-500">Already a member? </span>
          <Link
            to="/login"
            className="font-medium text-zinc-400 hover:text-zinc-300 underline-offset-4 hover:underline"
          >
            Log in here
          </Link>
        </div>
      </div>
    </div>
  );
};

export default Signup;