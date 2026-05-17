"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { ChevronDown, LogOut, Settings, UserRound } from "lucide-react";
import { browserLogout, browserMe, copyText, type CopySettings, type User } from "@/lib/api";
import { roleLabel } from "@/lib/permissions";

export function AccountMenu({ copySettings = null, initialUser = null }: { copySettings?: CopySettings | null; initialUser?: User | null }) {
  const [user, setUser] = useState<User | null>(initialUser);
  const t = (key: string, fallback = key) => copyText(copySettings, key, fallback);

  useEffect(() => {
    if (initialUser) {
      setUser(initialUser);
      return;
    }
    browserMe()
      .then(setUser)
      .catch(() => setUser(null));
  }, [initialUser]);

  async function logout() {
    await browserLogout();
    setUser(null);
    window.location.href = "/login";
  }

  if (!user) {
    return (
      <div className="flex shrink-0 items-center justify-end gap-1 sm:gap-2">
        <Link className="inline-flex h-9 items-center justify-center whitespace-nowrap rounded-md px-2 text-xs font-medium text-muted-foreground hover:bg-accent hover:text-primary sm:min-w-16 sm:px-3 sm:text-sm" href="/login">
          {t("登录")}
        </Link>
        <Link className="inline-flex h-9 items-center justify-center whitespace-nowrap rounded-md bg-primary px-2 text-xs font-medium text-primary-foreground sm:min-w-16 sm:px-3 sm:text-sm" href="/register">
          {t("注册")}
        </Link>
      </div>
    );
  }

  return (
    <div className="group relative shrink-0">
      <button className="inline-flex h-9 max-w-48 items-center justify-end gap-2 whitespace-nowrap rounded-full py-1 pl-1 pr-2 transition-colors hover:bg-accent" type="button">
        <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full border border-primary/20 bg-primary/10 text-xs font-bold text-primary">
          {user.name.slice(0, 1)}
        </span>
        <span className="hidden min-w-0 text-left xl:block">
          <span className="block text-xs font-bold leading-none">{user.name}</span>
          <span className="mt-0.5 block overflow-hidden text-ellipsis text-[10px] text-muted-foreground">{roleLabel(user.role)} · {user.department}</span>
        </span>
        <ChevronDown className="h-3.5 w-3.5 shrink-0 text-muted-foreground" aria-hidden="true" />
      </button>
      <div className="absolute right-0 top-full hidden w-56 max-w-[calc(100vw-2rem)] pt-2 group-hover:block group-focus-within:block">
        <div className="rounded-md border bg-white p-1 shadow-md">
          <div className="mb-1 border-b px-2 py-1.5">
            <p className="text-[10px] font-bold uppercase text-muted-foreground">{t("当前账号")}</p>
            <p className="text-xs font-medium">{user.email}</p>
          </div>
          <Link className="flex h-8 items-center gap-2 rounded-sm px-2 text-sm hover:bg-accent" href="/settings/profile">
            <UserRound className="h-4 w-4" aria-hidden="true" />
            {t("个人信息")}
          </Link>
          <Link className="flex h-8 items-center gap-2 rounded-sm px-2 text-sm hover:bg-accent" href="/settings/account">
            <Settings className="h-4 w-4" aria-hidden="true" />
            {t("账户设置")}
          </Link>
          <button className="flex h-8 w-full items-center gap-2 rounded-sm px-2 text-left text-sm hover:bg-accent" onClick={logout} type="button">
            <LogOut className="h-4 w-4" aria-hidden="true" />
            {t("退出登录")}
          </button>
        </div>
      </div>
    </div>
  );
}
