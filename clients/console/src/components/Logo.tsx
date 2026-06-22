import { cn } from "@/lib/cn";

// Inline brand mark so it inherits the ember gradient and stays crisp at any size
// without a network round-trip. The favicon variant lives in public/dante-mark.svg.
export function Logo({ className, showWord = true }: { className?: string; showWord?: boolean }) {
  return (
    <div className={cn("flex items-center gap-2.5", className)}>
      <svg width="32" height="32" viewBox="0 0 64 64" fill="none" className="shrink-0">
        <defs>
          <linearGradient id="logo-ember" x1="16" y1="6" x2="48" y2="58" gradientUnits="userSpaceOnUse">
            <stop stopColor="#ff7438" />
            <stop offset="1" stopColor="#f03d0c" />
          </linearGradient>
        </defs>
        <rect width="64" height="64" rx="16" fill="#101014" />
        <path
          d="M22 14h10c10 0 17 7.5 17 18s-7 18-17 18H22V14Z"
          fill="none"
          stroke="url(#logo-ember)"
          strokeWidth="4.5"
          strokeLinejoin="round"
        />
        <path
          d="M33 24c4 2.5 5.5 6 4 9.5-1 2.4-3.4 3.5-3 6.2.2 1.6 1.3 2.8 2.7 3.3-3.8.6-7.2-1.9-7.7-5.6-.5-3.6 2.2-5.3 2.3-8.1.04-1.9-.8-3.6-2.3-5.3 1.6.2 3 .9 4 1Z"
          fill="url(#logo-ember)"
        />
      </svg>
      {showWord && (
        <span className="font-display text-lg font-bold tracking-tight text-ink-50">
          Dante<span className="text-ember-gradient">GPU</span>
        </span>
      )}
    </div>
  );
}
