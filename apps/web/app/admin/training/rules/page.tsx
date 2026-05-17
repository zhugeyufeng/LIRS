import { Search, ShieldCheck } from "lucide-react";
import { AdminShell, requireAdminSection } from "@/components/admin-shell";
import { TrainingRuleForm } from "@/components/extension-forms";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";

export default async function AdminTrainingRulesPage({
  searchParams,
}: {
  searchParams?: Promise<{ q?: string }>;
}) {
  const currentUser = await requireAdminSection("trainingRules");
  const params = (await searchParams) ?? {};
  const query = (params.q ?? "").trim().toLowerCase();
  const [rules, instruments] = await Promise.all([api.trainingRules().catch(() => []), api.instruments()]);
  const visibleRules = rules.filter((item) => [item.instrumentName ?? "", item.status, item.notes].some((value) => value.toLowerCase().includes(query)));
  const requiredCount = rules.filter((item) => item.requireTraining || item.requireExam || item.requireApproval).length;

  return (
    <AdminShell active="trainingRules" title="准入规则" description="配置仪器是否需要培训、考试和审批后才能预约。">
      <div className="mb-6 grid gap-4 sm:grid-cols-3">
        <Metric label="规则总数" value={rules.length} />
        <Metric label="启用规则" value={rules.filter((item) => item.status === "active").length} />
        <Metric label="强制约束" value={requiredCount} />
        <Metric label="维护人" value={currentUser.name} />
      </div>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_400px]">
        <Card className="min-w-0">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Search className="h-5 w-5 text-primary" />
              规则列表
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <form action="/admin/training/rules" className="mb-4 flex gap-2">
              <div className="relative min-w-0 flex-1">
                <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" aria-hidden="true" />
                <input className="h-10 w-full rounded-md border bg-white pl-9 pr-3 text-sm" defaultValue={params.q ?? ""} name="q" placeholder="搜索仪器、规则说明、状态..." />
              </div>
              <button className="inline-flex h-10 items-center justify-center rounded-md bg-primary px-4 text-sm font-bold text-white" type="submit">
                筛选
              </button>
            </form>
            {visibleRules.map((item) => (
              <div className="rounded-lg border p-4" key={item.id}>
                <div className="flex flex-col justify-between gap-3 sm:flex-row sm:items-start">
                  <div className="min-w-0">
                    <p className="break-words font-semibold text-slate-900">{item.instrumentName || "未设置仪器"}</p>
                    <p className="mt-1 text-xs text-slate-500">最低分 {item.minScore.toFixed(1)}</p>
                  </div>
                  <span className={`w-fit rounded px-2 py-1 text-xs font-bold ${item.status === "active" ? "bg-emerald-50 text-emerald-700" : "bg-slate-100 text-slate-600"}`}>
                    {item.status}
                  </span>
                </div>
                <div className="mt-4 grid gap-3 text-sm md:grid-cols-2">
                  <InfoItem label="培训" value={item.requireTraining ? "需要" : "不需要"} />
                  <InfoItem label="考试" value={item.requireExam ? "需要" : "不需要"} />
                  <InfoItem label="审批" value={item.requireApproval ? "需要" : "不需要"} />
                  <InfoItem label="更新时间" value={formatDateTime(item.updatedAt)} />
                </div>
                <p className="mt-3 break-words text-sm leading-6 text-slate-600">{item.notes || "暂无规则说明。"}</p>
              </div>
            ))}
            {visibleRules.length === 0 ? <p className="rounded-lg border border-dashed p-4 text-sm text-slate-500">暂无准入规则。</p> : null}
          </CardContent>
        </Card>

        <aside className="min-w-0 space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <ShieldCheck className="h-5 w-5 text-primary" />
                新建规则
              </CardTitle>
            </CardHeader>
            <CardContent>
              <TrainingRuleForm instruments={instruments} />
            </CardContent>
          </Card>
        </aside>
      </div>
    </AdminShell>
  );
}

function Metric({ label, value }: { label: string; value: number | string }) {
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

function formatDateTime(value: string) {
  return new Date(value).toLocaleString("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    timeZone: "Asia/Shanghai",
  });
}
