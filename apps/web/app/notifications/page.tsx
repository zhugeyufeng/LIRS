import { AppShell } from "@/components/app-shell";
import { MarkAllNotificationsRead, MarkNotificationRead } from "@/components/notification-actions";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";

export default async function NotificationsPage() {
  const notifications = await api.notifications();
  const unreadCount = notifications.filter((item) => !item.read).length;

  return (
    <AppShell mainClassName="mx-auto w-full max-w-[88rem] px-4 pt-6 pb-4 sm:px-6 sm:pt-8 sm:pb-4 lg:px-8">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-slate-900 sm:text-3xl">消息中心</h1>
        <p className="mt-2 text-sm text-muted-foreground">查看系统通知、审批提醒、预约提醒、库存预警和管理员公告。</p>
      </div>
      <Card className="min-w-0 overflow-hidden">
        <CardHeader className="p-4 sm:p-6">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <CardTitle>我的通知</CardTitle>
            <MarkAllNotificationsRead disabled={unreadCount === 0} />
          </div>
          <p className="mt-2 text-sm text-muted-foreground">未读 {unreadCount} 条</p>
        </CardHeader>
        <CardContent className="space-y-3 px-4 pb-4 pt-0 sm:px-6 sm:pb-6">
          {notifications.map((item) => (
            <div className={`min-w-0 rounded-lg border p-3 sm:p-4 ${item.read ? "bg-white" : "border-primary/20 bg-primary/5"}`} key={item.id}>
              <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                <div className="min-w-0 flex-1">
                  <p className="break-words font-semibold leading-6">{item.title}</p>
                  <p className="mt-1 flex flex-wrap gap-x-2 gap-y-1 text-xs text-slate-500">
                    <span>{formatDateTime(item.createdAt)}</span>
                    <span>{levelLabel(item.level)}</span>
                    <span>{sourceLabel(item.source)}</span>
                    {item.source === "announcement" ? <span>发送人：{item.publisher || "管理员"}</span> : null}
                  </p>
                </div>
                <MarkNotificationRead id={item.id} read={item.read} />
              </div>
              <p className="mt-2 break-words text-sm leading-6 text-slate-600">{item.body}</p>
            </div>
          ))}
          {notifications.length === 0 ? (
            <div className="rounded-lg border border-dashed bg-slate-50 p-6 text-center text-sm text-slate-500">
              暂无通知
            </div>
          ) : null}
        </CardContent>
      </Card>
    </AppShell>
  );
}

function levelLabel(level: string) {
  const labels: Record<string, string> = {
    info: "普通",
    warning: "提醒",
    success: "成功",
  };
  return labels[level] ?? level;
}

function sourceLabel(source: string) {
  return source === "announcement" ? "管理员公告" : "系统通知";
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
