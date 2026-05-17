import { AdminSettingsNav } from "@/components/admin-settings-nav";
import { AdminShell, requireAdminSection } from "@/components/admin-shell";
import { OrganizationUnitManager } from "@/components/organization-unit-management";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";

type SearchParams = {
  tenantId?: string;
};

export default async function AdminOrganizationSettingsPage({ searchParams }: { searchParams?: Promise<SearchParams> }) {
  const currentUser = await requireAdminSection("settings");
  const params = (await searchParams) ?? {};
  const tenants = currentUser.role === "super_admin" ? await api.tenants().catch(() => []) : [];
  const selectedTenantId = resolveSelectedTenantId(currentUser.tenantId, currentUser.role, tenants, params.tenantId);
  const selectedTenant = tenants.find((tenant) => tenant.id === selectedTenantId) ?? {
    id: currentUser.tenantId,
    name: currentUser.tenantName,
    code: currentUser.tenantId,
    financeEnabled: currentUser.financeEnabled,
    status: "active",
    createdAt: "",
    updatedAt: "",
  };
  const organizationUnits = await api.organizationUnits(undefined, selectedTenantId).catch(() => []);
  const departments = organizationUnits.filter((item) => item.kind === "department");
  const groups = organizationUnits.filter((item) => item.kind === "group");

  return (
    <AdminShell active="settings" title="组织架构管理" description="按机构独立维护部门、实验室和部门二级团队；仪器可直接归属部门，也可归属到部门下团队。">
      <AdminSettingsNav active="organization" role={currentUser.role} />
      <Card>
        <CardHeader>
          <CardTitle>组织基础数据</CardTitle>
        </CardHeader>
        <CardContent>
          <OrganizationUnitManager
            currentUser={currentUser}
            departments={departments}
            groups={groups}
            selectedTenantId={selectedTenantId}
            selectedTenantName={selectedTenant.name}
            tenants={tenants}
          />
        </CardContent>
      </Card>
    </AdminShell>
  );
}

function resolveSelectedTenantId(
  currentTenantId: string,
  role: string,
  tenants: { id: string }[],
  requestedTenantId?: string,
) {
  if (role !== "super_admin") {
    return currentTenantId;
  }
  const normalized = requestedTenantId?.trim() ?? "";
  if (normalized && tenants.some((tenant) => tenant.id === normalized)) {
    return normalized;
  }
  return tenants.some((tenant) => tenant.id === currentTenantId) ? currentTenantId : tenants[0]?.id ?? currentTenantId;
}
