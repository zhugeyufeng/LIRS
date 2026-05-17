import Link from "next/link";
import type { ReactNode } from "react";
import { BarChart3, Bell, BookOpen, ClipboardCheck, GitBranch, GraduationCap, LayoutGrid, MonitorPlay, PackageSearch, Settings2, ShieldCheck, ThermometerSun, UsersRound, Wallet, Wrench } from "lucide-react";
import { redirect } from "next/navigation";
import { AdminSidebar, type AdminSidebarGroup } from "@/components/admin-sidebar";
import { AppShell } from "@/components/app-shell";
import { api, type User } from "@/lib/api";
import { canAccessAdminSection, isAnyAdminRole, type AdminSection } from "@/lib/permissions";

type AdminNavGroup = {
  label: string;
  items: {
    key: AdminSection;
    label: string;
    href: string;
    icon: AdminSidebarGroup["items"][number]["icon"];
    MobileIcon: typeof LayoutGrid;
  }[];
};

const adminNavGroups: AdminNavGroup[] = [
  {
    label: "运营中心",
    items: [
      { key: "overview", label: "工作概览", href: "/admin", icon: "layout", MobileIcon: LayoutGrid },
      { key: "analytics", label: "运营分析", href: "/admin/analytics", icon: "analytics", MobileIcon: BarChart3 },
      { key: "notifications", label: "通知管理", href: "/admin/notifications", icon: "bell", MobileIcon: Bell },
      { key: "operations", label: "运营看板", href: "/operations", icon: "monitor", MobileIcon: MonitorPlay },
    ],
  },
  {
    label: "实验资源中心",
    items: [
      { key: "instruments", label: "仪器管理", href: "/admin/instruments", icon: "thermometer", MobileIcon: ThermometerSun },
      { key: "materials", label: "资源管理", href: "/admin/materials", icon: "package", MobileIcon: PackageSearch },
      { key: "maintenance", label: "设备维护", href: "/maintenance", icon: "wrench", MobileIcon: Wrench },
    ],
  },
  {
    label: "培训与准入中心",
    items: [
      { key: "trainingQuestions", label: "题库管理", href: "/admin/training/questions", icon: "book", MobileIcon: BookOpen },
      { key: "trainingPractical", label: "线下考核", href: "/admin/training/practical", icon: "clipboard", MobileIcon: ClipboardCheck },
      { key: "trainingAuthorizations", label: "授权审批", href: "/admin/training/authorizations", icon: "shield", MobileIcon: ShieldCheck },
      { key: "trainingRules", label: "准入规则", href: "/admin/training/rules", icon: "graduation", MobileIcon: GraduationCap },
    ],
  },
  {
    label: "业务流程中心",
    items: [
      { key: "approvals", label: "审批中心", href: "/approvals", icon: "clipboard", MobileIcon: ClipboardCheck },
      { key: "workflows", label: "审批流配置", href: "/admin/workflows/templates", icon: "gitBranch", MobileIcon: GitBranch },
    ],
  },
  {
    label: "财务与计费中心",
    items: [
      { key: "finance", label: "财务管理", href: "/finance", icon: "wallet", MobileIcon: Wallet },
      { key: "finance", label: "仪器计费规则", href: "/admin/billing/instrument-rules", icon: "wallet", MobileIcon: Wallet },
      { key: "finance", label: "产品计费规则", href: "/admin/billing/material-rules", icon: "wallet", MobileIcon: Wallet },
    ],
  },
  {
    label: "组织与权限中心",
    items: [{ key: "users", label: "人员管理", href: "/admin/users", icon: "users", MobileIcon: UsersRound }],
  },
  {
    label: "安全审计与合规中心",
    items: [{ key: "security", label: "安全审计", href: "/admin/security", icon: "shield", MobileIcon: ShieldCheck }],
  },
  {
    label: "平台配置中心",
    items: [{ key: "settings", label: "平台配置", href: "/admin/settings", icon: "settings", MobileIcon: Settings2 }],
  },
];

export async function requireAdmin() {
  const currentUser = await api.me();
  if (!isAnyAdminRole(currentUser.role)) {
    redirect("/dashboard");
  }
  return currentUser;
}

export async function requireAdminSection(section: AdminSection) {
  const currentUser = await api.me();
  if (!canAccessAdminSection(currentUser.role, section, currentUser.financeEnabled)) {
    redirect("/dashboard");
  }
  return currentUser;
}

export async function AdminShell({
  active,
  title,
  description,
  children,
  allowedSections,
  currentUser,
}: {
  active: AdminSection;
  title: string;
  description: string;
  children: ReactNode;
  allowedSections?: AdminSection[];
  currentUser?: User | null;
}) {
  const resolvedCurrentUser = currentUser === undefined ? await api.currentUserOptional() : currentUser;
  const visibleGroups = adminNavGroups
    .map((group) => ({
      ...group,
      items: group.items.filter(
        (item) =>
          (!allowedSections || allowedSections.includes(item.key)) &&
          canAccessAdminSection(resolvedCurrentUser?.role, item.key, resolvedCurrentUser?.financeEnabled === true),
      ),
    }))
    .filter((group) => group.items.length > 0);

  return (
    <AppShell currentUser={resolvedCurrentUser} mainClassName="flex w-full gap-0 px-0 py-0">
      <AdminSidebar active={active} groups={visibleGroups.map((group) => ({ label: group.label, items: group.items.map(({ MobileIcon, ...item }) => item) }))} />

      <div className="min-w-0 flex-1 px-4 pt-6 pb-4 sm:px-6 xl:px-8">
        <nav className="mb-5 space-y-3 rounded-lg border bg-white p-2 xl:hidden" aria-label="移动端管理中心导航">
          {visibleGroups.map((group) => (
            <div key={`mobile-${group.label}`}>
              <p className="mb-2 px-2 text-[10px] font-bold uppercase tracking-widest text-slate-400">{group.label}</p>
              <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-4">
                {group.items.map((item) => (
                  <Link
                    className={`inline-flex h-10 min-w-0 items-center justify-center gap-2 rounded-md px-2 text-sm font-medium ${
                      active === item.key ? "bg-primary text-white" : "text-slate-600 hover:bg-slate-50"
                    }`}
                    href={item.href}
                    key={`mobile-${item.href}`}
                    prefetch={false}
                  >
                    <item.MobileIcon className="h-4 w-4 shrink-0" aria-hidden="true" />
                    <span className="min-w-0 truncate">{item.label}</span>
                  </Link>
                ))}
              </div>
            </div>
          ))}
        </nav>
        <div className="mx-auto max-w-[88rem]">
          <div className="mb-6">
            <p className="text-xs font-bold uppercase tracking-widest text-primary">管理中心</p>
            <h1 className="mt-2 text-2xl font-bold text-slate-900 sm:text-3xl">{title}</h1>
            <p className="mt-2 max-w-3xl text-sm leading-6 text-muted-foreground">{description}</p>
          </div>
          {children}
        </div>
      </div>
    </AppShell>
  );
}
