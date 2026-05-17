import { ShieldCheck, Search } from "lucide-react";
import { AppShell } from "@/components/app-shell";
import { TrainingAuthorizationForm } from "@/components/extension-forms";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";

export default async function TrainingAuthorizationsPage({
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
            <CardTitle>授权记录</CardTitle>
          </CardHeader>
          <CardContent className="text-sm leading-6 text-slate-600">
            账号暂未通过审核，无法查看授权记录。
          </CardContent>
        </Card>
      </AppShell>
    );
  }
  const [authorizations, courses, instruments] = await Promise.all([api.trainingAuthorizations(), api.trainingCourses(), api.instruments()]);
  const visibleItems = authorizations.filter((item) => [item.userName, item.courseTitle, item.instrumentName ?? "", item.notes].some((value) => value.toLowerCase().includes(query)));
  const activeCount = authorizations.filter((item) => item.status === "active").length;
  const expiringCount = authorizations.filter((item) => item.status === "active" && new Date(item.expiresAt).getTime() - Date.now() <= 30 * 24 * 60 * 60 * 1000).length;

  return (
    <AppShell currentUser={currentUser}>
      <div className="mb-6 flex flex-col justify-between gap-4 xl:flex-row xl:items-end">
        <div className="min-w-0">
          <h1 className="text-2xl font-bold">授权记录</h1>
          <p className="mt-1 text-sm text-muted-foreground">查看培训授权、到期时间和限制条件。</p>
        </div>
        <form action="/training/authorizations" className="flex w-full max-w-xl gap-2 xl:w-auto">
          <div className="relative min-w-0 flex-1">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" aria-hidden="true" />
            <input className="h-10 w-full rounded-md border bg-white pl-9 pr-3 text-sm" defaultValue={params.q ?? ""} name="q" placeholder="搜索用户、课程、仪器..." />
          </div>
          <button className="inline-flex h-10 items-center justify-center rounded-md bg-primary px-4 text-sm font-bold text-white" type="submit">
            筛选
          </button>
        </form>
      </div>

      <div className="mb-6 grid gap-4 sm:grid-cols-3">
        <Metric label="授权总数" value={authorizations.length} />
        <Metric label="有效授权" value={activeCount} />
        <Metric label="30 天到期" value={expiringCount} />
      </div>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_360px]">
        <Card>
          <CardHeader>
            <CardTitle>授权列表</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {visibleItems.map((item) => (
              <div className="rounded-lg border p-4" key={item.id}>
                <div className="flex flex-col justify-between gap-3 sm:flex-row sm:items-start">
                  <div className="min-w-0">
                    <p className="break-words font-semibold text-slate-900">{item.userName}</p>
                    <p className="mt-1 break-words text-xs text-slate-500">
                      {item.courseTitle}
                      {item.instrumentName ? ` / ${item.instrumentName}` : ""}
                    </p>
                  </div>
                  <span className="w-fit rounded bg-slate-100 px-2 py-1 text-xs font-bold text-slate-700">{authorizationStatusLabel(item.status)}</span>
                </div>
                <div className="mt-4 grid gap-3 text-sm md:grid-cols-2">
                  <InfoItem label="到期时间" value={formatDateTime(item.expiresAt)} />
                  <InfoItem label="备注" value={item.notes || "未填写"} />
                </div>
              </div>
            ))}
            {visibleItems.length === 0 ? <p className="rounded-lg border border-dashed p-4 text-sm text-slate-500">暂无授权记录。</p> : null}
          </CardContent>
        </Card>

        <aside className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <ShieldCheck className="h-5 w-5 text-primary" />
                提交授权
              </CardTitle>
            </CardHeader>
            <CardContent>
              <TrainingAuthorizationForm actorName={currentUser.name} courses={courses} instruments={instruments} />
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle>到期提醒</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3 text-sm leading-6 text-slate-600">
              <p>授权到期后需要重新培训或延长有效期。</p>
              <p>仪器若启用准入控制，未授权用户将无法预约。</p>
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

function authorizationStatusLabel(status: string) {
  const labels: Record<string, string> = {
    pending: "待审核",
    active: "有效",
    expired: "已过期",
    revoked: "已撤销",
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
