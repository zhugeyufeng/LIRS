import Link from "next/link";
import { GraduationCap, Search } from "lucide-react";
import { AppShell } from "@/components/app-shell";
import { TrainingCourseForm } from "@/components/extension-forms";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";
import { isTenantAdminRole } from "@/lib/permissions";

export default async function TrainingCoursesPage({
  searchParams,
}: {
  searchParams?: Promise<{ q?: string }>;
}) {
  const params = (await searchParams) ?? {};
  const query = (params.q ?? "").trim().toLowerCase();
  const [courses, instruments, currentUser] = await Promise.all([api.trainingCourses(), api.instruments(), api.me()]);
  const visibleCourses = courses.filter((course) => [course.title, course.category, course.instructor, course.description, course.instrumentName ?? ""].some((value) => value.toLowerCase().includes(query)));
  const activeCount = courses.filter((item) => item.status === "active").length;
  const requiredCount = courses.filter((item) => item.requiredForBooking).length;

  return (
    <AppShell currentUser={currentUser}>
      <div className="mb-6 flex flex-col justify-between gap-4 xl:flex-row xl:items-end">
        <div className="min-w-0">
          <h1 className="text-2xl font-bold">培训课程</h1>
          <p className="mt-1 text-sm text-muted-foreground">按课程、仪器和授权要求管理培训内容。</p>
        </div>
        <form action="/training/courses" className="flex w-full max-w-xl gap-2 xl:w-auto">
          <div className="relative min-w-0 flex-1">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" aria-hidden="true" />
            <input className="h-10 w-full rounded-md border bg-white pl-9 pr-3 text-sm" defaultValue={params.q ?? ""} name="q" placeholder="搜索课程、讲师、仪器..." />
          </div>
          <button className="inline-flex h-10 items-center justify-center rounded-md bg-primary px-4 text-sm font-bold text-white" type="submit">
            筛选
          </button>
        </form>
      </div>

      <div className="mb-6 grid gap-4 sm:grid-cols-3">
        <Metric label="课程总数" value={courses.length} />
        <Metric label="启用课程" value={activeCount} />
        <Metric label="准入必修" value={requiredCount} />
      </div>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_360px]">
        <Card>
          <CardHeader>
            <CardTitle>课程列表</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {visibleCourses.map((course) => (
              <div className="rounded-lg border p-4" key={course.id}>
                <div className="flex flex-col justify-between gap-3 sm:flex-row sm:items-start">
                  <div className="min-w-0">
                    <Link className="break-words font-semibold text-slate-900 hover:text-primary" href={`/training/courses/${course.id}`}>
                      {course.title}
                    </Link>
                    <p className="mt-1 break-words text-xs text-slate-500">
                      {course.category}
                      {course.instrumentName ? ` / ${course.instrumentName}` : ""}
                    </p>
                  </div>
                  <span className="w-fit rounded bg-slate-100 px-2 py-1 text-xs font-bold text-slate-700">{course.status}</span>
                </div>
                <div className="mt-4 grid gap-3 text-sm md:grid-cols-2">
                  <InfoItem label="讲师" value={course.instructor} />
                  <InfoItem label="授课方式" value={deliveryLabel(course.deliveryMode)} />
                  <InfoItem label="时长" value={`${course.durationHours} 小时`} />
                  <InfoItem label="必修" value={course.requiredForBooking ? "是" : "否"} />
                </div>
                <p className="mt-3 break-words text-sm leading-6 text-slate-600">{course.description || "未填写课程说明"}</p>
                <div className="mt-4">
                  <Link className="inline-flex h-9 items-center justify-center rounded-md border px-3 text-sm font-medium text-slate-700 hover:bg-slate-50" href={`/training/courses/${course.id}`}>
                    查看详情
                  </Link>
                </div>
              </div>
            ))}
            {visibleCourses.length === 0 ? <p className="rounded-lg border border-dashed p-4 text-sm text-slate-500">暂无课程。</p> : null}
          </CardContent>
        </Card>

        <aside className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <GraduationCap className="h-5 w-5 text-primary" />
                新建课程
              </CardTitle>
            </CardHeader>
            <CardContent>
              {isTenantAdminRole(currentUser.role) ? (
                <TrainingCourseForm actorName={currentUser.name} instruments={instruments} />
              ) : (
                <p className="text-sm leading-6 text-slate-500">当前账号没有课程维护权限，只能查看课程列表。</p>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>课程提醒</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3 text-sm leading-6 text-slate-600">
              <p>课程可关联具体仪器，也可以作为通用安全培训使用。</p>
              <p>已启用的课程可作为预约前置条件。</p>
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

function deliveryLabel(value: string) {
  const labels: Record<string, string> = {
    online: "线上",
    offline: "线下",
    blended: "混合",
  };
  return labels[value] ?? value;
}
