import { useState, type FormEvent } from "react";
import { Link, useNavigate } from "react-router-dom";
import { Lock, User, Mail } from "lucide-react";
import { AuthShell } from "@/components/layout/AuthShell";
import { Input } from "@/components/ui/Input";
import { Button } from "@/components/ui/Button";
import { useAuth } from "@/providers/AuthProvider";
import { ApiError } from "@/lib/api";

export default function Register() {
  const { register } = useAuth();
  const navigate = useNavigate();
  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    if (password.length < 8) {
      setError("Password must be at least 8 characters.");
      return;
    }
    setBusy(true);
    try {
      await register(username.trim(), email.trim(), password);
      navigate("/", { replace: true });
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not create the account. Try a different username.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <AuthShell>
      <h2 className="text-2xl font-bold text-ink-50">Create your account</h2>
      <p className="mt-1 text-sm text-ink-400">Start renting GPUs in minutes.</p>

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
          name="email"
          type="email"
          label="Email"
          placeholder="you@example.com"
          autoComplete="email"
          leading={<Mail className="size-4" />}
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          required
        />
        <Input
          name="password"
          type="password"
          label="Password"
          placeholder="At least 8 characters"
          autoComplete="new-password"
          leading={<Lock className="size-4" />}
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          hint="Use 8+ characters with a mix of letters and numbers."
          required
        />
        {error && (
          <div className="rounded-lg border border-critical/30 bg-critical/10 px-3 py-2 text-sm text-critical">
            {error}
          </div>
        )}
        <Button type="submit" size="lg" loading={busy} className="w-full">
          Create account
        </Button>
      </form>

      <p className="mt-6 text-center text-sm text-ink-400">
        Already have an account?{" "}
        <Link to="/login" className="font-medium text-ember-300 hover:text-ember-200">
          Sign in
        </Link>
      </p>
    </AuthShell>
  );
}
