import Link from "next/link";
import type { ReactNode } from "react";
import {
  Beaker,
  Bell,
  ChevronDown,
  Search,
} from "lucide-react";
import { AccountMenu } from "@/components/account-menu";
import { ThemeToggle } from "@/components/theme-toggle";
import { api, copyText, createDefaultCopySettings, createDefaultFooterSettings, type User } from "@/lib/api";
import { canAccessAdminSection, isAnyAdminRole, type AdminSection } from "@/lib/permissions";

type PrimaryNavItem = {
  href: string;
  label: string;
  requiresAuth?: boolean;
  requiresFinance?: boolean;
  adminSection?: AdminSection;
};

type PrimaryNavGroup = {
  label: string;
  href?: string;
  requiresAuth?: boolean;
  requiresFinance?: boolean;
  items?: PrimaryNavItem[];
};

type AppShellProps = {
  children: ReactNode;
  mainClassName?: string;
  currentUser?: User | null;
};

const primaryNavGroups: PrimaryNavGroup[] = [
  { href: "/", label: "首页" },
  { href: "/instruments", label: "仪器预约" },
  {
    label: "资源",
    items: [
      { href: "/materials/standards", label: "标准品/标准物质目录" },
      { href: "/materials/reagents", label: "试剂目录" },
      { href: "/materials/consumables", label: "耗材目录" },
      { href: "/materials/requests", label: "资源申领", requiresAuth: true },
      { href: "/materials/purchases", label: "资源申购", requiresAuth: true },
      { href: "/spaces", label: "空间资源", requiresAuth: true },
      { href: "/samples", label: "样本管理", requiresAuth: true },
    ],
  },
  {
    label: "业务",
    requiresAuth: true,
    items: [
      { href: "/dashboard", label: "个人工作台", requiresAuth: true },
      { href: "/reservations", label: "预约记录", requiresAuth: true },
      { href: "/approvals", label: "审批中心", adminSection: "approvals" },
      { href: "/notifications", label: "通知中心", requiresAuth: true },
      { href: "/finance", label: "财务管理", requiresAuth: true, requiresFinance: true },
    ],
  },
  {
    label: "培训",
    requiresAuth: true,
    items: [
      { href: "/training", label: "培训总览", requiresAuth: true },
      { href: "/training/courses", label: "课程管理", requiresAuth: true },
      { href: "/training/authorizations", label: "授权记录", requiresAuth: true },
      { href: "/training/exams", label: "在线考试", requiresAuth: true },
    ],
  },
  {
    label: "更多",
    requiresAuth: true,
    items: [
      { href: "/lims/tasks", label: "LIMS 任务", requiresAuth: true },
      { href: "/eln/records", label: "ELN 记录", requiresAuth: true },
      { href: "/iot/devices", label: "IoT 设备", requiresAuth: true },
      { href: "/ai-assistant", label: "AI 助手", requiresAuth: true },
      { href: "/data-center", label: "数据中台", requiresAuth: true },
    ],
  },
];

