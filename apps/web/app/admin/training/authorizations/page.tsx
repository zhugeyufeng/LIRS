import { ShieldCheck, Search } from "lucide-react";
import { AdminShell, requireAdminSection } from "@/components/admin-shell";
import { TrainingAuthorizationEditDialog } from "@/components/training-authorization-actions";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";

export default async function AdminTrainingAuthorizationsPage({
  searchParams,
}: {
  searchParams?: Promise<{ q?: string; status?: string }>;
}) {
  await requireAdminSection("trainingAuthorizations");
  const params = (await searchParams) ?? {};
  const query = (params.q ?? "").trim().toLowerCase();
  const status = (params.status ?? "").trim();
  const [authorizations, courses, instruments, users] = await Promise.all([
    api.trainingAuthorizations().catch(() => []),
    api.trainingCourses().catch(() => []),
    api.instruments().catch(() => []),
    api.users().catch(() => []),
  ]);
  const visibleItems = authorizations.filter((item) => {
    const matchesStatus = !status || item.status === status;
    const matchesQuery =
      query === "" ||
      [item.userName, item.courseTitle, item.instrumentName ?? "", item.status, item.notes].some((value) => value.toLowerCase().includes(query));
    return matchesStatus && matchesQuery;
  });
  const pendingCount = authorizations.filter((item) => item.status === "pending").length;
  const activeCount = authorizations.filter((item) => item.status === "active").length;
  const expiringCount = authorizations.filter((item) => item.status === "active" && new Date(item.expiresAt).getTime() - Date.now() <= 30 * 24 * 60 * 60 * 1000).length;

  return (
    <AdminShell active="trainingAuthorizations" title="授权审批" description="审核仪器准入授权申请，维护有效期、授权仪器和撤销状态。">
      <div className="mb-6 grid gap-4 sm:grid-cols-3">
        <Metric label="待审核" value={pendingCount} />
        <Metric label="有效授权" value={activeCount} />
        <Metric label="30 天内到期" value={expiringCount} />
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <ShieldCheck className="h-5 w-5 text-primary" aria-hidden="true" />
            授权记录
          </CardTitle>
        </CardHeader>
        <CardContent>
          <form action="/admin/training/authorizations" className="mb-4 grid gap-3 lg:grid-cols-[minmax(0,1fr)_180px_auto]">
            <div className="relative min-w-0">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" aria-hidden="true" />
              <input className="h-10 w-full rounded-md border bg-white pl-9 pr-3 text-sm" defaultValue={params.q ?? ""} name="q" placeholder="搜索人员、课程、仪器、备注..." />
            </div>
            <select className="h-10 rounded-md border bg-white px-3 text-sm" defaultValue={status} name="status">
              <option value="">全部状态</option>
              <option value="pending">待审核</option>
              <option value="active">已授权</option>
              <option value="expired">已过期</option>
              <option value="revoked">已撤销</option>
            </select>
            <button className="inline-flex h-10 items-center justify-center rounded-md bg-primary px-4 text-sm font-bold text-white" type="submit">
              筛选
            </button>
          </form>

          <div className="grid gap-3 xl:hidden">
            {visibleItems.map((item) => (
              <article className="rounded-lg border bg-white p-4" key={item.id}>
                <div className="flex flex-col justify-between gap-3 sm:flex-row sm:items-start">
                  <div className="min-w-0">
                    <p className="break-words font-bold text-slate-900">{item.userName}</p>
                    <p className="mt-1 break-words text-xs text-slate-500">{item.courseTitle || "未关联课程"}</p>
                  </div>
                  <StatusBadge status={item.status} />
                </div>
                <div className="mt-4 grid gap-3 text-sm sm:grid-cols-2">
                  <Info label="仪器" value={item.instrumentName || "未关联仪器"} />
                  <Info label="到期" value={formatDateTime(item.expiresAt)} />
                </div>
                <p className="mt-3 break-words text-sm leading-6 text-slate-600">{item.notes || "暂无备注。"}</p>
                <div className="mt-4">
                  <TrainingAuthorizationEditDialog authorization={item} courses={courses} instruments={instruments} users={users} />
                </div>
              </article>
            ))}
          </div>

          <div className="hidden overflow-x-auto xl:block">
            <table className="w-full text-left text-sm">
              <thead className="border-b text-slate-500">
                <tr>
                  <th className="py-3 pr-4">用户</th>
                  <th className="py-3 pr-4">课程</th>
                  <th className="py-3 pr-4">仪器</th>
                  <th className="py-3 pr-4">到期时间</th>
                  <th className="py-3 pr-4">状态</th>
                  <th className="py-3 text-right">操作</th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {visibleItems.map((item) => (
                  <tr key={item.id}>
                    <td className="py-3 pr-4 font-medium">{item.userName}</td>
                    <td className="py-3 pr-4">{item.courseTitle || "未关联课程"}</td>
                    <td className="py-3 pr-4">{item.instrumentName || "未关联仪器"}</td>
                    <td className="py-3 pr-4 text-xs text-slate-500">{formatDateTime(item.expiresAt)}</td>
                    <td className="py-3 pr-4">
                      <StatusBadge status={item.status} />
                    </td>
                    <td className="py-3 text-right">
                      <TrainingAuthorizationEditDialog authorization={item} courses={courses} instruments={instruments} users={users} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {visibleItems.length === 0 ? <p className="rounded-lg border border-dashed p-4 text-sm text-slate-500">暂无授权记录。</p> : null}
        </CardContent>
      </Card>
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

function Info({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-xs text-slate-500">{label}</p>
      <p className="mt-1 break-words font-medium text-slate-800">{value}</p>
    </div>
  );
}

function StatusBadge({ status }: { status: string }) {
  const classes: Record<string, string> = {
    pending: "bg-amber-50 text-amber-700",
    active: "bg-emerald-50 text-emerald-700",
    expired: "bg-slate-100 text-slate-600",
    revoked: "bg-rose-50 text-rose-700",
  };
  return <span className={`inline-flex w-fit rounded-full px-2 py-1 text-xs font-bold ${classes[status] ?? "bg-slate-100 text-slate-600"}`}>{statusLabel(status)}</span>;
}

function statusLabel(status: string) {
  const labels: Record<string, string> = {
    pending: "待审核",
    active: "已授权",
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
