import Link from "next/link";
import { BarChart3, Bell, MonitorPlay, TrendingUp } from "lucide-react";
import { AdminShell, requireAdmin } from "@/components/admin-shell";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";

export default async function AdminAnalyticsPage() {
  await requireAdmin();
  const [operations, dashboard, materials] = await Promise.all([api.operations(), api.dashboard(), api.materials()]);
  const topLoads = operations.instrumentLoads.slice(0, 6);
  const alerts = operations.alerts.slice(0, 8);
  const materialWarnings = materials.filter((item) => ["near_expiry", "expired", "open_expired", "freeze_thaw_exceeded", "low", "damaged"].includes(item.status));

  return (
    <AdminShell active="analytics" title="运营分析中心" description="从仪器、耗材、审批、财务和风险预警五个方向观察平台运行。">
      <div className="mb-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <Metric label="今日预约" value={dashboard.todayReservations} />
        <Metric label="待审批" value={dashboard.pendingApprovals} />
        <Metric label="已履约" value={dashboard.completedReservations} />
        <Metric label="履约率" value={`${dashboard.fulfillmentRate.toFixed(1)}%`} />
      </div>

      <div className="grid gap-6 xl:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <BarChart3 className="h-5 w-5 text-primary" />
              热门仪器
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {topLoads.map((item) => (
              <div className="flex items-center justify-between rounded-lg border p-3 text-sm" key={item.instrumentName}>
                <span className="min-w-0 break-words font-medium text-slate-900">{item.instrumentName}</span>
                <span className="shrink-0 rounded bg-slate-100 px-2 py-1 text-xs font-bold text-slate-600">{item.hours.toFixed(1)} 小时</span>
              </div>
            ))}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <TrendingUp className="h-5 w-5 text-primary" />
              审批效率
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {operations.approvalEfficiency.map((item) => (
              <div className="flex items-center justify-between rounded-lg border p-3 text-sm" key={item.label}>
                <span className="font-medium text-slate-900">{item.label}</span>
                <span className="shrink-0 rounded bg-slate-100 px-2 py-1 text-xs font-bold text-slate-600">{item.hours.toFixed(1)} 小时</span>
              </div>
            ))}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <MonitorPlay className="h-5 w-5 text-primary" />
              风险预警
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {alerts.length === 0 ? <p className="text-sm text-slate-500">暂无预警。</p> : null}
            {alerts.map((item, index) => (
              <div className="rounded-lg border bg-amber-50 p-3 text-sm text-amber-800" key={`${item.source}-${index}`}>
                <p className="font-medium">{item.source}</p>
                <p className="mt-1 break-words text-xs leading-5">{item.body}</p>
              </div>
            ))}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Bell className="h-5 w-5 text-primary" />
              资源预警
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {materialWarnings.length === 0 ? <p className="text-sm text-slate-500">暂无资源预警。</p> : null}
            {materialWarnings.slice(0, 8).map((item) => (
              <div className="rounded-lg border p-3 text-sm" key={item.id}>
                <p className="font-medium text-slate-900">{item.name}</p>
                <p className="mt-1 text-xs text-slate-500">
                  {materialStatusLabel(item.status)} / 当前 {item.stock}
                  {item.unit} / 告警线 {item.warningLine}
                  {item.unit} / 有效期 {item.expiresAt || "未登记"}
                </p>
              </div>
            ))}
          </CardContent>
        </Card>
      </div>

      <div className="mt-6 flex flex-wrap gap-3">
        <Link className="inline-flex h-10 items-center justify-center rounded-md border px-4 text-sm font-medium text-slate-700 hover:bg-slate-50" href="/operations">
          运营看板
        </Link>
        <Link className="inline-flex h-10 items-center justify-center rounded-md border px-4 text-sm font-medium text-slate-700 hover:bg-slate-50" href="/admin/security">
          安全审计
        </Link>
      </div>
    </AdminShell>
  );
}

function materialStatusLabel(status: string) {
  const labels: Record<string, string> = {
    normal: "正常",
    near_expiry: "临期",
    low: "低库存",
    expired: "过期",
    open_expired: "开封超期",
    freeze_thaw_exceeded: "冻融超限",
    damaged: "损毁",
    disabled: "停用",
  };
  return labels[status] ?? status;
}

function Metric({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="rounded-lg border bg-white p-4">
      <p className="text-sm text-slate-500">{label}</p>
      <p className="mt-2 text-2xl font-bold">{value}</p>
    </div>
  );
}
