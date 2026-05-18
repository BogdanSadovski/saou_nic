import { useId } from "react";
import type { InputHTMLAttributes } from "react";

import { cn } from "@/shared/lib/cn";

type FloatingInputProps = InputHTMLAttributes<HTMLInputElement> & {
  label: string;
};

/**
 * Legacy shim. Renders a RealSync `.field` with stacked label above
 * an `.input` — replaces the prior floating-label design.
 */
export function FloatingInput({ label, className, id: idProp, ...props }: FloatingInputProps) {
  const generatedId = useId();
  const id = idProp ?? generatedId;

  return (
    <div className={cn("field", className)}>
      <label htmlFor={id}>{label}</label>
      <input id={id} className="input" {...props} />
    </div>
  );
}
