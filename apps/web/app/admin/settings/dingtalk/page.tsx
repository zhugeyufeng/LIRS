import { headers } from "next/headers";
import { AdminSettingsNav } from "@/components/admin-settings-nav";
import { AdminShell, requireAdminSection } from "@/components/admin-shell";
import { DingTalkSettingsForm } from "@/components/dingtalk-settings-form";
import { api, createDefaultDingTalkSettings } from "@/lib/api";

type SearchParams = {
  tenantId?: string;
};

export default async function DingTalkSettingsPage({ searchParams }: { searchParams?: Promise<SearchParams> }) {
  const currentUser = await requireAdminSection("settings");
  const params = (await searchParams) ?? {};
  const [tenants, users] = await Promise.all([
    currentUser.role === "super_admin" ? api.tenants().catch(() => []) : Promise.resolve([]),
    api.users().catch(() => []),
  ]);
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
  const settings = await api.dingTalkSettings(selectedTenantId).catch(() => createDefaultDingTalkSettings());
  const origin = await requestOrigin();

  return (
    <AdminShell active="settings" title="钉钉应用设置" description="按机构配置企业内部应用 Client ID、Client Secret、Corp ID、机器人编码、扫码绑定回调地址和 HTTP 事件订阅。">
      <AdminSettingsNav active="dingtalk" role={currentUser.role} />
      <DingTalkSettingsForm
        currentUser={currentUser}
        origin={origin}
        selectedTenant={selectedTenant}
        settings={settings}
        tenants={tenants}
        users={users.filter((user) => user.tenantId === selectedTenant.id && user.status === "active")}
      />
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

async function requestOrigin() {
  const headerStore = await headers();
  const proto = headerStore.get("x-forwarded-proto") ?? "https";
  const host = headerStore.get("x-forwarded-host") ?? headerStore.get("host") ?? "example.com";
  return `${proto.split(",")[0].trim()}://${host.split(",")[0].trim()}`;
}
