import { AdminSettingsNav } from "@/components/admin-settings-nav";
import { AdminShell, requireAdminSection } from "@/components/admin-shell";
import { TenantManagement } from "@/components/tenant-management";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";

export default async function BillingSettingsPage() {
  const currentUser = await requireAdminSection("settings");
  const tenants = await api.tenants();
  const visibleTenants = currentUser.role === "super_admin" ? tenants : tenants.filter((tenant) => tenant.id === currentUser.tenantId);

  return (
    <AdminShell active="settings" title="财务模块开关" description="按单位/机构启用或停用财务模块；停用后财务菜单和财务接口会按机构关闭。">
      <AdminSettingsNav active="billing" role={currentUser.role} />
      <Card>
        <CardHeader>
          <CardTitle>单位/机构财务配置</CardTitle>
        </CardHeader>
        <CardContent>
          <TenantManagement currentUser={currentUser} tenants={visibleTenants} />
        </CardContent>
      </Card>
    </AdminShell>
  );
}
