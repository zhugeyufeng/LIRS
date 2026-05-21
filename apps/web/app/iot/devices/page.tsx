import { Search, Wifi, Zap } from "lucide-react";
import { AppShell } from "@/components/app-shell";
import { IotDeviceForm } from "@/components/extension-forms";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";
import { isTenantAdminRole } from "@/lib/permissions";
import { iotDeviceStatusLabel } from "@/lib/status-labels";

export default async function IotDevicesPage({
  searchParams,
}: {
  searchParams?: Promise<{ q?: string }>;
}) {
  const params = (await searchParams) ?? {};
  const query = (params.q ?? "").trim().toLowerCase();
  const currentUser = await api.me();
  if (currentUser.status !== "active" || currentUser.role === "unassigned") {
    return (
      <AppShell currentUser={currentUser}>
        <Card>
          <CardHeader>
            <CardTitle>IoT 设备中心</CardTitle>
          </CardHeader>
          <CardContent className="text-sm leading-6 text-slate-600">账号暂未通过审核，无法查看 IoT 设备。</CardContent>
        </Card>
      </AppShell>
    );
  }
  const [devices, instruments] = await Promise.all([api.iotDevices(), api.instruments()]);
  const visibleDevices = devices.filter((device) =>
    [device.name, device.vendor, device.deviceCode, device.instrumentName ?? "", device.status, device.telemetry, device.notes].some((value) =>
      String(value ?? "").toLowerCase().includes(query),
    ),
  );
  const onlineCount = devices.filter((item) => item.online || item.status === "online").length;
  const boundCount = devices.filter((item) => item.instrumentId).length;
  const warningCount = devices.filter((item) => item.status === "warning").length;

  return (
    <AppShell currentUser={currentUser}>
      <div className="mb-6 flex flex-col justify-between gap-4 xl:flex-row xl:items-end">
        <div className="min-w-0">
          <h1 className="text-2xl font-bold">IoT 设备中心</h1>
          <p className="mt-1 text-sm text-muted-foreground">维护采集终端、设备绑定、在线状态和遥测数据。</p>
        </div>
        <form action="/iot/devices" className="flex w-full max-w-xl gap-2 xl:w-auto">
          <div className="relative min-w-0 flex-1">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" aria-hidden="true" />
            <input className="h-10 w-full rounded-md border bg-white pl-9 pr-3 text-sm" defaultValue={params.q ?? ""} name="q" placeholder="搜索设备、厂商、编码、仪器..." />
          </div>
          <button className="inline-flex h-10 items-center justify-center rounded-md bg-primary px-4 text-sm font-bold text-white" type="submit">
            筛选
          </button>
        </form>
      </div>

      <div className="mb-6 grid gap-4 sm:grid-cols-3">
        <Metric label="设备总数" value={devices.length} />
        <Metric label="在线设备" value={onlineCount} />
        <Metric label="仪器绑定" value={boundCount} />
      </div>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_400px]">
        <Card className="min-w-0">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Wifi className="h-5 w-5 text-primary" />
              设备列表
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {visibleDevices.map((device) => (
              <div className="rounded-lg border p-4" key={device.id}>
                <div className="flex flex-col justify-between gap-3 sm:flex-row sm:items-start">
                  <div className="min-w-0">
                    <p className="break-words font-semibold text-slate-900">{device.name}</p>
                    <p className="mt-1 break-words text-xs text-slate-500">
                      {device.vendor || "未设置厂商"} / {device.deviceCode || "未设置编码"}
                    </p>
                  </div>
                  <span className={`w-fit rounded px-2 py-1 text-xs font-bold ${deviceStatusClass(device.status, device.online)}`}>{deviceStatusLabel(device.status, device.online)}</span>
                </div>
                <div className="mt-4 grid gap-3 text-sm md:grid-cols-2">
                  <InfoItem label="绑定仪器" value={device.instrumentName || "未绑定"} />
                  <InfoItem label="最后在线" value={formatDateTime(device.lastSeenAt)} />
                </div>
                <div className="mt-4 space-y-2">
                  <p className="text-xs text-slate-500">遥测数据</p>
                  <div className="rounded-md bg-slate-50 p-3 text-sm leading-6 text-slate-700">{formatTelemetry(device.telemetry)}</div>
                </div>
                <p className="mt-3 break-words text-sm leading-6 text-slate-600">{device.notes || "暂无备注。"}</p>
              </div>
            ))}
            {visibleDevices.length === 0 ? <p className="rounded-lg border border-dashed p-4 text-sm text-slate-500">暂无 IoT 设备。</p> : null}
          </CardContent>
        </Card>

        <aside className="min-w-0 space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Zap className="h-5 w-5 text-primary" />
                新建设备
              </CardTitle>
            </CardHeader>
            <CardContent>
              {isTenantAdminRole(currentUser.role) ? <IotDeviceForm actorName={currentUser.name} instruments={instruments} /> : <p className="text-sm leading-6 text-slate-500">当前账号没有 IoT 设备维护权限。</p>}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>接入说明</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3 text-sm leading-6 text-slate-600">
              <p>IoT 设备可以按仪器绑定，后续可继续接入门禁、采集网关和运行状态同步。</p>
              <p>仪器状态和遥测数据都从数据库读取，避免页面显示与实际状态脱节。</p>
            </CardContent>
          </Card>
        </aside>
      </div>
    </AppShell>
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

function InfoItem({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-xs text-slate-500">{label}</p>
      <p className="mt-1 break-words font-medium text-slate-800">{value}</p>
    </div>
  );
}

function deviceStatusLabel(status: string, online: boolean) {
  return iotDeviceStatusLabel(status, online);
}

function deviceStatusClass(status: string, online: boolean) {
  if (online && status !== "disabled") {
    return "bg-emerald-50 text-emerald-700";
  }
  const classes: Record<string, string> = {
    online: "bg-emerald-50 text-emerald-700",
    offline: "bg-slate-100 text-slate-600",
    warning: "bg-amber-50 text-amber-700",
    disabled: "bg-slate-100 text-slate-500",
  };
  return classes[status] ?? "bg-slate-100 text-slate-700";
}

function formatTelemetry(value: string) {
  if (!value) {
    return "无遥测数据";
  }
  try {
    const parsed = JSON.parse(value) as Record<string, unknown>;
    if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
      return Object.entries(parsed)
        .slice(0, 6)
        .map(([key, item]) => `${key}: ${String(item)}`)
        .join(" / ");
    }
    return JSON.stringify(parsed);
  } catch {
    return value;
  }
}

function formatDateTime(value: string) {
  if (!value) {
    return "未设置";
  }
  return new Date(value).toLocaleString("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    timeZone: "Asia/Shanghai",
  });
}
