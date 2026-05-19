import { AdminSettingsNav } from "@/components/admin-settings-nav";
import { AdminShell, requireAdminSection } from "@/components/admin-shell";
import { FooterSettingsForm } from "@/components/footer-settings-form";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api, createDefaultFooterSettings } from "@/lib/api";

export default async function AdminFooterSettingsPage() {
  const currentUser = await requireAdminSection("settings");
  const footerSettings = await api.footerSettings().catch(() => createDefaultFooterSettings());

  return (
    <AdminShell active="settings" title="系统基础配置" description="维护网站域名、全站底部品牌、栏目、版权信息，并作为后续 Logo 和系统名称配置入口。">
      <AdminSettingsNav active="footer" role={currentUser.role} />
      <Card>
        <CardHeader>
          <CardTitle>网站域名与 Footer 页面自定义</CardTitle>
        </CardHeader>
        <CardContent>
          <FooterSettingsForm settings={footerSettings} />
        </CardContent>
      </Card>
    </AdminShell>
  );
}
