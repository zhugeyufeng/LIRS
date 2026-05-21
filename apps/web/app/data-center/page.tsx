import { Database, Activity, Clock3, Cpu, ShieldCheck, TestTube2 } from "lucide-react";
import { AppShell } from "@/components/app-shell";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";

export default async function DataCenterPage() {
  const [dashboard, operations, instruments, materials, samples, devices, notifications] = await Promise.all([
    api.dashboard().catch(() => null),
    api.operations().catch(() => null),
    api.instruments().catch(() => []),
    api.materials().catch(() => []),
    api.samples().catch(() => []),
    api.iotDevices().catch(() => []),
    api.notifications().catch(() => []),
  ]);

  return (
    <AppShell>
      <div className="mb-6">
        <h1 className="text-2xl font-bold">数据中台</h1>
        <p className="mt-1 text-sm text-muted-foreground">汇总实验室运行、培训、样本和物联网设备的核心数据。</p>
      </div>

      <div className="mb-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <Metric label="仪器" value={instruments.length} icon={Cpu} />
        <Metric label="样本" value={samples.length} icon={TestTube2} />
        <Metric label="物联网设备" value={devices.length} icon={Database} />
        <Metric label="通知" value={notifications.length} icon={ShieldCheck} />
      </div>

      <div className="mb-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <Metric label="资源" value={materials.length} icon={Activity} />
        <Metric label="更新时间" value={operations ? formatDateTime(operations.updatedAt) : "未获取"} icon={Clock3} />
      </div>

      <div className="grid gap-6 xl:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>运行概览</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 text-sm leading-6 text-slate-600">
            {dashboard ? (
              <>
                <p>今日预约：{dashboard.todayReservations}</p>
                <p>待审批：{dashboard.pendingApprovals}</p>
                <p>使用中：{dashboard.inUseReservations}</p>
                <p>已履约：{dashboard.completedReservations}</p>
                <p>履约率：{dashboard.fulfillmentRate.toFixed(1)}%</p>
                <p>月收入：¥{dashboard.monthlyRevenue.toFixed(0)}</p>
              </>
            ) : (
              <p>暂无运行概览数据。</p>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>数据说明</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 text-sm leading-6 text-slate-600">
            <p>这里用于后续统一对接报表、导出和外部系统接口。</p>
            <p>当前页面直接聚合数据库中的业务数据，避免再维护一套脱节的静态统计。</p>
          </CardContent>
        </Card>
      </div>
    </AppShell>
  );
}

function Metric({ label, value, icon: Icon }: { label: string; value: string | number; icon: typeof Cpu }) {
  return (
    <div className="rounded-lg border bg-white p-4">
      <div className="flex items-center justify-between gap-3">
        <p className="text-sm text-slate-500">{label}</p>
        <Icon className="h-4 w-4 text-primary" aria-hidden="true" />
      </div>
      <p className="mt-2 break-words text-2xl font-bold">{value}</p>
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
