import Link from "next/link";
import { notFound } from "next/navigation";
import { BookOpen, ClipboardCheck, ShieldCheck } from "lucide-react";
import { AppShell } from "@/components/app-shell";
import { TrainingAuthorizationForm } from "@/components/extension-forms";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";
import { trainingAuthorizationStatusLabel, trainingCourseStatusLabel, trainingDeliveryModeLabel, trainingExamStatusLabel } from "@/lib/status-labels";

export default async function TrainingCourseDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const currentUser = await api.me();
  const [courses, authorizations, exams, instruments] = await Promise.all([
    api.trainingCourses(),
    api.trainingAuthorizations().catch(() => []),
    api.trainingExams().catch(() => []),
    api.instruments().catch(() => []),
  ]);
  const course = courses.find((item) => item.id === id);
  if (!course) {
    notFound();
  }
  const courseAuthorizations = authorizations.filter((item) => item.courseId === id);
  const courseExams = exams.filter((item) => item.courseId === id);
  const myAuthorization = courseAuthorizations.find((item) => item.userId === currentUser.id || item.userName === currentUser.name);
  const passedExams = courseExams.filter((item) => item.passed).length;

  return (
    <AppShell currentUser={currentUser}>
      <div className="mb-6 flex flex-col justify-between gap-4 lg:flex-row lg:items-end">
        <div className="min-w-0">
          <p className="text-xs font-bold uppercase tracking-widest text-primary">培训课程</p>
          <h1 className="mt-2 break-words text-2xl font-bold sm:text-3xl">{course.title}</h1>
          <p className="mt-2 max-w-3xl break-words text-sm leading-6 text-muted-foreground">{course.description || "暂无课程说明。"}</p>
        </div>
        <Link className="inline-flex h-10 items-center justify-center rounded-md border px-4 text-sm font-bold text-slate-700 hover:bg-slate-50" href="/training/courses">
          返回课程列表
        </Link>
      </div>

      <div className="mb-6 grid gap-4 sm:grid-cols-3">
        <Metric label="课程状态" value={trainingCourseStatusLabel(course.status)} />
        <Metric label="已通过考试" value={passedExams} />
        <Metric label="授权记录" value={courseAuthorizations.length} />
      </div>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_360px]">
        <div className="min-w-0 space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <BookOpen className="h-5 w-5 text-primary" aria-hidden="true" />
                课程信息
              </CardTitle>
            </CardHeader>
            <CardContent className="grid gap-4 sm:grid-cols-2">
              <Info label="课程分类" value={course.category} />
              <Info label="关联仪器" value={course.instrumentName || "通用培训"} />
              <Info label="授课方式" value={trainingDeliveryModeLabel(course.deliveryMode)} />
              <Info label="讲师/负责人" value={course.instructor || "未填写"} />
              <Info label="课程时长" value={`${course.durationHours} 小时`} />
              <Info label="预约必修" value={course.requiredForBooking ? "是" : "否"} />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <ClipboardCheck className="h-5 w-5 text-primary" aria-hidden="true" />
                我的学习与考试
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              {courseExams.map((exam) => (
                <div className="rounded-lg border p-4" key={exam.id}>
                  <div className="flex flex-col justify-between gap-2 sm:flex-row sm:items-start">
                    <div>
                      <p className="font-bold text-slate-900">{exam.userName}</p>
                      <p className="mt-1 text-xs text-slate-500">{formatDateTime(exam.examAt)}</p>
                    </div>
                    <span className={exam.passed ? "w-fit rounded-full bg-emerald-50 px-2 py-1 text-xs font-bold text-emerald-700" : "w-fit rounded-full bg-amber-50 px-2 py-1 text-xs font-bold text-amber-700"}>
                      {exam.passed ? "通过" : "未通过"}
                    </span>
                  </div>
                  <p className="mt-3 text-sm text-slate-600">得分：{exam.score.toFixed(1)} / 状态：{trainingExamStatusLabel(exam.status)}</p>
                </div>
              ))}
              {courseExams.length === 0 ? <p className="rounded-lg border border-dashed p-4 text-sm text-slate-500">暂无考试记录。</p> : null}
            </CardContent>
          </Card>
        </div>

        <aside className="min-w-0 space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <ShieldCheck className="h-5 w-5 text-primary" aria-hidden="true" />
                授权状态
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              {myAuthorization ? (
                <div className="rounded-lg border bg-slate-50/50 p-4 text-sm">
                  <p className="font-bold text-slate-900">{trainingAuthorizationStatusLabel(myAuthorization.status)}</p>
                  <p className="mt-2 text-slate-600">到期：{formatDateTime(myAuthorization.expiresAt)}</p>
                  <p className="mt-2 break-words text-slate-500">{myAuthorization.notes || "暂无备注。"}</p>
                </div>
              ) : (
                <p className="text-sm leading-6 text-slate-600">你还没有这门课程的授权记录，可以在这里提交申请。</p>
              )}
              <TrainingAuthorizationForm actorName={currentUser.name} courses={[course]} instruments={instruments} />
            </CardContent>
          </Card>
        </aside>
      </div>
    </AppShell>
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

function Info({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border bg-slate-50/40 p-4">
      <p className="text-xs text-slate-500">{label}</p>
      <p className="mt-1 break-words font-bold text-slate-900">{value}</p>
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
