import { useState, type FormEvent } from "react";
import { Link, useNavigate } from "react-router-dom";
import { Lock, User } from "lucide-react";
import { AuthShell } from "@/components/layout/AuthShell";
import { Input } from "@/components/ui/Input";
import { Button } from "@/components/ui/Button";
import { useAuth } from "@/providers/AuthProvider";
import { ApiError } from "@/lib/api";

export default function Login() {
  const { login } = useAuth();
  const navigate = useNavigate();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setBusy(true);
    try {
      await login(username.trim(), password);
      navigate("/", { replace: true });
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not sign in. Check your credentials.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <AuthShell>
      <h2 className="text-2xl font-bold text-ink-50">Welcome back</h2>
      <p className="mt-1 text-sm text-ink-400">Sign in to manage your rentals and balance.</p>

      <form onSubmit={onSubmit} className="mt-7 space-y-4">
        <Input
          name="username"
          label="Username"
          placeholder="your-handle"
          autoComplete="username"
          leading={<User className="size-4" />}
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          required
        />
        <Input
          name="password"
          type="password"
          label="Password"
          placeholder="••••••••"
          autoComplete="current-password"
          leading={<Lock className="size-4" />}
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          required
        />
        {error && (
          <div className="rounded-lg border border-critical/30 bg-critical/10 px-3 py-2 text-sm text-critical">
            {error}
          </div>
        )}
        <Button type="submit" size="lg" loading={busy} className="w-full">
          Sign in
        </Button>
      </form>

      <p className="mt-6 text-center text-sm text-ink-400">
        New to DanteGPU?{" "}
        <Link to="/register" className="font-medium text-ember-300 hover:text-ember-200">
          Create an account
        </Link>
      </p>
    </AuthShell>
  );
}
