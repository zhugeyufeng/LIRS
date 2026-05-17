import Link from "next/link";
import {
  BarChart3,
  Bell,
  ClipboardCheck,
  GraduationCap,
  Megaphone,
  MonitorPlay,
  PackageSearch,
  Settings2,
  ShieldCheck,
  ThermometerSun,
  UsersRound,
  type LucideIcon,
} from "lucide-react";
import { AdminShell, requireAdmin } from "@/components/admin-shell";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";
import { canAccessAdminSection, type AdminSection } from "@/lib/permissions";

export default async function AdminPage() {
  const currentUser = await requireAdmin();
  const [dashboardResult, notificationsResult] = await Promise.allSettled([api.dashboard(), api.notifications()]);
  const dashboard = dashboardResult.status === "fulfilled" ? dashboardResult.value : emptyDashboard();
  const notifications = notificationsResult.status === "fulfilled" ? notificationsResult.value : [];
  const unreadNotifications = notifications.filter((item) => !item.read).length;
  const canManageNotifications = canAccessAdminSection(currentUser.role, "notifications", currentUser.financeEnabled);
  const canManageSettings = canAccessAdminSection(currentUser.role, "settings", currentUser.financeEnabled);
  const managementLinks = [
    { href: "/admin/users", icon: UsersRound, title: "用户管理", description: "用户审核、机构归属、角色分配和部门基础数据。", section: "users" },
    { href: "/admin/instruments", icon: ThermometerSun, title: "仪器资源后台", description: "仪器档案、状态、预约规则、可预约时段和门禁绑定。", section: "instruments" },
    { href: "/admin/materials", icon: PackageSearch, title: "资源管理后台", description: "维护一级目录、二级目录，并进入标准品、试剂和耗材独立管理页。", section: "materials" },
    { href: "/admin/training/questions", icon: GraduationCap, title: "培训与准入中心", description: "维护题库、实操考核、考试记录和仪器准入规则。", section: "trainingQuestions" },
    { href: "/admin/notifications", icon: Megaphone, title: "通知管理", description: "发布全局、部门或个人公告，并查看最近通知。", section: "notifications" },
    { href: "/admin/analytics", icon: BarChart3, title: "运营分析中心", description: "查看仪器、耗材、审批、财务和风险预警分析。", section: "analytics" },
    { href: "/operations", icon: MonitorPlay, title: "运营看板", description: "查看运行态指标、负载、审批效率和告警。", section: "operations" },
    { href: "/admin/security", icon: ShieldCheck, title: "安全审计与合规", description: "查看登录、操作、数据、权限和异常访问审计。", section: "security" },
    { href: "/admin/settings", icon: Settings2, title: "平台配置中心", description: "组织架构、租户配置、通知通道和第三方集成。", section: "settings" },
  ].filter((item) => canAccessAdminSection(currentUser.role, item.section as AdminSection, currentUser.financeEnabled));

  return (
    <AdminShell active="overview" title="工作概览" description="集中查看预约、审批、通知和基础配置；具体业务管理按中心拆分到独立页面。">
      <div className="mb-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <AdminMetric label="今日预约" value={dashboard.todayReservations} />
        <AdminMetric label="待审批" value={dashboard.pendingApprovals} />
        <AdminMetric label="使用中" value={dashboard.inUseReservations} />
        <AdminMetric label="已履约" value={dashboard.completedReservations} />
      </div>

      <div className="mb-8 grid gap-4 sm:grid-cols-2">
        <AdminMetric label="运营仪器" value={dashboard.activeInstruments} />
        <AdminMetric label="本月收入" value={`¥${dashboard.monthlyRevenue.toFixed(0)}`} />
      </div>

      <div className="mb-8 grid gap-4 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-6">
        {managementLinks.map((item) => (
          <ManagementLinkCard description={item.description} href={item.href} icon={item.icon} key={item.href} title={item.title} />
        ))}
      </div>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_360px]">
        <div className="min-w-0 space-y-6">
          <ReservationQueueCard />
          <AuditEventsCard />
        </div>
        <aside className="min-w-0 space-y-6">
          <NotificationSummaryCard canPublish={canManageNotifications} unread={unreadNotifications} total={notifications.length} />
          {canManageSettings ? <SettingsSummaryCard /> : null}
        </aside>
      </div>
    </AdminShell>
  );
}

