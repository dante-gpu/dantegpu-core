import { cn } from "@/lib/cn";

export function Skeleton({ className }: { className?: string }) {
  return <div className={cn("animate-pulse rounded-md bg-ink-700/60", className)} />;
}

export function SkeletonCard() {
  return (
    <div className="card p-5">
      <Skeleton className="h-4 w-1/3" />
      <Skeleton className="mt-4 h-8 w-2/3" />
      <Skeleton className="mt-3 h-3 w-1/2" />
    </div>
  );
}
