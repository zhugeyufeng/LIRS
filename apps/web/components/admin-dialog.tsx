"use client";

import { ReactElement, ReactNode, cloneElement, useEffect, useId, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { X } from "lucide-react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";

type TriggerElement = ReactElement<{
  onClick?: () => void;
  type?: "button" | "submit" | "reset";
}>;

export function AdminDialog({
  children,
  description,
  maxWidth = "max-w-3xl",
  title,
  trigger,
}: {
  children: ReactNode | ((close: () => void) => ReactNode);
  description?: string;
  maxWidth?: string;
  title: string;
  trigger: TriggerElement;
}) {
  const [open, setOpen] = useState(false);
  const [mounted, setMounted] = useState(false);
  const titleId = useId();
  const descriptionId = useId();
  const dialogRef = useRef<HTMLElement>(null);
  const previousFocusRef = useRef<HTMLElement | null>(null);

  useEffect(() => {
    setMounted(true);
  }, []);

  useEffect(() => {
    if (!open) {
      return;
    }
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    function onKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape") {
        setOpen(false);
        return;
      }
      if (event.key !== "Tab") {
        return;
      }
      const focusable = focusableElements(dialogRef.current);
      if (focusable.length === 0) {
        event.preventDefault();
        return;
      }
      const first = focusable[0];
      const last = focusable[focusable.length - 1];
      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault();
        last.focus();
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault();
        first.focus();
      }
    }
    window.setTimeout(() => {
      focusableElements(dialogRef.current)[0]?.focus();
    }, 0);
    document.addEventListener("keydown", onKeyDown);
    return () => {
      document.body.style.overflow = previousOverflow;
      document.removeEventListener("keydown", onKeyDown);
      previousFocusRef.current?.focus();
    };
  }, [open]);

  const triggerNode = cloneElement(trigger, {
    onClick: () => {
      trigger.props.onClick?.();
      previousFocusRef.current = document.activeElement instanceof HTMLElement ? document.activeElement : null;
      setOpen(true);
    },
    type: trigger.props.type ?? "button",
  });

  const dialog = open ? (
    <div className="fixed inset-0 z-[100] flex items-end justify-center bg-slate-950/55 p-0 sm:items-center sm:p-4" role="presentation">
      <button className="absolute inset-0 h-full w-full cursor-default" onClick={() => setOpen(false)} type="button" aria-label="关闭弹窗" />
      <section
        aria-describedby={description ? descriptionId : undefined}
        aria-labelledby={titleId}
        aria-modal="true"
        className={cn(
          "relative max-h-[100dvh] w-full min-w-0 overflow-y-auto rounded-t-lg border bg-background shadow-2xl sm:max-h-[calc(100dvh-2rem)] sm:w-[calc(100vw-2rem)] sm:rounded-lg",
          maxWidth,
        )}
        ref={dialogRef}
        role="dialog"
      >
        <div className="sticky top-0 z-10 flex items-start justify-between gap-4 border-b bg-background/95 p-4 backdrop-blur sm:p-5">
          <div className="min-w-0 flex-1">
            <h2 className="break-words text-base font-bold text-foreground sm:text-lg" id={titleId}>
              {title}
            </h2>
            {description ? (
              <p className="mt-1 break-words text-sm leading-6 text-muted-foreground" id={descriptionId}>
                {description}
              </p>
            ) : null}
          </div>
          <Button className="h-9 w-9 shrink-0" onClick={() => setOpen(false)} size="icon" type="button" variant="ghost">
            <X className="h-4 w-4" aria-hidden="true" />
            <span className="sr-only">关闭</span>
          </Button>
        </div>
        <div className="p-4 sm:p-5">{typeof children === "function" ? children(() => setOpen(false)) : children}</div>
      </section>
    </div>
  ) : null;

  return (
    <>
      {triggerNode}
      {mounted && dialog ? createPortal(dialog, document.body) : null}
    </>
  );
}

function focusableElements(root: HTMLElement | null) {
  if (!root) {
    return [];
  }
  return Array.from(
    root.querySelectorAll<HTMLElement>(
      'a[href], button:not([disabled]), textarea:not([disabled]), input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])',
    ),
  ).filter((element) => !element.hasAttribute("disabled") && element.getAttribute("aria-hidden") !== "true");
}
