import { AdminSettingsNav } from "@/components/admin-settings-nav";
import { AdminShell, requireAdminSection } from "@/components/admin-shell";
import { TenantManagement } from "@/components/tenant-management";
import { api } from "@/lib/api";

export default async function TenantSettingsPage() {
  const currentUser = await requireAdminSection("settings");
  const tenants = await api.tenants();
  const visibleTenants = currentUser.role === "super_admin" ? tenants : tenants.filter((tenant) => tenant.id === currentUser.tenantId);

  return (
    <AdminShell active="settings" title="单位/机构信息" description="多单位/机构共用系统但数据按机构隔离；单位名称、机构编码、机构 ID 和财务模块状态在这里维护。">
      <AdminSettingsNav active="tenants" role={currentUser.role} />
      <TenantManagement currentUser={currentUser} tenants={visibleTenants} />
    </AdminShell>
  );
}