async function ReservationQueueCard() {
  const reservations = await api.reservations().catch(() => []);
  const recentReservations = reservations.slice(0, 8);
  return (
    <Card className="overflow-hidden">
      <CardHeader className="border-b bg-slate-50/50">
        <CardTitle className="flex items-center gap-2">
          <ClipboardCheck className="h-5 w-5 text-primary" aria-hidden="true" />
          预约申请队列
        </CardTitle>
      </CardHeader>
      <CardContent className="p-0">
        <div className="space-y-3 p-4 xl:hidden">
          {recentReservations.map((item) => (
            <div className="rounded-lg border bg-white p-3 text-sm" key={item.id}>
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0">
                  <p className="truncate font-bold text-slate-900">{item.instrumentName}</p>
                  <p className="mt-1 text-xs text-slate-500">{item.userName}</p>
                </div>
                <span className="shrink-0 rounded-full bg-amber-50 px-2 py-0.5 text-xs font-bold text-amber-700">{statusLabel(item.status)}</span>
              </div>
              <p className="mt-2 text-xs text-muted-foreground">{formatDateTime(item.startTime)}</p>
            </div>
          ))}
        </div>
        <div className="hidden overflow-x-auto xl:block">
          <table className="w-full text-left text-sm">
            <thead className="bg-slate-50 text-slate-500">
              <tr>
                <th className="px-4 py-3">仪器</th>
                <th className="px-4 py-3">申请人</th>
                <th className="px-4 py-3">时间</th>
                <th className="px-4 py-3">状态</th>
              </tr>
            </thead>
            <tbody className="divide-y">
              {recentReservations.map((item) => (
                <tr className="hover:bg-slate-50" key={item.id}>
                  <td className="px-4 py-3 font-medium">{item.instrumentName}</td>
                  <td className="px-4 py-3">{item.userName}</td>
                  <td className="px-4 py-3 text-xs text-muted-foreground">{formatDateTime(item.startTime)}</td>
                  <td className="px-4 py-3">
                    <span className="rounded-full bg-amber-50 px-2 py-0.5 text-xs font-bold text-amber-700">{statusLabel(item.status)}</span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {recentReservations.length === 0 ? <p className="p-4 text-sm text-slate-500">暂无预约申请。</p> : null}
      </CardContent>
    </Card>
  );
}

async function AuditEventsCard() {
  const auditEvents = await api.auditEvents().catch(() => []);
  return (
    <Card>
      <CardHeader>
        <CardTitle>审计记录</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        {auditEvents.length === 0 ? <p className="text-sm text-slate-500">暂无审计记录。</p> : null}
        {auditEvents.slice(0, 12).map((event) => (
          <div className="flex flex-wrap items-center justify-between gap-3 rounded-lg border p-3 text-sm" key={event.id}>
            <div className="min-w-0">
              <p className="font-bold">{event.action}</p>
              <p className="mt-1 break-words text-xs text-slate-500">
                {event.actor} / {event.targetType} / {event.targetId}
              </p>
            </div>
            <span className="shrink-0 text-xs text-slate-500">{formatDateTime(event.createdAt)}</span>
          </div>
        ))}
      </CardContent>
    </Card>
  );
}

function SettingsSummaryCard() {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Settings2 className="h-5 w-5 text-primary" />
          平台配置中心
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <Link className="flex h-9 items-center justify-center rounded-md border px-3 text-sm font-medium text-slate-700 hover:bg-slate-50" href="/admin/settings/organization">
          组织架构管理
        </Link>
        <Link className="flex h-9 items-center justify-center rounded-md border px-3 text-sm font-medium text-slate-700 hover:bg-slate-50" href="/admin/settings/footer">
          Footer 自定义
        </Link>
        <Link className="flex h-9 items-center justify-center rounded-md border px-3 text-sm font-medium text-slate-700 hover:bg-slate-50" href="/admin/settings/copy">
          文案中心
        </Link>
      </CardContent>
    </Card>
  );
}

function NotificationSummaryCard({ canPublish, total, unread }: { canPublish: boolean; total: number; unread: number }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Bell className="h-5 w-5 text-primary" aria-hidden="true" />
          消息中心
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="grid grid-cols-2 gap-3">
          <AdminMetric label="全部通知" value={total} compact />
          <AdminMetric label="未读通知" value={unread} compact />
        </div>
        <Link className="inline-flex h-9 items-center justify-center rounded-md border px-3 text-sm font-medium text-slate-700 hover:bg-slate-50" href="/notifications">
          查看消息
        </Link>
        {canPublish ? (
          <Link className="inline-flex h-9 items-center justify-center rounded-md bg-primary px-3 text-sm font-medium text-white hover:bg-primary/90" href="/admin/notifications">
            发布公告
          </Link>
        ) : null}
      </CardContent>
    </Card>
  );
}

function ManagementLinkCard({ description, href, icon: Icon, title }: { description: string; href: string; icon: LucideIcon; title: string }) {
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

function AdminMetric({ compact = false, label, value }: { compact?: boolean; label: string; value: string | number }) {
  return (
    <div className={`rounded-lg border bg-white ${compact ? "p-3" : "p-4"}`}>
      <p className="text-sm text-slate-500">{label}</p>
      <p className={`mt-2 font-bold ${compact ? "text-xl" : "text-2xl"}`}>{value}</p>
    </div>
  );
}

function statusLabel(status: string) {
  const labels: Record<string, string> = {
    pending: "待审批",
    approved: "已通过",
    rejected: "已驳回",
    in_use: "使用中",
    completed: "已完成",
    cancelled: "已取消",
  };
  return labels[status] ?? status;
}

function formatDateTime(value: string) {
  return new Date(value).toLocaleString("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    timeZone: "Asia/Shanghai",
  });
}

function emptyDashboard() {
  return {
    todayReservations: 0,
    pendingApprovals: 0,
    inUseReservations: 0,
    completedReservations: 0,
    fulfillmentRate: 0,
    activeInstruments: 0,
    monthlyRevenue: 0,
  };
}
