import Link from "next/link";
import { Search, ThermometerSun } from "lucide-react";
import { AdminShell, requireAdminSection } from "@/components/admin-shell";
import { InstrumentCreateForm, InstrumentStatusForm } from "@/components/instrument-management-form";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";
import { formatServiceWindow } from "@/lib/instrument-rules";
import { instrumentStatusLabel } from "@/lib/status-labels";

export default async function AdminInstrumentsPage({
  searchParams,
}: {
  searchParams?: Promise<{ q?: string; category?: string; status?: string }>;
}) {
  await requireAdminSection("instruments");
  const params = (await searchParams) ?? {};
  const query = (params.q ?? "").trim().toLowerCase();
  const [instruments, organizationUnits] = await Promise.all([api.instruments().catch(() => []), api.organizationUnits().catch(() => [])]);
  const departments = organizationUnits.filter((item) => item.kind === "department");
  const groups = organizationUnits.filter((item) => item.kind === "group");
  const categories = Array.from(new Set(instruments.map((item) => item.category).filter(Boolean))).sort((a, b) => a.localeCompare(b, "zh-CN"));
  const visibleInstruments = instruments.filter((item) => {
    const matchesSearch =
      query === "" ||
      [item.name, item.category, item.department, item.groupName, item.location, item.model, item.assetCode].some((value) => value.toLowerCase().includes(query));
    const matchesCategory = !params.category || item.category === params.category;
    const matchesStatus = !params.status || item.status === params.status;
    return matchesSearch && matchesCategory && matchesStatus;
  });

  return (
    <AdminShell active="instruments" title="仪器管理" description="维护仪器档案、部门分类、归属团队、预约窗口、可预约时段和每台仪器对应的门禁授权组/点位。">
      <div className="mb-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <Metric label="仪器总数" value={instruments.length} />
        <Metric label="可用仪器" value={instruments.filter((item) => item.status === "available").length} />
        <Metric label="维护中" value={instruments.filter((item) => item.status === "maintenance").length} />
        <Metric label="已停用" value={instruments.filter((item) => item.status === "disabled").length} />
      </div>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_420px]">
        <Card className="min-w-0">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <ThermometerSun className="h-5 w-5 text-primary" />
              仪器状态与预约规则
            </CardTitle>
          </CardHeader>
          <CardContent>
            <form action="/admin/instruments" className="mb-4 grid gap-3 xl:grid-cols-[minmax(0,1fr)_180px_180px_auto]">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" aria-hidden="true" />
                <input className="h-10 w-full rounded-md border bg-white pl-10 pr-3 text-sm" defaultValue={params.q ?? ""} name="q" placeholder="搜索名称、型号、资产编号、位置" />
              </div>
              <select className="h-10 rounded-md border bg-white px-3 text-sm" defaultValue={params.category ?? ""} name="category">
                <option value="">全部分类</option>
                {categories.map((category) => (
                  <option key={category} value={category}>
                    {category}
                  </option>
                ))}
              </select>
              <select className="h-10 rounded-md border bg-white px-3 text-sm" defaultValue={params.status ?? ""} name="status">
                <option value="">全部状态</option>
                <option value="available">可用</option>
                <option value="busy">繁忙</option>
                <option value="maintenance">维护中</option>
                <option value="disabled">停用</option>
              </select>
              <button className="inline-flex h-10 w-full min-w-20 items-center justify-center whitespace-nowrap rounded-md bg-primary px-4 text-sm font-bold text-white xl:w-auto" type="submit">
                筛选
              </button>
            </form>

            <div className="space-y-3">
              {visibleInstruments.map((item) => (
                <div className="rounded-lg border p-3" key={item.id}>
                  <div className="flex flex-col justify-between gap-3 sm:flex-row sm:items-start">
                    <div className="min-w-0">
                      <Link className="text-sm font-bold text-primary hover:underline" href={`/instruments/${item.id}`}>
                        {item.name}
                      </Link>
                      <p className="mt-1 break-words text-xs leading-5 text-muted-foreground">
                        {item.category} / {item.department} / {item.location}
                      </p>
                      <p className="mt-1 break-words text-xs leading-5 text-slate-500">
                        团队：{item.groupName || "部门直属"}
                      </p>
                      <p className="mt-1 break-words text-xs leading-5 text-slate-500">
                        开放 {formatServiceWindow(item)}，未来 {item.bookingWindowDays} 天可预约，每段 {item.bookingIntervalHours} 小时，最长 {item.maxBookingHours} 小时。
                      </p>
                      <p className="mt-1 break-words text-xs leading-5 text-slate-500">
                        门禁：{item.accessControlEnabled ? `${item.accessControlGroup || "使用全局默认授权组"}${item.accessControlPoint ? ` / ${item.accessControlPoint}` : ""}` : "未启用"}
                      </p>
                    </div>
                    <span className="w-fit shrink-0 rounded bg-slate-100 px-2 py-1 text-[10px] font-bold text-slate-600">{instrumentStatusLabel(item.status)}</span>
                  </div>
                  <InstrumentStatusForm departments={departments.map((unit) => unit.name)} groups={groups} instrument={item} />
                </div>
              ))}
              {visibleInstruments.length === 0 ? <p className="rounded-lg border p-4 text-sm text-slate-500">当前筛选下暂无仪器。</p> : null}
            </div>
          </CardContent>
        </Card>

        <aside className="min-w-0">
          <Card>
            <CardHeader>
              <CardTitle>新增仪器</CardTitle>
            </CardHeader>
            <CardContent>
              <InstrumentCreateForm departments={departments.map((item) => item.name)} groups={groups} />
            </CardContent>
          </Card>
        </aside>
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
