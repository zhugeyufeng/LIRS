import { Search, TestTube2 } from "lucide-react";
import { AppShell } from "@/components/app-shell";
import { SampleForm } from "@/components/extension-forms";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";
import { isMaterialAdminRole } from "@/lib/permissions";

export default async function SamplesPage({
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
            <CardTitle>样本管理</CardTitle>
          </CardHeader>
          <CardContent className="text-sm leading-6 text-slate-600">
            账号暂未通过审核，无法查看样本管理。
          </CardContent>
        </Card>
      </AppShell>
    );
  }
  const [samples, movements] = await Promise.all([api.samples(), api.sampleMovements()]);
  const visibleSamples = samples.filter((sample) => [sample.code, sample.name, sample.ownerName, sample.department, sample.groupName, sample.location, sample.status].some((value) => value.toLowerCase().includes(query)));
  const warningCount = samples.filter((sample) => sample.hazardLevel !== "normal").length;

  return (
    <AppShell currentUser={currentUser}>
      <div className="mb-6 flex flex-col justify-between gap-4 xl:flex-row xl:items-end">
        <div className="min-w-0">
          <h1 className="text-2xl font-bold">样本管理</h1>
          <p className="mt-1 text-sm text-muted-foreground">样本台账、存储位置和流转记录。</p>
        </div>
        <form action="/samples" className="flex w-full max-w-xl gap-2 xl:w-auto">
          <div className="relative min-w-0 flex-1">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" aria-hidden="true" />
            <input className="h-10 w-full rounded-md border bg-white pl-9 pr-3 text-sm" defaultValue={params.q ?? ""} name="q" placeholder="搜索编号、名称、负责人..." />
          </div>
          <button className="inline-flex h-10 items-center justify-center rounded-md bg-primary px-4 text-sm font-bold text-white" type="submit">
            筛选
          </button>
        </form>
      </div>

      <div className="mb-6 grid gap-4 sm:grid-cols-3">
        <Metric label="样本总数" value={samples.length} />
        <Metric label="风险样本" value={warningCount} />
        <Metric label="流转记录" value={movements.length} />
      </div>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_360px]">
        <Card>
          <CardHeader>
            <CardTitle>样本台账</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {visibleSamples.map((sample) => (
              <div className="rounded-lg border p-4" key={sample.id}>
                <div className="flex flex-col justify-between gap-3 sm:flex-row sm:items-start">
                  <div className="min-w-0">
                    <p className="break-words font-semibold text-slate-900">{sample.code}</p>
                    <p className="mt-1 break-words text-xs text-slate-500">{sample.name}</p>
                  </div>
                  <span className="w-fit rounded bg-slate-100 px-2 py-1 text-xs font-bold text-slate-700">{sampleStatusLabel(sample.status)}</span>
                </div>
                <div className="mt-4 grid gap-3 text-sm md:grid-cols-2">
                  <InfoItem label="负责人" value={sample.ownerName || "未设置"} />
                  <InfoItem label="位置" value={sample.location || "未设置"} />
                  <InfoItem label="部门/课题组" value={`${sample.department || "未设置"} / ${sample.groupName || "未设置"}`} />
                  <InfoItem label="风险等级" value={hazardLabel(sample.hazardLevel)} />
                  <InfoItem label="保存条件" value={sample.storageCondition || "未设置"} />
                  <InfoItem label="流转提醒" value={sample.hazardLevel === "danger" ? "高危样本需优先处理" : "正常"} />
                </div>
                <p className="mt-3 break-words text-sm leading-6 text-slate-600">{sample.description || "未填写说明"}</p>
              </div>
            ))}
            {visibleSamples.length === 0 ? <p className="rounded-lg border border-dashed p-4 text-sm text-slate-500">暂无样本台账。</p> : null}
          </CardContent>
        </Card>

        <aside className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <TestTube2 className="h-5 w-5 text-primary" />
                新建样本
              </CardTitle>
            </CardHeader>
            <CardContent>
              {isMaterialAdminRole(currentUser.role) ? <SampleForm actorName={currentUser.name} /> : <p className="text-sm leading-6 text-slate-500">当前账号没有样本维护权限。</p>}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>最近流转</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              {movements.slice(0, 8).map((item) => (
                <div className="rounded-lg border p-3 text-sm" key={item.id}>
                  <p className="font-semibold text-slate-900">{item.sampleCode}</p>
                  <p className="mt-1 break-words text-xs text-slate-500">{item.movementType} / {item.reason || "未填写原因"}</p>
                  <p className="mt-1 text-xs text-slate-500">
                    {item.fromLocation || "未登记"} → {item.toLocation || "未登记"}
                  </p>
                </div>
              ))}
              {movements.length === 0 ? <p className="rounded-lg border border-dashed p-4 text-sm text-slate-500">暂无流转记录。</p> : null}
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

function sampleStatusLabel(status: string) {
  const labels: Record<string, string> = {
    stored: "入库",
    testing: "检测中",
    checked_out: "外借",
    archived: "归档",
    disposed: "销毁",
  };
  return labels[status] ?? status;
}

function hazardLabel(level: string) {
  const labels: Record<string, string> = {
    normal: "普通",
    warning: "警示",
    danger: "高危",
  };
  return labels[level] ?? level;
}
