import { FlaskConical, Search } from "lucide-react";
import { AdminShell, requireAdminSection } from "@/components/admin-shell";
import { TrainingPracticalForm } from "@/components/extension-forms";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";
import { trainingPracticalResultLabel } from "@/lib/status-labels";

export default async function AdminTrainingPracticalPage({
  searchParams,
}: {
  searchParams?: Promise<{ q?: string }>;
}) {
  const currentUser = await requireAdminSection("trainingPractical");
  const params = (await searchParams) ?? {};
  const query = (params.q ?? "").trim().toLowerCase();
  const [practicals, instruments] = await Promise.all([api.trainingPracticals().catch(() => []), api.instruments()]);
  const visiblePracticals = practicals.filter((item) => [item.userName, item.instrumentName ?? "", item.assessor, item.result, item.notes].some((value) => value.toLowerCase().includes(query)));
  const passCount = practicals.filter((item) => item.result === "pass").length;
  const pendingCount = practicals.filter((item) => item.result === "pending").length;

  return (
    <AdminShell active="trainingPractical" title="线下考核" description="记录仪器实操考核、评分、考核人和结果状态。">
      <div className="mb-6 grid gap-4 sm:grid-cols-3">
        <Metric label="考核总数" value={practicals.length} />
        <Metric label="通过数量" value={passCount} />
        <Metric label="待确认" value={pendingCount} />
      </div>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_400px]">
        <Card className="min-w-0">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Search className="h-5 w-5 text-primary" />
              考核记录
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <form action="/admin/training/practical" className="mb-4 flex gap-2">
              <div className="relative min-w-0 flex-1">
                <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" aria-hidden="true" />
                <input className="h-10 w-full rounded-md border bg-white pl-9 pr-3 text-sm" defaultValue={params.q ?? ""} name="q" placeholder="搜索人员、仪器、考核人、结果..." />
              </div>
              <button className="inline-flex h-10 items-center justify-center rounded-md bg-primary px-4 text-sm font-bold text-white" type="submit">
                筛选
              </button>
            </form>
            {visiblePracticals.map((item) => (
              <div className="rounded-lg border p-4" key={item.id}>
                <div className="flex flex-col justify-between gap-3 sm:flex-row sm:items-start">
                  <div className="min-w-0">
                    <p className="break-words font-semibold text-slate-900">{item.userName}</p>
                    <p className="mt-1 break-words text-xs text-slate-500">{item.instrumentName || "未关联仪器"}</p>
                  </div>
                  <span className={`w-fit rounded px-2 py-1 text-xs font-bold ${item.result === "pass" ? "bg-emerald-50 text-emerald-700" : item.result === "fail" ? "bg-rose-50 text-rose-700" : "bg-amber-50 text-amber-700"}`}>
                    {trainingPracticalResultLabel(item.result)}
                  </span>
                </div>
                <div className="mt-4 grid gap-3 text-sm md:grid-cols-2">
                  <InfoItem label="考核人" value={item.assessor || "未填写"} />
                  <InfoItem label="时间" value={formatDateTime(item.assessmentAt)} />
                  <InfoItem label="得分" value={`${item.score.toFixed(1)} 分`} />
                  <InfoItem label="更新时间" value={formatDateTime(item.updatedAt)} />
                </div>
                <p className="mt-3 break-words text-sm leading-6 text-slate-600">{item.notes || "暂无备注。"}</p>
              </div>
            ))}
            {visiblePracticals.length === 0 ? <p className="rounded-lg border border-dashed p-4 text-sm text-slate-500">暂无考核记录。</p> : null}
          </CardContent>
        </Card>

        <aside className="min-w-0 space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <FlaskConical className="h-5 w-5 text-primary" />
                新建考核
              </CardTitle>
            </CardHeader>
            <CardContent>
              <TrainingPracticalForm actorName={currentUser.name} instruments={instruments} />
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
