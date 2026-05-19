import { redirect } from "next/navigation";
import { ProfileSettingsForm } from "@/components/account-settings-forms";
import { SettingsShell } from "@/components/settings-shell";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";

export default async function ProfileSettingsPage() {
  const [currentUser, announcements] = await Promise.all([
    api.me().catch(() => null),
    api.notifications(undefined, "announcement").catch(() => []),
  ]);
  if (!currentUser) {
    redirect("/login");
  }

  return (
    <SettingsShell active="profile" currentUser={currentUser} title="个人资料" description="查看当前账号身份信息，姓名、手机号和所属部门由管理员维护。">
      <Card>
        <CardHeader>
          <CardTitle>个人资料</CardTitle>
        </CardHeader>
        <CardContent>
          <ProfileSettingsForm user={currentUser} />
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>管理员公告</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {announcements.length === 0 ? <p className="rounded-lg border p-4 text-sm text-slate-500">暂无管理员公告。</p> : null}
          {announcements.slice(0, 20).map((item) => (
            <div className="rounded-lg border bg-white p-4" key={item.id}>
              <div className="flex flex-col justify-between gap-2 sm:flex-row sm:items-start">
                <div className="min-w-0">
                  <p className="break-words font-bold text-slate-900">{item.title}</p>
                  <p className="mt-1 text-xs text-slate-500">发布人：{item.publisher || "管理员"}</p>
                </div>
                <span className="shrink-0 text-xs text-slate-500">{new Date(item.createdAt).toLocaleString("zh-CN")}</span>
              </div>
              <p className="mt-3 whitespace-pre-wrap break-words text-sm leading-6 text-slate-700">{item.body}</p>
            </div>
          ))}
        </CardContent>
      </Card>
    </SettingsShell>
  );
}
