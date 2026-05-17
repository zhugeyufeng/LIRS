import Link from "next/link";
import { Bell, Megaphone, Send } from "lucide-react";
import { AdminShell, requireAdminSection } from "@/components/admin-shell";
import { AnnouncementForm } from "@/components/notification-actions";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api, Notification } from "@/lib/api";

export default async function AdminNotificationsPage() {
  await requireAdminSection("notifications");
  const [notifications, organizationUnits] = await Promise.all([
    api.notifications().catch(() => []),
    api.organizationUnits().catch(() => []),
  ]);
  const departments = organizationUnits.filter((item) => item.kind === "department").map((item) => item.name);
  const unread = notifications.filter((item) => !item.read).length;

  return (
    <AdminShell active="notifications" title="通知管理" description="管理员在这里发布全局、部门或个人公告；消息中心只保留用户通知展示。">
      <div className="mb-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <Metric label="全部通知" value={notifications.length} />
        <Metric label="未读通知" value={unread} />
        <Metric label="部门/实验室" value={departments.length} />
        <Metric label="个人通知" value={notifications.filter((item) => item.targetScope === "personal").length} />
      </div>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_400px]">
        <Card className="min-w-0">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Bell className="h-5 w-5 text-primary" aria-hidden="true" />
              最近通知
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {notifications.length === 0 ? <p className="rounded-lg border p-4 text-sm text-slate-500">暂无通知。</p> : null}
            {notifications.slice(0, 20).map((item) => (
              <NotificationCard item={item} key={item.id} />
            ))}
          </CardContent>
        </Card>

        <aside className="min-w-0 space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Megaphone className="h-5 w-5 text-primary" aria-hidden="true" />
                发布公告
              </CardTitle>
            </CardHeader>
            <CardContent>
              <AnnouncementForm departments={departments} />
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Send className="h-5 w-5 text-primary" aria-hidden="true" />
                消息中心
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-3 text-sm leading-6 text-slate-600">
              <p>普通用户和管理员都在消息中心查看自己的通知，发布入口已经迁移到当前管理页面。</p>
              <Link className="inline-flex h-10 w-full items-center justify-center rounded-md border px-4 text-sm font-bold text-slate-700 hover:bg-slate-50" href="/notifications">
                查看消息中心
              </Link>
            </CardContent>
          </Card>
        </aside>
      </div>
    </AdminShell>
  );
}

function NotificationCard({ item }: { item: Notification }) {
  return (
    <div className={`rounded-lg border p-4 ${item.read ? "bg-white" : "border-primary/20 bg-primary/5"}`}>
      <div className="flex flex-col justify-between gap-3 sm:flex-row sm:items-start">
        <div className="min-w-0">
          <p className="break-words font-bold text-slate-900">{item.title}</p>
          <p className="mt-1 flex flex-wrap gap-x-2 gap-y-1 text-xs text-slate-500">
            <span>{formatDateTime(item.createdAt)}</span>
            <span>{levelLabel(item.level)}</span>
            <span>{scopeLabel(item)}</span>
          </p>
        </div>
        <span className={`w-fit shrink-0 rounded px-2 py-1 text-xs font-bold ${item.read ? "bg-slate-100 text-slate-600" : "bg-primary text-white"}`}>
          {item.read ? "已读" : "未读"}
        </span>
      </div>
      <p className="mt-3 break-words text-sm leading-6 text-slate-600">{item.body}</p>
    </div>
  );
}

function Metric({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="rounded-lg border bg-white p-4">
      <p className="text-sm text-slate-500">{label}</p>
      <p className="mt-2 text-2xl font-bold">{value}</p>
    </div>
  );
}

function scopeLabel(item: Notification) {
  if (item.targetScope === "department") {
    return item.department ? `部门：${item.department}` : "部门";
  }
  if (item.targetScope === "group") {
    return item.groupName ? `团队：${item.groupName}` : "团队";
  }
  if (item.targetScope === "personal") {
    return "个人";
  }
  return "全局";
}

function levelLabel(level: string) {
  const labels: Record<string, string> = {
    info: "普通",
    warning: "提醒",
    success: "成功",
  };
  return labels[level] ?? level;
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
