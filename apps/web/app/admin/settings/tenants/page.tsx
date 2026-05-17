import { AdminSettingsNav } from "@/components/admin-settings-nav";
import { AdminShell, requireAdminSection } from "@/components/admin-shell";
import { TenantManagement } from "@/components/tenant-management";
import { api } from "@/lib/api";

export default async function TenantSettingsPage() {
  const currentUser = await requireAdminSection("settings");
  const tenants = await api.tenants();
  const visibleTenants = currentUser.role === "super_admin" ? tenants : tenants.filter((tenant) => tenant.id === currentUser.tenantId);

  return (
    <AdminShell active="settings" title="租户配置" description="多机构共用系统但数据按租户隔离；机构编码创建时自动生成，机构 ID 和财务模块状态在这里维护。">
      <AdminSettingsNav active="tenants" role={currentUser.role} />
      <TenantManagement currentUser={currentUser} tenants={visibleTenants} />
    </AdminShell>
  );
}
