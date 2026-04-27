import { useId } from "react";
import type { InputHTMLAttributes } from "react";

import { cn } from "@/shared/lib/cn";

type FloatingInputProps = InputHTMLAttributes<HTMLInputElement> & {
  label: string;
};

export function FloatingInput({ label, className, ...props }: FloatingInputProps) {
  const id = useId();

  return (
    <div className={cn("floating-input", className)}>
      <input id={id} placeholder=" " {...props} />
      <label htmlFor={id}>{label}</label>
    </div>
  );
}
