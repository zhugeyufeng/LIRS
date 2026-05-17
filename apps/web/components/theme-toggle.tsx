"use client";

import { Moon, Sun } from "lucide-react";
import { useEffect, useState } from "react";

export function ThemeToggle({ title = "主题" }: { title?: string }) {
  const [dark, setDark] = useState(false);
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    const saved = window.localStorage.getItem("lirs.theme");
    const initialDark = saved === "dark";
    document.documentElement.classList.toggle("dark", initialDark);
    setDark(initialDark);
    setMounted(true);
  }, []);

  function toggle() {
    const next = !dark;
    document.documentElement.classList.toggle("dark", next);
    window.localStorage.setItem("lirs.theme", next ? "dark" : "light");
    setDark(next);
  }

  return (
    <button
      aria-pressed={dark}
      className={`inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-primary ${mounted ? "" : "invisible"}`}
      onClick={toggle}
      title={title}
      type="button"
    >
      {dark ? <Sun className="h-4 w-4" aria-hidden="true" /> : <Moon className="h-4 w-4" aria-hidden="true" />}
    </button>
  );
}
