import { redirect } from "next/navigation";
import { AdminSettingsNav } from "@/components/admin-settings-nav";
import { AdminShell, requireAdminSection } from "@/components/admin-shell";
import { AccessControlSettingsForm } from "@/components/access-control-settings-form";
import { api, createDefaultAccessControlSettings } from "@/lib/api";

export default async function AccessControlSettingsPage() {
  const currentUser = await requireAdminSection("settings");
  if (currentUser.role !== "super_admin") {
    redirect("/admin/settings");
  }
  const settings = await api.accessControlSettings().catch(() => createDefaultAccessControlSettings());

  return (
    <AdminShell active="settings" title="第三方集成" description="维护大华、海康威视门禁对接参数；具体仪器对应的授权组和点位在仪器管理中逐台设置。">
      <AdminSettingsNav active="access-control" role={currentUser.role} />
      <AccessControlSettingsForm settings={settings} />
    </AdminShell>
  );
}
