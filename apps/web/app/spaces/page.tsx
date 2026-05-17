import Link from "next/link";
import { Building2, Search } from "lucide-react";
import { AppShell } from "@/components/app-shell";
import { SpaceForm } from "@/components/extension-forms";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";
import { isTenantAdminRole } from "@/lib/permissions";

export default async function SpacesPage({
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
            <CardTitle>空间资源</CardTitle>
          </CardHeader>
          <CardContent className="text-sm leading-6 text-slate-600">
            账号暂未通过审核，无法查看空间资源。
          </CardContent>
        </Card>
      </AppShell>
    );
  }
  const [spaces, reservations] = await Promise.all([api.spaces(), api.spaceReservations()]);
  const visibleSpaces = spaces.filter((space) => [space.name, space.kind, space.department, space.location, space.description].some((value) => value.toLowerCase().includes(query)));
  const availableCount = spaces.filter((space) => space.status === "available").length;
  const reservationCount = reservations.length;

  return (
    <AppShell currentUser={currentUser}>
      <div className="mb-6 flex flex-col justify-between gap-4 xl:flex-row xl:items-end">
        <div className="min-w-0">
          <h1 className="text-2xl font-bold">空间资源</h1>
          <p className="mt-1 text-sm text-muted-foreground">实验空间、会议室和样本库预约入口。</p>
        </div>
        <form action="/spaces" className="flex w-full max-w-xl gap-2 xl:w-auto">
          <div className="relative min-w-0 flex-1">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" aria-hidden="true" />
            <input className="h-10 w-full rounded-md border bg-white pl-9 pr-3 text-sm" defaultValue={params.q ?? ""} name="q" placeholder="搜索空间、位置、部门..." />
          </div>
          <button className="inline-flex h-10 items-center justify-center rounded-md bg-primary px-4 text-sm font-bold text-white" type="submit">
            筛选
          </button>
        </form>
      </div>

      <div className="mb-6 grid gap-4 sm:grid-cols-3">
        <Metric label="空间总数" value={spaces.length} />
        <Metric label="可用空间" value={availableCount} />
        <Metric label="预约记录" value={reservationCount} />
      </div>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_360px]">
        <Card>
          <CardHeader>
            <CardTitle>空间列表</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {visibleSpaces.map((space) => (
              <div className="rounded-lg border p-4" key={space.id}>
                <div className="flex flex-col justify-between gap-3 sm:flex-row sm:items-start">
                  <div className="min-w-0">
                    <p className="break-words font-semibold text-slate-900">{space.name}</p>
                    <p className="mt-1 break-words text-xs text-slate-500">
                      {space.kind} / {space.department || "未设置部门"} / {space.location}
                    </p>
                  </div>
                  <span className="w-fit rounded bg-slate-100 px-2 py-1 text-xs font-bold text-slate-700">{spaceStatusLabel(space.status)}</span>
                </div>
                <div className="mt-4 grid gap-3 text-sm md:grid-cols-2">
                  <InfoItem label="容量" value={`${space.capacity} 人`} />
                  <InfoItem label="门禁点位" value={space.accessControlPoint || "未设置"} />
                </div>
                <div className="mt-3 flex flex-wrap gap-2">
                  <Link className="inline-flex h-9 items-center justify-center rounded-md border px-3 text-xs font-bold text-slate-700 hover:bg-slate-50" href={`/spaces/${space.id}/reserve`}>
                    进入预约
                  </Link>
                </div>
                <p className="mt-3 break-words text-sm leading-6 text-slate-600">{space.description || "未填写空间说明"}</p>
              </div>
            ))}
            {visibleSpaces.length === 0 ? <p className="rounded-lg border border-dashed p-4 text-sm text-slate-500">暂无空间资源。</p> : null}
          </CardContent>
        </Card>

        <aside className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Building2 className="h-5 w-5 text-primary" />
                新建空间
              </CardTitle>
            </CardHeader>
            <CardContent>
              {isTenantAdminRole(currentUser.role) ? <SpaceForm actorName={currentUser.name} /> : <p className="text-sm leading-6 text-slate-500">当前账号没有空间维护权限。</p>}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>预约说明</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3 text-sm leading-6 text-slate-600">
              <p>空间预约会按租户、时间段和状态自动检查冲突。</p>
              <p>会议室和样本前处理区都可以在这里统一管理。</p>
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

function spaceStatusLabel(status: string) {
  const labels: Record<string, string> = {
    available: "可用",
    busy: "占用",
    maintenance: "维护中",
    disabled: "停用",
  };
  return labels[status] ?? status;
}
