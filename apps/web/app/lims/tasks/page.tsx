import { Microscope, Search } from "lucide-react";
import { AppShell } from "@/components/app-shell";
import { LimsTaskForm } from "@/components/extension-forms";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";
import { isTenantAdminRole } from "@/lib/permissions";

export default async function LimsTasksPage({
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
            <CardTitle>LIMS 检测任务</CardTitle>
          </CardHeader>
          <CardContent className="text-sm leading-6 text-slate-600">
            账号暂未通过审核，无法查看 LIMS 任务。
          </CardContent>
        </Card>
      </AppShell>
    );
  }
  const [tasks, samples, instruments] = await Promise.all([api.limsTasks(), api.samples(), api.instruments()]);
  const visibleTasks = tasks.filter((task) => [task.title, task.assayType, task.sampleCode ?? "", task.instrumentName ?? "", task.requesterName, task.status].some((value) => value.toLowerCase().includes(query)));
  const pendingCount = tasks.filter((item) => item.status === "pending" || item.status === "assigned").length;
  const runningCount = tasks.filter((item) => item.status === "running").length;

  return (
    <AppShell currentUser={currentUser}>
      <div className="mb-6 flex flex-col justify-between gap-4 xl:flex-row xl:items-end">
        <div className="min-w-0">
          <h1 className="text-2xl font-bold">LIMS 检测任务</h1>
          <p className="mt-1 text-sm text-muted-foreground">样本登记、任务分派、检测记录和结果摘要。</p>
        </div>
        <form action="/lims/tasks" className="flex w-full max-w-xl gap-2 xl:w-auto">
          <div className="relative min-w-0 flex-1">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" aria-hidden="true" />
            <input className="h-10 w-full rounded-md border bg-white pl-9 pr-3 text-sm" defaultValue={params.q ?? ""} name="q" placeholder="搜索任务、样本、仪器..." />
          </div>
          <button className="inline-flex h-10 items-center justify-center rounded-md bg-primary px-4 text-sm font-bold text-white" type="submit">
            筛选
          </button>
        </form>
      </div>

      <div className="mb-6 grid gap-4 sm:grid-cols-3">
        <Metric label="任务总数" value={tasks.length} />
        <Metric label="待处理" value={pendingCount} />
        <Metric label="进行中" value={runningCount} />
      </div>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_380px]">
        <Card>
          <CardHeader>
            <CardTitle>任务列表</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {visibleTasks.map((task) => (
              <div className="rounded-lg border p-4" key={task.id}>
                <div className="flex flex-col justify-between gap-3 sm:flex-row sm:items-start">
                  <div className="min-w-0">
                    <p className="break-words font-semibold text-slate-900">{task.title}</p>
                    <p className="mt-1 break-words text-xs text-slate-500">
                      {task.sampleCode || "未关联样本"} / {task.instrumentName || "未关联仪器"}
                    </p>
                  </div>
                  <span className="w-fit rounded bg-slate-100 px-2 py-1 text-xs font-bold text-slate-700">{taskStatusLabel(task.status)}</span>
                </div>
                <div className="mt-4 grid gap-3 text-sm md:grid-cols-2">
                  <InfoItem label="检测类型" value={task.assayType || "未设置"} />
                  <InfoItem label="优先级" value={task.priority} />
                  <InfoItem label="申请人" value={task.requesterName || "未设置"} />
                  <InfoItem label="截止时间" value={formatDateTime(task.dueAt)} />
                </div>
                <p className="mt-3 break-words text-sm leading-6 text-slate-600">{task.resultSummary || "暂无结果摘要"}</p>
              </div>
            ))}
            {visibleTasks.length === 0 ? <p className="rounded-lg border border-dashed p-4 text-sm text-slate-500">暂无 LIMS 任务。</p> : null}
          </CardContent>
        </Card>

        <aside className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Microscope className="h-5 w-5 text-primary" />
                新建任务
              </CardTitle>
            </CardHeader>
            <CardContent>
              {isTenantAdminRole(currentUser.role) ? <LimsTaskForm actorName={currentUser.name} instruments={instruments} samples={samples} /> : <p className="text-sm leading-6 text-slate-500">当前账号没有 LIMS 任务维护权限。</p>}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>流程说明</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3 text-sm leading-6 text-slate-600">
              <p>LIMS 任务可关联样本和仪器，结果摘要会成为 ELN 记录的上下文。</p>
              <p>紧急任务建议同步通知仪器管理员。</p>
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

function taskStatusLabel(status: string) {
  const labels: Record<string, string> = {
    pending: "待分配",
    assigned: "已分配",
    running: "进行中",
    completed: "已完成",
    cancelled: "已取消",
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
