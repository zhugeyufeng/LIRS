import { CalendarClock, Wrench } from "lucide-react";
import { AdminShell, requireAdminSection } from "@/components/admin-shell";
import { MaintenanceCompleteButton, MaintenanceForm } from "@/components/maintenance-form";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";
import { maintenanceStatusLabel, maintenanceTypeLabel, priorityLabel } from "@/lib/status-labels";

export default async function MaintenancePage() {
  await requireAdminSection("maintenance");
  const [instruments, orders] = await Promise.all([api.instruments(), api.maintenance()]);
  const activeOrders = orders.filter((item) => item.status !== "completed" && item.status !== "cancelled");

  return (
    <AdminShell active="maintenance" title="设备维护日志" description="维护窗口会联动仪器可预约状态，并取消受影响的未开始预约。">
      <div className="mb-6 grid gap-4 md:grid-cols-3">
        <Metric label="维护中工单" value={activeOrders.length} />
        <Metric label="维护中仪器" value={instruments.filter((item) => item.status === "maintenance").length} />
        <Metric label="历史工单" value={orders.length} />
      </div>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_380px]">
        <Card className="min-w-0">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <CalendarClock className="h-5 w-5 text-primary" />
              工单列表
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {orders.length === 0 ? <p className="text-sm text-slate-500">暂无维护工单。</p> : null}
            {orders.map((item) => (
              <div className="rounded-lg border p-4" key={item.id}>
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div className="min-w-0">
                    <p className="font-bold">{item.instrumentName}</p>
                    <p className="mt-1 text-sm text-slate-500">
                      {maintenanceTypeLabel(item.type)} / {priorityLabel(item.priority)} / {item.handler}
                    </p>
                    <p className="mt-2 break-words text-sm leading-6">{item.description}</p>
                    <p className="mt-1 break-words text-xs text-slate-500">
                      {formatDateTime(item.startTime)} - {formatDateTime(item.endTime)}
                    </p>
                  </div>
                  <span className="shrink-0 rounded bg-slate-100 px-2 py-1 text-xs font-bold">{maintenanceStatusLabel(item.status)}</span>
                </div>
                <div className="mt-3">
                  <MaintenanceCompleteButton id={item.id} status={item.status} />
                </div>
              </div>
            ))}
          </CardContent>
        </Card>

        <Card className="min-w-0">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Wrench className="h-5 w-5 text-primary" />
              创建维护窗口
            </CardTitle>
          </CardHeader>
          <CardContent>
            <MaintenanceForm instruments={instruments} />
          </CardContent>
        </Card>
      </div>
    </AdminShell>
  );
}

function Metric({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-lg border bg-white p-4">
      <p className="text-sm text-slate-500">{label}</p>
      <p className="mt-2 text-2xl font-bold">{value}</p>
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
