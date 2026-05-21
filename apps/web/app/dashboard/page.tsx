import Link from "next/link";
import { redirect } from "next/navigation";
import {
  Bell,
  CalendarCheck2,
  CheckCircle2,
  Clock3,
  LayoutDashboard,
  Package,
  ShoppingCart,
  QrCode,
  Settings2,
  ShieldCheck,
  UsersRound,
  Wallet,
  type LucideIcon,
} from "lucide-react";
import { AppShell } from "@/components/app-shell";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api, type Reservation } from "@/lib/api";
import { reservationStatusLabel } from "@/lib/status-labels";

export default async function DashboardPage({
  searchParams,
}: {
  searchParams?: Promise<{ reservations?: string }>;
}) {
  const params = (await searchParams) ?? {};
  const reservationView = params.reservations === "completed" || params.reservations === "pending" ? params.reservations : "active";
  const currentUser = await api.me();
  const [reservationsResult, notificationsResult, ledgerResult] = await Promise.allSettled([
    api.reservations(),
    api.notifications(undefined, "announcement"),
    currentUser.role === "super_admin" || currentUser.financeEnabled ? api.ledger() : Promise.resolve([]),
  ]);
  const reservations = reservationsResult.status === "fulfilled" ? reservationsResult.value : [];
  const notifications = notificationsResult.status === "fulfilled" ? notificationsResult.value : [];
  const showFinance = currentUser.role === "super_admin" || currentUser.financeEnabled;
  const ledger = ledgerResult.status === "fulfilled" ? ledgerResult.value : [];
  const activeReservations = reservations.filter((item) => !["completed", "rejected", "cancelled"].includes(item.status));
  const visibleReservations = reservations.filter((item) => {
    if (reservationView === "completed") {
      return item.status === "completed";
    }
    if (reservationView === "pending") {
      return item.status === "pending";
    }
    return !["completed", "rejected", "cancelled"].includes(item.status);
  });
  const activeAuthorization = reservations.find((item) => item.status === "approved" || item.status === "in_use");
  const globalAnnouncement = notifications.find((item) => item.targetScope === "global");
  const totalSpend = ledger.reduce((sum, item) => sum + (item.amount ?? 0), 0);
  const now = new Date();
  const monthlyReservations = reservations.filter((item) => {
    const start = new Date(item.startTime);
    return start.getFullYear() === now.getFullYear() && start.getMonth() === now.getMonth();
  });

  return (
    <AppShell currentUser={currentUser}>
      <div className="grid gap-8 xl:grid-cols-[280px_minmax(0,1fr)]">
        <aside className="space-y-4">
          <Card className="p-2">
            {[
              { label: "控制面板", icon: LayoutDashboard, href: "/dashboard#overview", active: true },
              { label: "个人资料", icon: UsersRound, href: "/settings/profile" },
              { label: "账户安全", icon: Settings2, href: "/settings/account" },
              { label: "预约记录", icon: CalendarCheck2, href: "/reservations" },
              ...(showFinance ? [{ label: "财务管理", icon: Wallet }] : []),
              { label: "申领管理", icon: Package, href: "/materials/requests" },
              { label: "资源申购", icon: ShoppingCart, href: "/materials/purchases" },
              { label: "我的通知", icon: Bell },
            ].map((item) => (
              <Link
                className={`flex w-full items-center gap-3 rounded-md px-4 py-2.5 text-sm font-medium transition-all ${
                  item.active ? "bg-primary/10 text-primary" : "text-muted-foreground hover:bg-slate-50"
                }`}
                href={item.href ?? (item.label === "财务管理" ? "/finance" : item.label === "申领管理" ? "/materials/requests" : item.label === "我的通知" ? "/notifications" : "/dashboard")}
                key={item.label}
              >
                <item.icon className="h-4 w-4" aria-hidden="true" />
                {item.label}
              </Link>
            ))}
          </Card>
          <div className="relative overflow-hidden rounded-xl bg-primary p-6 text-white shadow-lg">
            <ShieldCheck className="absolute -bottom-4 -right-4 h-24 w-24 rotate-12 text-white/10" aria-hidden="true" />
            <h4 className="mb-4 text-sm font-bold uppercase tracking-wider text-white/80">当前有效授权</h4>
            <div className="flex items-end justify-between">
              <div>
                <p className="font-mono text-2xl font-bold">{activeAuthorization ? activeAuthorization.id.slice(0, 8).toUpperCase() : "暂无"}</p>
                <p className="mt-1 text-[10px] uppercase text-white/70">
                  {activeAuthorization ? `有效至: ${formatDateTime(activeAuthorization.endTime)}` : "无可用中的预约授权"}
                </p>
              </div>
              <QrCode className="h-8 w-8" aria-hidden="true" />
            </div>
          </div>
        </aside>

        <div className="space-y-8" id="overview">
          <div className={`grid grid-cols-1 gap-4 ${showFinance ? "sm:grid-cols-3" : "sm:grid-cols-2"}`}>
            <Metric label="本月预约总数" value={`${monthlyReservations.length} 次`} note="按预约开始时间统计" />
            {showFinance ? <Metric label="已入账费用" value={`¥${totalSpend.toFixed(2)}`} note="个人费用流水" /> : null}
            <Metric label="待处理事项" value={`${activeReservations.length} 项`} note="审批/使用中预约" />
          </div>

          {globalAnnouncement ? (
            <div className="flex flex-col gap-3 rounded-xl border border-amber-200 bg-amber-50 p-4 text-amber-800 sm:flex-row sm:items-center sm:gap-4">
              <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-amber-100">
                <Clock3 className="h-5 w-5" aria-hidden="true" />
              </div>
              <div className="min-w-0 flex-1">
                <dl className="grid gap-2 text-xs sm:grid-cols-[72px_minmax(0,1fr)]">
                  <dt className="font-medium text-amber-700">标题</dt>
                  <dd className="break-words font-bold text-amber-900">{globalAnnouncement.title}</dd>
                  <dt className="font-medium text-amber-700">内容</dt>
                  <dd className="break-words leading-5">{globalAnnouncement.body}</dd>
                  <dt className="font-medium text-amber-700">发送人</dt>
                  <dd className="break-words">{globalAnnouncement.publisher || "管理员"}</dd>
                  <dt className="font-medium text-amber-700">发送时间</dt>
                  <dd>{formatFullDateTime(globalAnnouncement.createdAt)}</dd>
                </dl>
              </div>
            </div>
          ) : null}

          <div className="grid gap-4 md:grid-cols-2">
            <QuickLinkCard href="/settings/profile" icon={UsersRound} title="个人资料" description="查看姓名、手机号和所属部门/实验室。" />
            <QuickLinkCard href="/settings/account" icon={Settings2} title="账户安全" description="修改密码并刷新登录会话。" />
            <QuickLinkCard href="/reservations" icon={CalendarCheck2} title="预约记录" description="查看当前账号可访问的预约和使用状态。" />
            {showFinance ? <QuickLinkCard href="/finance" icon={Wallet} title="财务管理" description="查看个人费用流水和账户额度。" /> : null}
          </div>

          <Card className="overflow-hidden">
            <div className="flex flex-col justify-between gap-3 border-b bg-slate-50/30 p-4 sm:flex-row sm:items-center sm:p-6">
              <h3 className="text-lg font-bold text-slate-800">最近预约</h3>
              <div className="flex gap-1 rounded-md border bg-white p-1">
                <ReservationTab active={reservationView === "active"} href="/dashboard?reservations=active#reservations" label="进行中" />
                <ReservationTab active={reservationView === "completed"} href="/dashboard?reservations=completed#reservations" label="已完成" />
                <ReservationTab active={reservationView === "pending"} href="/dashboard?reservations=pending#reservations" label="审批中" />
              </div>
            </div>
            <div id="reservations">
              <div className="grid gap-3 p-4 xl:hidden">
                {visibleReservations.map((item) => (
                  <ReservationCard item={item} key={item.id} />
                ))}
              </div>
              <div className="hidden overflow-x-auto xl:block">
                <table className="w-full text-sm">
                <thead>
                  <tr className="border-b bg-slate-50/50">
                    <th className="px-6 py-4 text-left font-medium text-muted-foreground">单号 / 预约日期</th>
                    <th className="px-6 py-4 text-left font-medium text-muted-foreground">仪器名称</th>
                    <th className="px-6 py-4 text-left font-medium text-muted-foreground">使用时段</th>
                    <th className="px-6 py-4 text-left font-medium text-muted-foreground">状态</th>
                    <th className="px-6 py-4 text-right font-medium text-muted-foreground">费用</th>
                  </tr>
                </thead>
                <tbody className="divide-y">
                  {visibleReservations.map((item) => (
                    <tr className="transition-colors hover:bg-slate-50/50" key={item.id}>
                      <td className="px-6 py-4">
                        <p className="font-medium text-slate-800">#{item.id.slice(0, 8).toUpperCase()}</p>
                        <p className="text-xs text-muted-foreground">{formatDate(item.startTime)}</p>
                      </td>
                      <td className="px-6 py-4">{item.instrumentName}</td>
                      <td className="px-6 py-4">
                        {formatTime(item.startTime)} - {formatTime(item.endTime)}
                      </td>
                      <td className="px-6 py-4">
                        <StatusBadge status={item.status} />
                      </td>
                      <td className="px-6 py-4 text-right font-bold text-primary">¥{(item.fee ?? 0).toFixed(2)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
              </div>
              {visibleReservations.length === 0 ? <p className="p-6 text-sm text-slate-500">当前筛选下暂无预约记录。</p> : null}
            </div>
          </Card>

          {showFinance ? (
            <Card>
              <CardHeader>
                <CardTitle>费用流水</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                {ledger.length === 0 ? <p className="text-sm text-slate-500">暂无已完成预约产生的费用。</p> : null}
                {ledger.map((item) => (
                  <div className="flex flex-wrap items-center justify-between gap-3 rounded-lg border p-3" key={item.id}>
                    <div>
                      <p className="font-medium">{item.description}</p>
                      <p className="mt-1 text-sm text-slate-500">{item.groupName}</p>
                    </div>
                    <span className="font-bold">¥{(item.amount ?? 0).toFixed(2)}</span>
                  </div>
                ))}
              </CardContent>
            </Card>
          ) : null}
        </div>
      </div>
    </AppShell>
  );
}

function QuickLinkCard({ description, href, icon: Icon, title }: { description: string; href: string; icon: LucideIcon; title: string }) {
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

function Metric({ label, value, note }: { label: string; value: string; note: string }) {
  return (
    <Card className="p-6">
      <p className="mb-1 text-sm text-muted-foreground">{label}</p>
      <h3 className="text-2xl font-bold">{value}</h3>
      <p className="mt-2 flex items-center gap-1 text-xs font-medium text-emerald-600">
        <CheckCircle2 className="h-3.5 w-3.5" aria-hidden="true" />
        {note}
      </p>
    </Card>
  );
}

function ReservationTab({ active, href, label }: { active: boolean; href: string; label: string }) {
  return (
    <Link
      className={`inline-flex h-8 min-w-16 items-center justify-center whitespace-nowrap rounded px-3 text-xs font-medium ${
        active ? "bg-slate-100 text-slate-900 shadow-sm" : "text-muted-foreground hover:bg-slate-50"
      }`}
      href={href}
    >
      {label}
    </Link>
  );
}

function ReservationCard({ item }: { item: Reservation }) {
  return (
    <div className="rounded-lg border bg-white p-4 text-sm">
      <div className="flex flex-col justify-between gap-3 sm:flex-row sm:items-start">
        <div className="min-w-0">
          <p className="font-bold text-slate-900">#{item.id.slice(0, 8).toUpperCase()}</p>
          <p className="mt-1 break-words text-slate-600">{item.instrumentName}</p>
        </div>
        <StatusBadge status={item.status} />
      </div>
      <div className="mt-4 grid gap-3 text-xs text-slate-500 sm:grid-cols-3">
        <InfoItem label="预约日期" value={formatDate(item.startTime)} />
        <InfoItem label="使用时段" value={`${formatTime(item.startTime)} - ${formatTime(item.endTime)}`} />
        <InfoItem label="费用" value={`¥${(item.fee ?? 0).toFixed(2)}`} />
      </div>
    </div>
  );
}

function InfoItem({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-xs text-slate-500">{label}</p>
      <p className="mt-1 break-words font-medium text-slate-800">{value}</p>
    </div>
  );
}

function StatusBadge({ status }: { status: string }) {
  const className = status === "approved" ? "bg-emerald-50 text-emerald-700" : status === "pending" ? "bg-amber-50 text-amber-700" : "bg-slate-100 text-slate-600";
  return <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-bold ${className}`}>{reservationStatusLabel(status)}</span>;
}

function formatDate(value: string) {
  return new Date(value).toLocaleDateString("zh-CN", { timeZone: "Asia/Shanghai" });
}

function formatTime(value: string) {
  return new Date(value).toLocaleTimeString("zh-CN", { hour: "2-digit", minute: "2-digit", timeZone: "Asia/Shanghai" });
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

function formatFullDateTime(value: string) {
  if (/^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/.test(value)) {
    return value;
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  const parts = new Intl.DateTimeFormat("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false,
    timeZone: "Asia/Shanghai",
  }).formatToParts(date);
  const getPart = (type: Intl.DateTimeFormatPartTypes) => parts.find((part) => part.type === type)?.value ?? "";
  return `${getPart("year")}-${getPart("month")}-${getPart("day")} ${getPart("hour")}:${getPart("minute")}:${getPart("second")}`;
}