export async function AppShell({
  children,
  mainClassName = "mx-auto w-full max-w-7xl px-4 pt-6 pb-4 sm:px-6 sm:pt-8 sm:pb-4 lg:px-8",
  currentUser: providedCurrentUser,
}: AppShellProps) {
  const [currentUser, footerSettings, copySettings] = await Promise.all([
    providedCurrentUser === undefined ? api.currentUserOptional() : Promise.resolve(providedCurrentUser),
    api.cachedFooterSettings().catch(() => createDefaultFooterSettings()),
    api.cachedCopySettings().catch(() => createDefaultCopySettings()),
  ]);
  const t = (key: string, fallback = key) => copyText(copySettings, key, fallback);
  const unreadNotificationCount = 0;
  const showAdminMenu = isAnyAdminRole(currentUser?.role);
  const showFinance = currentUser?.role === "super_admin" || currentUser?.financeEnabled === true;
  const visiblePrimaryNavGroups = primaryNavGroups
    .map((group) => {
      const items = group.items?.filter((item) => {
        if (item.requiresAuth && !currentUser) {
          return false;
        }
        if (item.requiresFinance && !showFinance) {
          return false;
        }
        if (item.adminSection) {
          return canAccessAdminSection(currentUser?.role, item.adminSection, currentUser?.financeEnabled === true);
        }
        return true;
      });
      const groupVisible = (!group.requiresAuth || currentUser) && (!group.requiresFinance || showFinance);
      if (group.items) {
        return groupVisible && items && items.length > 0 ? { ...group, items } : null;
      }
      return groupVisible ? group : null;
    })
    .filter((group): group is PrimaryNavGroup => group !== null);
  const mobilePrimaryNav = visiblePrimaryNavGroups
    .map((group) => (group.href ? { href: group.href, label: group.label } : group.items?.[0] ? { href: group.items[0].href, label: group.label } : null))
    .filter((item): item is PrimaryNavItem => item !== null);
  const quickCategories: string[] = [];

  return (
    <div className="flex min-h-screen flex-col bg-background text-foreground">
      <header className="sticky top-0 z-50 border-b bg-background/95 backdrop-blur">
        <div className="mx-auto flex h-16 w-full max-w-[88rem] items-center gap-2 px-4 sm:gap-3 sm:px-6 xl:gap-4 lg:px-8">
          <Link className="group flex shrink-0 items-center gap-2" href="/">
            <span className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary text-primary-foreground shadow-lg shadow-primary/20 transition-transform group-hover:rotate-12">
              <Beaker className="h-5 w-5" aria-hidden="true" />
            </span>
            <span className="hidden flex-col sm:flex">
              <span className="text-lg font-bold leading-none tracking-tight">{t("实验室运营系统")}</span>
              <span className="mt-1 hidden font-mono text-[10px] uppercase tracking-widest text-muted-foreground xl:block">LIRS 2026 VERSION</span>
            </span>
          </Link>
          <nav className="hidden min-w-0 flex-1 items-center justify-start gap-1 xl:flex xl:justify-center" aria-label={t("主导航")}>
            {visiblePrimaryNavGroups.map((group) =>
              group.items?.length ? (
                <div className="group relative" key={group.label}>
                  <button className="inline-flex h-9 items-center justify-center gap-1 whitespace-nowrap rounded-md px-3 text-sm font-medium leading-none text-muted-foreground transition-colors hover:bg-accent hover:text-primary" type="button">
                    {t(group.label)}
                    <ChevronDown className="h-3.5 w-3.5" aria-hidden="true" />
                  </button>
                  <div className="absolute left-0 top-full hidden w-52 pt-2 group-hover:block group-focus-within:block">
                    <div className="space-y-1 rounded-md border bg-white p-2 text-sm shadow-md">
                      {group.items.map((item) => (
                        <Link
                          className="flex h-9 items-center whitespace-nowrap rounded-sm px-3 text-slate-700 transition-colors hover:bg-accent hover:text-primary"
                          href={item.href}
                          key={`${item.href}-${item.label}`}
                          prefetch={false}
                        >
                          {t(item.label)}
                        </Link>
                      ))}
                    </div>
                  </div>
                </div>
              ) : (
                <Link
                  className="inline-flex h-9 items-center justify-center whitespace-nowrap rounded-md px-3 text-sm font-medium leading-none text-muted-foreground transition-colors hover:bg-accent hover:text-primary"
                  href={group.href ?? "/"}
                  key={`${group.href}-${group.label}`}
                  prefetch={false}
                >
                  {t(group.label)}
                </Link>
              ),
            )}
            {showAdminMenu ? (
              <Link
                className="inline-flex h-9 items-center justify-center whitespace-nowrap rounded-md px-2 text-sm font-medium leading-none text-muted-foreground transition-colors hover:bg-accent hover:text-primary"
                href="/admin"
                prefetch={false}
              >
                {t("管理中心")}
              </Link>
            ) : null}
          </nav>
          <div className="ml-auto flex shrink-0 items-center justify-end gap-2 sm:gap-3">
            <form action="/instruments" className="hidden items-center gap-0 2xl:flex">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" aria-hidden="true" />
                <input
                  className="h-9 w-40 rounded-l-md border-0 bg-muted/60 pl-9 pr-3 text-sm outline-none ring-1 ring-transparent transition focus:ring-primary"
                  name="search"
                  placeholder={t("快速查找仪器...")}
                  type="search"
                />
              </div>
              <select
                className="h-9 rounded-r-md border-0 bg-primary/10 px-3 text-xs font-bold text-primary outline-none ring-1 ring-primary/10 transition focus:ring-primary"
                name="category"
                title={t("仪器分类")}
              >
                <option value="">{t("全部分类")}</option>
                {quickCategories.map((category) => (
                  <option key={category} value={category}>
                    {category}
                  </option>
                ))}
              </select>
            </form>
            <div className="flex shrink-0 items-center justify-end gap-1 border-l pl-2 sm:gap-2 sm:pl-4">
              <ThemeToggle title={t("主题")} />
              {currentUser ? (
                <Link className="relative flex h-9 w-9 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-primary" href="/notifications" title={t("通知")}>
                  <Bell className="h-4 w-4" aria-hidden="true" />
                  {unreadNotificationCount > 0 ? <span className="absolute right-2 top-2 h-2 w-2 rounded-full bg-destructive" /> : null}
                </Link>
              ) : null}
              <AccountMenu copySettings={copySettings} initialUser={currentUser} />
            </div>
          </div>
        </div>
        <nav className="border-t xl:hidden" aria-label={t("移动端主导航")}>
          <div className="mx-auto flex max-w-[88rem] gap-2 overflow-x-auto px-4 py-2 sm:px-6">
            {mobilePrimaryNav.map((item) => (
              <Link
                className="inline-flex h-8 shrink-0 items-center justify-center whitespace-nowrap rounded-md px-3 text-sm font-medium leading-none text-muted-foreground transition-colors hover:bg-accent hover:text-primary"
                href={item.href}
                key={`mobile-${item.href}-${item.label}`}
                prefetch={false}
              >
                {t(item.label)}
              </Link>
            ))}
            {showAdminMenu ? (
              <Link
                className="inline-flex h-8 shrink-0 items-center justify-center whitespace-nowrap rounded-md px-3 text-sm font-medium leading-none text-muted-foreground transition-colors hover:bg-accent hover:text-primary"
                href="/admin"
                prefetch={false}
              >
                {t("管理后台")}
              </Link>
            ) : null}
          </div>
        </nav>
      </header>
      <main className={mainClassName}>{children}</main>
      <footer className="border-t bg-slate-950 text-slate-200">
        <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
          <div className="grid gap-8 xl:grid-cols-[1.2fr_1fr]">
            <div className="space-y-4">
              <div className="flex items-center gap-3">
                <span className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary text-primary-foreground shadow-lg shadow-primary/20">
                  <Beaker className="h-5 w-5" aria-hidden="true" />
                </span>
                <div className="min-w-0">
                  <p className="text-base font-semibold text-white">{footerSettings.brandName}</p>
                  <p className="mt-1 text-sm text-slate-400">{footerSettings.brandTagline}</p>
                </div>
              </div>
              <p className="max-w-xl text-sm leading-6 text-slate-400">{footerSettings.description}</p>
            </div>
            <div className="grid gap-6 sm:grid-cols-2 xl:grid-cols-3">
              {footerSettings.sections.map((section, sectionIndex) => (
                <section key={`${section.title}-${sectionIndex}`} className="space-y-3">
                  <h3 className="text-sm font-semibold text-white">{section.title}</h3>
                  <div className="space-y-2 text-sm leading-6 text-slate-400">
                    {section.lines.map((line, lineIndex) => (
                      <p className="break-words" key={`${section.title}-${sectionIndex}-${lineIndex}`}>
                        {line}
                      </p>
                    ))}
                  </div>
                </section>
              ))}
            </div>
          </div>
          <div className="mt-8 border-t border-white/10 pt-4 text-xs text-slate-500">
            <p>{footerSettings.copyright}</p>
          </div>
        </div>
      </footer>
    </div>
  );
}
