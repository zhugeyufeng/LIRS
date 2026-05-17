import { FileText, Link2, Search, Signature } from "lucide-react";
import Link from "next/link";
import { AppShell } from "@/components/app-shell";
import { ElnRecordForm } from "@/components/extension-forms";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";

export default async function ElnRecordsPage({
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
            <CardTitle>ELN 实验记录</CardTitle>
          </CardHeader>
          <CardContent className="text-sm leading-6 text-slate-600">账号暂未通过审核，无法查看 ELN 记录。</CardContent>
        </Card>
      </AppShell>
    );
  }
  const [records, tasks] = await Promise.all([api.elnRecords(), api.limsTasks().catch(() => [])]);
  const visibleRecords = records.filter((record) =>
    [record.title, record.authorName, record.project, record.linkedTaskTitle ?? "", record.status, record.content].some((value) =>
      String(value ?? "").toLowerCase().includes(query),
    ),
  );
  const signedCount = records.filter((item) => item.status === "signed").length;
  const linkedCount = records.filter((item) => item.linkedTaskId).length;

  return (
    <AppShell currentUser={currentUser}>
      <div className="mb-6 flex flex-col justify-between gap-4 xl:flex-row xl:items-end">
        <div className="min-w-0">
          <h1 className="text-2xl font-bold">ELN 实验记录</h1>
          <p className="mt-1 text-sm text-muted-foreground">记录实验过程、原始数据说明、签名状态和关联任务。</p>
        </div>
        <form action="/eln/records" className="flex w-full max-w-xl gap-2 xl:w-auto">
          <div className="relative min-w-0 flex-1">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" aria-hidden="true" />
            <input className="h-10 w-full rounded-md border bg-white pl-9 pr-3 text-sm" defaultValue={params.q ?? ""} name="q" placeholder="搜索标题、作者、项目、任务..." />
          </div>
          <button className="inline-flex h-10 items-center justify-center rounded-md bg-primary px-4 text-sm font-bold text-white" type="submit">
            筛选
          </button>
        </form>
      </div>

      <div className="mb-6 grid gap-4 sm:grid-cols-3">
        <Metric label="记录总数" value={records.length} />
        <Metric label="已签名" value={signedCount} />
        <Metric label="关联任务" value={linkedCount} />
      </div>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_400px]">
        <Card className="min-w-0">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <FileText className="h-5 w-5 text-primary" />
              记录列表
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {visibleRecords.map((record) => (
              <div className="rounded-lg border p-4" key={record.id}>
                <div className="flex flex-col justify-between gap-3 sm:flex-row sm:items-start">
                  <div className="min-w-0">
                    <p className="break-words font-semibold text-slate-900">{record.title}</p>
                    <p className="mt-1 break-words text-xs text-slate-500">
                      {record.authorName} / {record.project || "未设置项目"}
                    </p>
                  </div>
                  <span className={`w-fit rounded px-2 py-1 text-xs font-bold ${statusClass(record.status)}`}>{statusLabel(record.status)}</span>
                </div>
                <div className="mt-4 grid gap-3 text-sm md:grid-cols-2">
                  <InfoItem label="关联任务" value={record.linkedTaskTitle || "未关联"} />
                  <InfoItem label="签名时间" value={formatDateTime(record.signedAt)} />
                </div>
                <p className="mt-3 break-words text-sm leading-6 text-slate-600">{record.content || "暂无实验记录内容。"}</p>
                {record.linkedTaskId ? (
                  <Link className="mt-3 inline-flex items-center gap-2 text-sm font-medium text-primary" href="/lims/tasks">
                    <Link2 className="h-4 w-4" />
                    跳转 LIMS 任务
                  </Link>
                ) : null}
              </div>
            ))}
            {visibleRecords.length === 0 ? <p className="rounded-lg border border-dashed p-4 text-sm text-slate-500">暂无 ELN 记录。</p> : null}
          </CardContent>
        </Card>

        <aside className="min-w-0 space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Signature className="h-5 w-5 text-primary" />
                新建记录
              </CardTitle>
            </CardHeader>
            <CardContent>
              <ElnRecordForm actorName={currentUser.name} tasks={tasks} />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>使用说明</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3 text-sm leading-6 text-slate-600">
              <p>ELN 记录会自动保留创建时间和签名时间，适合与 LIMS 任务联动归档。</p>
              <p>已签名记录可在列表中直接查看关联任务和实验内容摘要。</p>
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

function statusLabel(status: string) {
  const labels: Record<string, string> = {
    draft: "草稿",
    submitted: "已提交",
    signed: "已签名",
    archived: "已归档",
  };
  return labels[status] ?? status;
}

function statusClass(status: string) {
  const classes: Record<string, string> = {
    draft: "bg-slate-100 text-slate-700",
    submitted: "bg-amber-50 text-amber-700",
    signed: "bg-emerald-50 text-emerald-700",
    archived: "bg-slate-100 text-slate-500",
  };
  return classes[status] ?? "bg-slate-100 text-slate-700";
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
