import { AdminSettingsNav } from "@/components/admin-settings-nav";
import { AdminShell, requireAdminSection } from "@/components/admin-shell";
import { CopySettingsForm } from "@/components/copy-settings-form";
import { api, createDefaultCopySettings } from "@/lib/api";

export default async function AdminCopySettingsPage() {
  const currentUser = await requireAdminSection("settings");
  const settings = await api.copySettings().catch(() => createDefaultCopySettings());

  return (
    <AdminShell active="settings" title="文案中心" description="维护前台导航、首页入口、按钮、标题、占位符等可配置文案。">
      <AdminSettingsNav active="copy" role={currentUser.role} />
      <CopySettingsForm settings={settings} />
    </AdminShell>
  );
}
