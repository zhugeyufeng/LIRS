import { ClipboardCheck, Search, ShieldCheck } from "lucide-react";
import { AppShell } from "@/components/app-shell";
import { TrainingExamForm } from "@/components/extension-forms";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";
import { trainingExamStatusLabel, trainingQuestionTypeLabel } from "@/lib/status-labels";

export default async function TrainingExamsPage({
  searchParams,
}: {
  searchParams?: Promise<{ q?: string }>;
}) {
  const params = (await searchParams) ?? {};
  const query = (params.q ?? "").trim().toLowerCase();
  const currentUser = await api.me();
  const [exams, courses, questions] = await Promise.all([api.trainingExams(), api.trainingCourses(), api.trainingQuestions().catch(() => [])]);
  const visibleExams = exams.filter((item) => [item.userName, item.courseTitle, item.status, item.answers, item.notes].some((value) => String(value ?? "").toLowerCase().includes(query)));
  const passedCount = visibleExams.filter((item) => item.passed).length;

  return (
    <AppShell currentUser={currentUser}>
      <div className="mb-6 flex flex-col justify-between gap-4 xl:flex-row xl:items-end">
        <div className="min-w-0">
          <h1 className="text-2xl font-bold">在线考试</h1>
          <p className="mt-1 text-sm text-muted-foreground">查看题库、提交考试记录并追踪考试状态。</p>
        </div>
        <form action="/training/exams" className="flex w-full max-w-xl gap-2 xl:w-auto">
          <div className="relative min-w-0 flex-1">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" aria-hidden="true" />
            <input className="h-10 w-full rounded-md border bg-white pl-9 pr-3 text-sm" defaultValue={params.q ?? ""} name="q" placeholder="搜索考试记录、课程、答题内容..." />
          </div>
          <button className="inline-flex h-10 items-center justify-center rounded-md bg-primary px-4 text-sm font-bold text-white" type="submit">
            筛选
          </button>
        </form>
      </div>

      <div className="mb-6 grid gap-4 sm:grid-cols-3">
        <Metric label="考试记录" value={visibleExams.length} />
        <Metric label="已通过" value={passedCount} />
        <Metric label="题库数量" value={questions.length} />
      </div>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_400px]">
        <Card className="min-w-0">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <ClipboardCheck className="h-5 w-5 text-primary" />
              考试记录
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {visibleExams.map((exam) => (
              <div className="rounded-lg border p-4" key={exam.id}>
                <div className="flex flex-col justify-between gap-3 sm:flex-row sm:items-start">
                  <div className="min-w-0">
                    <p className="break-words font-semibold text-slate-900">{exam.userName}</p>
                    <p className="mt-1 break-words text-xs text-slate-500">{exam.courseTitle || "未关联课程"}</p>
                  </div>
                  <span className={`w-fit rounded px-2 py-1 text-xs font-bold ${exam.passed ? "bg-emerald-50 text-emerald-700" : "bg-amber-50 text-amber-700"}`}>
                    {exam.passed ? "通过" : "未通过"}
                  </span>
                </div>
                <div className="mt-4 grid gap-3 text-sm md:grid-cols-2">
                  <InfoItem label="得分" value={`${exam.score.toFixed(1)} 分`} />
                  <InfoItem label="状态" value={trainingExamStatusLabel(exam.status)} />
                  <InfoItem label="考试时间" value={formatDateTime(exam.examAt)} />
                  <InfoItem label="关联课程" value={exam.courseTitle || "未关联"} />
                </div>
                <p className="mt-3 break-words text-sm leading-6 text-slate-600">{exam.notes || "暂无备注。"}</p>
                {exam.answers ? <p className="mt-2 break-words text-xs leading-6 text-slate-500">答题记录：{exam.answers}</p> : null}
              </div>
            ))}
            {visibleExams.length === 0 ? <p className="rounded-lg border border-dashed p-4 text-sm text-slate-500">暂无考试记录。</p> : null}
          </CardContent>
        </Card>

        <aside className="min-w-0 space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <ShieldCheck className="h-5 w-5 text-primary" />
                提交考试
              </CardTitle>
            </CardHeader>
            <CardContent>
              <TrainingExamForm actorName={currentUser.name} courses={courses} />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>题库摘要</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              {questions.slice(0, 6).map((question) => (
                <div className="rounded-lg border p-3 text-sm" key={question.id}>
                  <p className="break-words font-semibold text-slate-900">{question.title}</p>
                  <p className="mt-1 text-xs text-slate-500">{trainingQuestionTypeLabel(question.questionType)}</p>
                  {question.options ? <p className="mt-2 whitespace-pre-line text-xs leading-6 text-slate-500">{question.options}</p> : null}
                </div>
              ))}
              {questions.length === 0 ? <p className="text-sm text-slate-500">暂无题库数据。</p> : null}
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

function formatDateTime(value: string) {
  return new Date(value).toLocaleString("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    timeZone: "Asia/Shanghai",
  });
}
