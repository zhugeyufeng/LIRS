import Link from "next/link";
import { Bell, Building2, Megaphone, Send } from "lucide-react";
import { AdminShell, requireAdminSection } from "@/components/admin-shell";
import { AnnouncementForm, DeleteNotificationButton } from "@/components/notification-actions";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api, Notification, Tenant } from "@/lib/api";
import { notificationLevelLabel } from "@/lib/status-labels";
import { TenantSelector } from "./tenant-selector";

type SearchParams = {
  tenantId?: string;
};

export default async function AdminNotificationsPage({ searchParams }: { searchParams?: Promise<SearchParams> }) {
  const currentUser = await requireAdminSection("notifications");
  const params = (await searchParams) ?? {};
  const tenants = currentUser.role === "super_admin" ? await api.tenants().catch(() => []) : [];
  const selectedTenantId = resolveSelectedTenantId(currentUser.tenantId, currentUser.role, tenants, params.tenantId);
  const selectedTenant = tenants.find((tenant) => tenant.id === selectedTenantId) ?? {
    id: currentUser.tenantId,
    name: currentUser.tenantName,
    code: currentUser.tenantCode || currentUser.tenantId,
    financeEnabled: currentUser.financeEnabled,
    status: "active",
    createdAt: "",
    updatedAt: "",
  };
  const [notifications, organizationUnits] = await Promise.all([
    api.notifications(selectedTenantId, "announcement").catch(() => []),
    api.organizationUnits(undefined, selectedTenantId).catch(() => []),
  ]);
  const departments = organizationUnits.filter((item) => item.kind === "department").map((item) => item.name);
  const groups = organizationUnits.filter((item) => item.kind === "group").map((item) => item.name);

  return (
    <AdminShell active="notifications" title="通知管理" description="管理员在这里按机构发布、修改和删除公告；系统流程通知在消息中心只读展示。">
      <div className="mb-6 space-y-4">
        {currentUser.role === "super_admin" ? (
          <TenantSelector selectedTenant={selectedTenant} tenants={tenants} />
        ) : (
          <div className="rounded-lg border bg-slate-50/60 p-4">
            <p className="text-sm font-medium text-slate-900">当前机构：{selectedTenant.name}</p>
            <p className="mt-1 text-xs text-slate-500">公告、通知和钉钉推送只作用于当前机构。</p>
          </div>
        )}
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <Metric label="全部通知" value={notifications.length} />
          <Metric label="发布公告" value={notifications.length} />
          <Metric label="部门/团队" value={`${departments.length}/${groups.length}`} />
          <Metric label="个人通知" value={notifications.filter((item) => item.targetScope === "personal").length} />
        </div>
      </div>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_400px]">
        <Card className="min-w-0">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Bell className="h-5 w-5 text-primary" aria-hidden="true" />
              已发布公告
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {notifications.length === 0 ? <p className="rounded-lg border p-4 text-sm text-slate-500">暂无通知。</p> : null}
            {notifications.slice(0, 50).map((item) => (
              <NotificationCard departments={departments} groups={groups} item={item} key={item.id} selectedTenantId={selectedTenantId} />
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
            <CardContent className="space-y-4">
              <div className="rounded-md border bg-slate-50 px-3 py-2 text-sm">
                <p className="font-medium text-slate-900">{selectedTenant.name}</p>
                <p className="mt-1 text-xs text-slate-500">发布后写入该机构通知列表，并按目标范围推送钉钉。</p>
              </div>
              <AnnouncementForm departments={departments} groups={groups} selectedTenantId={selectedTenantId} />
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
              <p>消息中心展示系统通知；管理员发布的公告会在用户个人中心展示，并标明发布人。</p>
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

function NotificationCard({
  departments,
  groups,
  item,
  selectedTenantId,
}: {
  departments: string[];
  groups: string[];
  item: Notification;
  selectedTenantId: string;
}) {
  return (
    <div className={`rounded-lg border p-4 ${item.read ? "bg-white" : "border-primary/20 bg-primary/5"}`}>
      <div className="flex flex-col justify-between gap-3 sm:flex-row sm:items-start">
        <div className="min-w-0">
          <p className="break-words font-bold text-slate-900">{item.title}</p>
          <p className="mt-1 flex flex-wrap gap-x-2 gap-y-1 text-xs text-slate-500">
        <span>{formatDateTime(item.createdAt)}</span>
        <span>{levelLabel(item.level)}</span>
        <span>{scopeLabel(item)}</span>
        <span>发布人：{item.publisher || "管理员"}</span>
            <span className="inline-flex items-center gap-1">
              <Building2 className="h-3 w-3" aria-hidden="true" />
              {item.tenantName || item.tenantId}
            </span>
          </p>
        </div>
        <span className={`w-fit shrink-0 rounded px-2 py-1 text-xs font-bold ${item.read ? "bg-slate-100 text-slate-600" : "bg-primary text-white"}`}>
          {item.read ? "已读" : "未读"}
        </span>
      </div>
      <p className="mt-3 break-words text-sm leading-6 text-slate-600">{item.body}</p>
      <div className="mt-4 flex flex-col gap-2 sm:flex-row sm:justify-end">
        <AnnouncementForm departments={departments} groups={groups} initial={item} mode="edit" selectedTenantId={selectedTenantId} />
        <DeleteNotificationButton id={item.id} selectedTenantId={selectedTenantId} title={item.title} />
      </div>
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

function resolveSelectedTenantId(currentTenantId: string, role: string, tenants: Tenant[], requestedTenantId?: string) {
  if (role !== "super_admin") {
    return currentTenantId;
  }
  const normalized = requestedTenantId?.trim() ?? "";
  if (normalized && tenants.some((tenant) => tenant.id === normalized)) {
    return normalized;
  }
  return tenants.some((tenant) => tenant.id === currentTenantId) ? currentTenantId : tenants[0]?.id ?? currentTenantId;
}

function scopeLabel(item: Notification) {
  if (item.targetScope === "department") {
    return item.department ? `部门：${item.department}` : "部门";
  }
  if (item.targetScope === "group") {
    return item.groupName ? `团队：${item.groupName}` : "团队";
  }
  if (item.targetScope === "personal") {
    return item.userId ? `个人：${item.userId}` : "个人";
  }
  return "全局";
}

function levelLabel(level: string) {
  return notificationLevelLabel(level);
}

function formatDateTime(value: string) {
  return new Date(value).toLocaleString("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}
