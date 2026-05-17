import { Activity, AlertTriangle, BarChart3, Clock3, Gauge, TrendingUp, type LucideIcon } from "lucide-react";
import { AdminShell, requireAdminSection } from "@/components/admin-shell";
import { OperationsExportButton } from "@/components/operations-export-button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";

export default async function OperationsPage() {
  await requireAdminSection("operations");
  const operations = await api.operations();
  const maxTrend = operations.reservationTrend.reduce((max, item) => Math.max(max, item.count), 1);
  const maxLoad = operations.instrumentLoads.reduce((max, item) => Math.max(max, item.hours), 1);

  return (
    <AdminShell active="operations" title="运营看板" description={`数据更新时间：${formatDateTime(operations.updatedAt)}，统计数据延迟不超过 5 分钟。`}>
      <div className="mb-6 flex flex-col justify-between gap-4 sm:flex-row sm:items-center">
        <div className="flex flex-wrap items-center gap-3">
          <span className="rounded bg-emerald-50 px-3 py-1 text-xs font-bold text-emerald-700">月度可用性目标 99.5%</span>
          <OperationsExportButton />
        </div>
      </div>

      <div className="mb-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-5">
        <Metric icon={Gauge} label="今日预约" value={operations.dashboard.todayReservations} />
        <Metric icon={Clock3} label="待审批" value={operations.dashboard.pendingApprovals} />
        <Metric icon={Activity} label="使用中仪器" value={operations.inUseInstruments} />
        <Metric icon={TrendingUp} label="本月收入" value={`¥${operations.dashboard.monthlyRevenue.toFixed(0)}`} />
        <Metric icon={AlertTriangle} label="告警" value={operations.alertCount} />
      </div>

      <div className="grid gap-6 xl:grid-cols-2">
        <Card className="min-w-0">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <BarChart3 className="h-5 w-5 text-primary" />
              24 小时预约趋势
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex h-56 items-end gap-1">
              {operations.reservationTrend.map((item) => (
                <div className="flex min-w-0 flex-1 flex-col items-center gap-2" key={item.hour}>
                  <div className="w-full rounded-t bg-primary/70" style={{ height: `${Math.max((item.count / maxTrend) * 180, 4)}px` }} />
                  <span className="text-[10px] text-slate-500">{item.hour.slice(0, 2)}</span>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        <Card className="min-w-0">
          <CardHeader>
            <CardTitle>仪器负荷分布</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {operations.instrumentLoads.map((item) => (
              <div key={item.instrumentName}>
                <div className="mb-1 flex items-start justify-between gap-3 text-sm">
                  <span className="min-w-0 break-words">{item.instrumentName}</span>
                  <span className="shrink-0 font-bold">{item.hours.toFixed(1)} 小时</span>
                </div>
                <div className="h-2 rounded bg-slate-100">
                  <div className="h-2 rounded bg-primary" style={{ width: `${Math.max((item.hours / maxLoad) * 100, 4)}%` }} />
                </div>
              </div>
            ))}
          </CardContent>
        </Card>

        <Card className="min-w-0">
          <CardHeader>
            <CardTitle>审批效率</CardTitle>
          </CardHeader>
          <CardContent className="grid gap-3 sm:grid-cols-3">
            {operations.approvalEfficiency.map((item) => (
              <div className="rounded-lg border bg-slate-50 p-4" key={item.label}>
                <p className="text-xs text-slate-500">{item.label}</p>
                <p className="mt-2 text-2xl font-bold">{item.hours.toFixed(1)}h</p>
              </div>
            ))}
          </CardContent>
        </Card>

        <Card className="min-w-0">
          <CardHeader>
            <CardTitle>告警来源</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {operations.alerts.length === 0 ? <p className="text-sm text-slate-500">当前无告警。</p> : null}
            {operations.alerts.map((item) => (
              <div className="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800" key={`${item.source}-${item.body}`}>
                <span className="font-bold">{item.source}</span> / {item.body}
              </div>
            ))}
          </CardContent>
        </Card>
      </div>
    </AdminShell>
  );
}

function Metric({ icon: Icon, label, value }: { icon: LucideIcon; label: string; value: string | number }) {
  return (
    <div className="rounded-lg border bg-white p-4">
      <div className="mb-2 flex items-center gap-2 text-xs text-slate-500">
        <Icon className="h-4 w-4" aria-hidden="true" />
        {label}
      </div>
      <p className="text-2xl font-bold">{value}</p>
    </div>
  );
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
