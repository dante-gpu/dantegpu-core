import * as React from "react"
import { cva, type VariantProps } from "class-variance-authority"
import { cn } from "../../lib/utils"

const badgeVariants = cva(
  "inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold transition-colors focus:outline-none focus:ring-2 focus:ring-cream-400 focus:ring-offset-2",
  {
    variants: {
      variant: {
        default: "border-transparent bg-cream-800 text-white hover:bg-cream-700",
        secondary: "border-transparent bg-cream-100 text-black hover:bg-cream-200",
        destructive: "border-transparent bg-cream-600 text-white hover:bg-cream-700",
        outline: "text-black border-cream-200",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  }
)

export interface BadgeProps
  extends React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof badgeVariants> {}

function Badge({ className, variant, ...props }: BadgeProps) {
  return (
    <div className={cn(badgeVariants({ variant }), className)} {...props} />
  )
}

export { Badge, badgeVariants }
