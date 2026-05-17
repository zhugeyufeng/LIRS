import Link from "next/link";
import { BellDot, Building2, Landmark, MessageSquareCode, PanelBottom, Settings2, ShieldCheck, Type, Wallet, type LucideIcon } from "lucide-react";

type AdminSettingsSection = "overview" | "organization" | "tenants" | "billing" | "notifications" | "dingtalk" | "access-control" | "footer" | "copy";

const items: { key: AdminSettingsSection; label: string; href: string; icon: LucideIcon }[] = [
  { key: "overview", label: "平台配置总览", href: "/admin/settings", icon: Settings2 },
  { key: "organization", label: "组织架构管理", href: "/admin/settings/organization", icon: Building2 },
  { key: "tenants", label: "租户配置", href: "/admin/settings/tenants", icon: Landmark },
  { key: "billing", label: "财务模块开关", href: "/admin/settings/billing", icon: Wallet },
  { key: "notifications", label: "通知通道配置", href: "/admin/settings/notifications", icon: BellDot },
  { key: "dingtalk", label: "钉钉应用设置", href: "/admin/settings/dingtalk", icon: MessageSquareCode },
  { key: "access-control", label: "第三方集成", href: "/admin/settings/access-control", icon: ShieldCheck },
  { key: "copy", label: "文案中心", href: "/admin/settings/copy", icon: Type },
  { key: "footer", label: "系统基础配置", href: "/admin/settings/footer", icon: PanelBottom },
];

export function AdminSettingsNav({ active, role }: { active: AdminSettingsSection; role?: string }) {
  const visibleItems = items.filter((item) => (item.key !== "notifications" && item.key !== "access-control") || role === undefined || role === "super_admin");
  return (
    <nav className="mb-6 grid grid-cols-1 gap-2 rounded-lg border bg-white p-2 sm:grid-cols-2 md:grid-cols-3 xl:grid-cols-6" aria-label="平台配置层级导航">
      {visibleItems.map((item) => (
        <Link
          className={`inline-flex h-10 min-w-0 items-center justify-center gap-2 rounded-md px-3 text-sm font-medium ${
            active === item.key ? "bg-primary text-white" : "text-slate-600 hover:bg-slate-50"
          }`}
          href={item.href}
          key={item.key}
          prefetch={false}
        >
          <item.icon className="h-4 w-4 shrink-0" aria-hidden="true" />
          <span className="min-w-0 truncate">{item.label}</span>
        </Link>
      ))}
    </nav>
  );
}

export function AdminSettingsCard({ description, href, icon: Icon, title }: { description: string; href: string; icon: LucideIcon; title: string }) {
  return (
    <Link className="rounded-lg border bg-white p-4 transition-colors hover:border-primary/40 hover:bg-primary/5" href={href} prefetch={false}>
      <div className="flex items-center gap-3">
        <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-md bg-primary/10 text-primary">
          <Icon className="h-5 w-5" aria-hidden="true" />
        </span>
        <h2 className="font-bold text-slate-900">{title}</h2>
      </div>
      <p className="mt-3 text-sm leading-6 text-slate-500">{description}</p>
    </Link>
  );
}
