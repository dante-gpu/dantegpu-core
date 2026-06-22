import { Link } from "react-router-dom";
import { Button } from "@/components/ui/Button";

export default function NotFound() {
  return (
    <div className="flex min-h-[60vh] flex-col items-center justify-center gap-4 text-center">
      <p className="font-display text-7xl font-bold text-ember-gradient">404</p>
      <h1 className="text-xl font-semibold text-ink-50">This page drifted into the inferno</h1>
      <p className="max-w-sm text-sm text-ink-400">The page you are looking for does not exist or has moved.</p>
      <Link to="/">
        <Button>Back to dashboard</Button>
      </Link>
    </div>
  );
}
