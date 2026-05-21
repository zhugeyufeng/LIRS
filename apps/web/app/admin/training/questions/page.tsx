import { BookOpen, Search } from "lucide-react";
import { AdminShell, requireAdminSection } from "@/components/admin-shell";
import { TrainingQuestionForm } from "@/components/extension-forms";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";
import { trainingQuestionStatusLabel, trainingQuestionTypeLabel } from "@/lib/status-labels";

export default async function AdminTrainingQuestionsPage({
  searchParams,
}: {
  searchParams?: Promise<{ q?: string }>;
}) {
  const currentUser = await requireAdminSection("trainingQuestions");
  const params = (await searchParams) ?? {};
  const query = (params.q ?? "").trim().toLowerCase();
  const [questions] = await Promise.all([api.trainingQuestions().catch(() => [])]);
  const visibleQuestions = questions.filter((item) => [item.title, item.questionType, item.status, item.options, item.correctAnswer, item.explanation].some((value) => value.toLowerCase().includes(query)));
  const activeCount = questions.filter((item) => item.status === "active").length;

  return (
    <AdminShell active="trainingQuestions" title="题库管理" description="维护单选、多选、判断和简答题题库。">
      <div className="mb-6 grid gap-4 sm:grid-cols-3">
        <Metric label="题目总数" value={questions.length} />
        <Metric label="启用题目" value={activeCount} />
        <Metric label="维护人" value={currentUser.name} />
      </div>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_400px]">
        <Card className="min-w-0">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Search className="h-5 w-5 text-primary" />
              题目列表
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <form action="/admin/training/questions" className="mb-4 flex gap-2">
              <div className="relative min-w-0 flex-1">
                <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" aria-hidden="true" />
                <input className="h-10 w-full rounded-md border bg-white pl-9 pr-3 text-sm" defaultValue={params.q ?? ""} name="q" placeholder="搜索题目、题型、答案、解析..." />
              </div>
              <button className="inline-flex h-10 items-center justify-center rounded-md bg-primary px-4 text-sm font-bold text-white" type="submit">
                筛选
              </button>
            </form>
            {visibleQuestions.map((question) => (
              <div className="rounded-lg border p-4" key={question.id}>
                <div className="flex flex-col justify-between gap-3 sm:flex-row sm:items-start">
                  <div className="min-w-0">
                    <p className="break-words font-semibold text-slate-900">{question.title}</p>
                    <p className="mt-1 text-xs text-slate-500">{trainingQuestionTypeLabel(question.questionType)}</p>
                  </div>
                  <span className="w-fit rounded bg-slate-100 px-2 py-1 text-xs font-bold text-slate-600">{trainingQuestionStatusLabel(question.status)}</span>
                </div>
                <div className="mt-4 grid gap-3 text-sm md:grid-cols-2">
                  <InfoItem label="正确答案" value={question.correctAnswer || "未填写"} />
                  <InfoItem label="更新时间" value={formatDateTime(question.updatedAt)} />
                </div>
                {question.options ? <p className="mt-3 whitespace-pre-line break-words text-sm leading-6 text-slate-600">{question.options}</p> : null}
                {question.explanation ? <p className="mt-2 break-words text-xs leading-6 text-slate-500">{question.explanation}</p> : null}
              </div>
            ))}
            {visibleQuestions.length === 0 ? <p className="rounded-lg border border-dashed p-4 text-sm text-slate-500">暂无题目。</p> : null}
          </CardContent>
        </Card>

        <aside className="min-w-0 space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <BookOpen className="h-5 w-5 text-primary" />
                新建题目
              </CardTitle>
            </CardHeader>
            <CardContent>
              <TrainingQuestionForm actorName={currentUser.name} />
            </CardContent>
          </Card>
        </aside>
      </div>
    </AdminShell>
  );
}

function Metric({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="rounded-lg border bg-white p-4">
      <p className="text-sm text-slate-500">{label}</p>
      <p className="mt-2 break-words text-2xl font-bold">{value}</p>
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
