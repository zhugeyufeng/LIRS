"use client";

import Link from "next/link";
import {
  BarChart3,
  Bell,
  BookOpen,
  ClipboardCheck,
  GitBranch,
  GraduationCap,
  LayoutGrid,
  MonitorPlay,
  PackageSearch,
  PanelLeftClose,
  PanelLeftOpen,
  Settings2,
  ShieldCheck,
  ThermometerSun,
  UsersRound,
  Wallet,
  Wrench,
  type LucideIcon,
} from "lucide-react";
import { useState } from "react";
import type { AdminSection } from "@/lib/permissions";

export type AdminSidebarIcon =
  | "analytics"
  | "bell"
  | "book"
  | "clipboard"
  | "gitBranch"
  | "graduation"
  | "layout"
  | "monitor"
  | "package"
  | "settings"
  | "shield"
  | "thermometer"
  | "users"
  | "wallet"
  | "wrench";

export type AdminSidebarGroup = {
  label: string;
  items: {
    key: AdminSection;
    label: string;
    href: string;
    icon: AdminSidebarIcon;
  }[];
};

const sidebarIcons: Record<AdminSidebarIcon, LucideIcon> = {
  analytics: BarChart3,
  bell: Bell,
  book: BookOpen,
  clipboard: ClipboardCheck,
  gitBranch: GitBranch,
  graduation: GraduationCap,
  layout: LayoutGrid,
  monitor: MonitorPlay,
  package: PackageSearch,
  settings: Settings2,
  shield: ShieldCheck,
  thermometer: ThermometerSun,
  users: UsersRound,
  wallet: Wallet,
  wrench: Wrench,
};

export function AdminSidebar({ active, groups }: { active: AdminSection; groups: AdminSidebarGroup[] }) {
  const [collapsed, setCollapsed] = useState(false);
  const widthClass = collapsed ? "w-16" : "w-64";
  const ToggleIcon = collapsed ? PanelLeftOpen : PanelLeftClose;

  return (
    <aside className={`sticky top-16 hidden h-[calc(100vh-64px)] ${widthClass} shrink-0 overflow-y-auto border-r bg-card transition-[width] duration-200 xl:flex xl:flex-col`}>
      <div className={`border-b ${collapsed ? "p-3" : "p-6"}`}>
        <div className={collapsed ? "flex justify-center" : "flex items-start justify-between gap-3"}>
          <div className={collapsed ? "sr-only" : "min-w-0"}>
            <h2 className="text-xs font-bold uppercase tracking-widest text-muted-foreground">管理中心</h2>
            <p className="mt-2 text-xl font-bold">实验室运营管理</p>
          </div>
          <button
            aria-label={collapsed ? "展开管理中心菜单" : "收缩管理中心菜单"}
            aria-pressed={collapsed}
            className="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-md border text-slate-600 transition-colors hover:bg-slate-50 hover:text-primary"
            onClick={() => setCollapsed((value) => !value)}
            title={collapsed ? "展开菜单" : "收缩菜单"}
            type="button"
          >
            <ToggleIcon className="h-4 w-4" aria-hidden="true" />
          </button>
        </div>
      </div>
      <nav className={collapsed ? "flex-1 space-y-3 p-2" : "flex-1 space-y-5 p-4"} aria-label="管理中心导航">
        {groups.map((group) => (
          <div key={group.label}>
            <p className={collapsed ? "sr-only" : "mb-2 px-4 text-[10px] font-bold uppercase tracking-widest text-slate-400"}>{group.label}</p>
            <div className="space-y-1">
              {group.items.map((item) => (
                <AdminSidebarLink active={active === item.key} collapsed={collapsed} href={item.href} icon={item.icon} key={item.href} label={item.label} />
              ))}
            </div>
          </div>
        ))}
      </nav>
    </aside>
  );
}

function AdminSidebarLink({ active, collapsed, href, icon, label }: { active: boolean; collapsed: boolean; href: string; icon: AdminSidebarIcon; label: string }) {
  const Icon = sidebarIcons[icon];

  return (
    <Link
      aria-label={collapsed ? label : undefined}
      className={`${collapsed ? "h-10 justify-center px-0" : "px-4 py-3"} flex w-full items-center gap-3 rounded-md text-sm transition-colors ${
        active ? "bg-primary/5 font-bold text-primary" : "text-slate-600 hover:bg-slate-50"
      }`}
      href={href}
      prefetch={false}
      title={collapsed ? label : undefined}
    >
      <Icon className="h-4 w-4 shrink-0" aria-hidden="true" />
      {collapsed ? null : <span className="min-w-0 truncate">{label}</span>}
    </Link>
  );
}
