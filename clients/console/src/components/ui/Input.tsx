import { forwardRef, type InputHTMLAttributes, type ReactNode } from "react";
import { cn } from "@/lib/cn";

interface FieldProps extends InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  hint?: string;
  error?: string;
  leading?: ReactNode;
  trailing?: ReactNode;
}

export const Input = forwardRef<HTMLInputElement, FieldProps>(function Input(
  { label, hint, error, leading, trailing, className, id, ...props },
  ref,
) {
  const inputId = id ?? props.name;
  return (
    <label htmlFor={inputId} className="block">
      {label && <span className="mb-1.5 block text-sm font-medium text-ink-100">{label}</span>}
      <div
        className={cn(
          "flex items-center gap-2 rounded-lg border bg-ink-850 px-3 transition-colors",
          "focus-within:border-ember-500 focus-within:ring-2 focus-within:ring-ember-500/25",
          error ? "border-critical/60" : "border-ink-500",
        )}
      >
        {leading && <span className="text-ink-300">{leading}</span>}
        <input
          ref={ref}
          id={inputId}
          className={cn(
            "h-11 w-full bg-transparent text-sm text-ink-50 placeholder:text-ink-400 focus:outline-none",
            className,
          )}
          {...props}
        />
        {trailing && <span className="text-ink-300">{trailing}</span>}
      </div>
      {error ? (
        <span className="mt-1 block text-xs text-critical">{error}</span>
      ) : (
        hint && <span className="mt-1 block text-xs text-ink-400">{hint}</span>
      )}
    </label>
  );
});
