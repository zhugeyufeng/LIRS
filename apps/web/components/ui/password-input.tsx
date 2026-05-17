"use client";

import { InputHTMLAttributes, forwardRef, useState } from "react";
import { Eye, EyeOff } from "lucide-react";
import { cn } from "@/lib/utils";

export type PasswordInputProps = Omit<InputHTMLAttributes<HTMLInputElement>, "type">;

export const PasswordInput = forwardRef<HTMLInputElement, PasswordInputProps>(({ className, ...props }, ref) => {
  const [visible, setVisible] = useState(false);
  const label = visible ? "隐藏密码" : "显示密码";

  return (
    <span className="relative block">
      <input
        className={cn("h-10 w-full rounded-md border bg-white px-3 pr-11 text-sm outline-none focus:ring-2 focus:ring-primary", className)}
        ref={ref}
        type={visible ? "text" : "password"}
        {...props}
      />
      <button
        aria-label={label}
        className="absolute inset-y-0 right-0 inline-flex w-10 items-center justify-center rounded-r-md text-slate-500 hover:text-slate-900 focus:outline-none focus:ring-2 focus:ring-primary"
        onClick={() => setVisible((current) => !current)}
        type="button"
      >
        {visible ? <EyeOff className="h-4 w-4" aria-hidden="true" /> : <Eye className="h-4 w-4" aria-hidden="true" />}
        <span className="sr-only">{label}</span>
      </button>
    </span>
  );
});

PasswordInput.displayName = "PasswordInput";
