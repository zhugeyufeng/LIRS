import { redirect } from "next/navigation";
import { AdminSettingsNav } from "@/components/admin-settings-nav";
import { AdminShell, requireAdminSection } from "@/components/admin-shell";
import { NotificationChannelSettingsForm } from "@/components/notification-channel-settings-form";
import { api } from "@/lib/api";

export default async function NotificationChannelSettingsPage() {
  const currentUser = await requireAdminSection("settings");
  if (currentUser.role !== "super_admin") {
    redirect("/admin/settings");
  }
  const settings = await api.notificationChannelSettings();

  return (
    <AdminShell active="settings" title="通知通道配置" description="配置注册邮箱验证码 Microsoft Graph 邮件，并预留微信公众号、服务号通知接口参数。">
      <AdminSettingsNav active="notifications" role={currentUser.role} />
      <NotificationChannelSettingsForm settings={settings} />
    </AdminShell>
  );
}
