import Link from "next/link";
import type { ReactNode } from "react";
import { LockKeyhole, MessageSquareCode, UserRound, type LucideIcon } from "lucide-react";
import { AppShell } from "@/components/app-shell";
import type { User } from "@/lib/api";

type SettingsSection = "profile" | "account" | "dingtalk";

const settingsNav: { key: SettingsSection; label: string; href: string; icon: LucideIcon }[] = [
  { key: "profile", label: "个人资料", href: "/settings/profile", icon: UserRound },
  { key: "account", label: "账户安全", href: "/settings/account", icon: LockKeyhole },
  { key: "dingtalk", label: "钉钉绑定", href: "/settings/dingtalk", icon: MessageSquareCode },
];

export function SettingsShell({
  active,
  title,
  description,
  children,
  currentUser,
}: {
  active: SettingsSection;
  title: string;
  description: string;
  children: ReactNode;
  currentUser?: User | null;
}) {
  return (
    <AppShell currentUser={currentUser} mainClassName="mx-auto flex w-full max-w-6xl gap-0 px-0 py-0">
      <aside className="sticky top-16 hidden h-[calc(100vh-64px)] w-56 shrink-0 overflow-y-auto border-r bg-card xl:flex xl:flex-col">
        <div className="border-b p-5">
          <p className="text-xs font-bold uppercase tracking-widest text-muted-foreground">账户设置</p>
          <h1 className="mt-2 text-xl font-bold">我的设置</h1>
        </div>
        <nav className="space-y-1 p-3" aria-label="账户设置导航">
          {settingsNav.map((item) => (
            <SettingsLink active={active === item.key} href={item.href} icon={item.icon} key={item.key} label={item.label} />
          ))}
        </nav>
      </aside>
      <div className="min-w-0 flex-1 px-4 pt-6 pb-4 sm:px-6 xl:px-8">
        <nav className="mb-5 flex gap-2 overflow-x-auto rounded-lg border bg-white p-2 xl:hidden" aria-label="移动端账户设置导航">
          {settingsNav.map((item) => (
            <SettingsLink active={active === item.key} href={item.href} icon={item.icon} key={`mobile-${item.key}`} label={item.label} mobile />
          ))}
        </nav>
        <div className="mb-6">
          <p className="text-xs font-bold uppercase tracking-widest text-primary">账户设置</p>
          <h1 className="mt-2 text-2xl font-bold text-slate-900 sm:text-3xl">{title}</h1>
          <p className="mt-2 max-w-3xl text-sm leading-6 text-muted-foreground">{description}</p>
        </div>
        {children}
      </div>
    </AppShell>
  );
}

function SettingsLink({ active, href, icon: Icon, label, mobile = false }: { active: boolean; href: string; icon: LucideIcon; label: string; mobile?: boolean }) {
  return (
    <Link
      className={`${mobile ? "inline-flex h-9 shrink-0 whitespace-nowrap px-3" : "flex w-full px-3 py-2.5"} items-center gap-2 rounded-md text-sm font-medium transition-colors ${
        active ? "bg-primary/10 text-primary" : "text-slate-600 hover:bg-slate-50"
      }`}
      href={href}
      prefetch={false}
    >
      <Icon className="h-4 w-4" aria-hidden="true" />
      {label}
    </Link>
  );
}
