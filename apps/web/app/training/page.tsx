import Link from "next/link";
import { BadgeCheck, BookOpen, ShieldCheck, TimerReset } from "lucide-react";
import { AppShell } from "@/components/app-shell";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";

export default async function TrainingHomePage() {
  const [courses, authorizations, exams, currentUser] = await Promise.all([
    api.trainingCourses(),
    api.trainingAuthorizations(),
    api.trainingExams().catch(() => []),
    api.me(),
  ]);
  const activeCourses = courses.filter((item) => item.status === "active");
  const activeAuthorizations = authorizations.filter((item) => item.status === "active");
  const passedExams = exams.filter((item) => item.passed).length;
  const expiringSoon = authorizations.filter((item) => {
    const expiresAt = new Date(item.expiresAt).getTime();
    return item.status === "active" && expiresAt - Date.now() <= 30 * 24 * 60 * 60 * 1000;
  });

  return (
    <AppShell currentUser={currentUser}>
      <div className="mb-6 flex flex-col justify-between gap-4 xl:flex-row xl:items-end">
        <div className="min-w-0">
          <h1 className="text-2xl font-bold">培训与准入中心</h1>
          <p className="mt-1 text-sm text-muted-foreground">管理仪器培训、课程授权、准入有效期和到期提醒。</p>
        </div>
        <div className="flex flex-wrap gap-3">
          <Link className="inline-flex h-10 items-center justify-center rounded-md border px-4 text-sm font-bold text-slate-700 hover:bg-slate-50" href="/training/courses">
            课程列表
          </Link>
          <Link className="inline-flex h-10 items-center justify-center rounded-md bg-primary px-4 text-sm font-bold text-white" href="/training/authorizations">
            授权记录
          </Link>
        </div>
      </div>

      <div className="mb-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <Metric label="培训课程" value={courses.length} icon={BookOpen} />
        <Metric label="已启用课程" value={activeCourses.length} icon={BadgeCheck} />
        <Metric label="有效授权" value={activeAuthorizations.length} icon={ShieldCheck} />
        <Metric label="30 天内到期" value={expiringSoon.length} icon={TimerReset} />
      </div>

      <div className="mb-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
        <Metric label="考试记录" value={exams.length} icon={TimerReset} />
        <Metric label="已通过考试" value={passedExams} icon={BadgeCheck} />
        <Metric label="当前账号" value={currentUser.name} icon={ShieldCheck} />
      </div>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_360px]">
        <Card>
          <CardHeader>
            <CardTitle>流程入口</CardTitle>
          </CardHeader>
          <CardContent className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
            <Entry href="/training/courses" title="课程管理" description="维护培训课程、仪器关联和准入说明。" />
            <Entry href="/training/authorizations" title="授权记录" description="查看授权、到期时间和申请状态。" />
            <Entry href="/training/exams" title="在线考试" description="查看题库和考试记录，提交考试结果。" />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>当前账号</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 text-sm leading-6 text-slate-700">
            <p className="break-words">账号：{currentUser.name}</p>
            <p className="break-words">角色：{currentUser.role}</p>
            <p className="break-words">机构：{currentUser.tenantName}</p>
            <p className="break-words">授权提示：到期前 30 天会进入预警。</p>
          </CardContent>
        </Card>
      </div>
    </AppShell>
  );
}

function Metric({ label, value, icon: Icon }: { label: string; value: number | string; icon: typeof BookOpen }) {
  return (
    <div className="rounded-lg border bg-white p-4">
      <div className="flex items-center justify-between gap-3">
        <p className="text-sm text-slate-500">{label}</p>
        <Icon className="h-4 w-4 text-primary" aria-hidden="true" />
      </div>
      <p className="mt-2 text-2xl font-bold">{value}</p>
    </div>
  );
}

function Entry({ href, title, description }: { href: string; title: string; description: string }) {
  return (
    <Link className="rounded-lg border bg-white p-4 transition hover:border-primary/40 hover:shadow-sm" href={href}>
      <p className="font-semibold text-slate-900">{title}</p>
      <p className="mt-2 text-sm leading-6 text-slate-500">{description}</p>
    </Link>
  );
}
