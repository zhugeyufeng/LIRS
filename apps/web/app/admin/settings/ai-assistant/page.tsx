import { AdminSettingsNav } from "@/components/admin-settings-nav";
import { AdminShell, requireAdminSection } from "@/components/admin-shell";
import { AIAssistantSettingsForm } from "@/components/ai-assistant-settings-form";
import { api, createDefaultAIAssistantSettings } from "@/lib/api";

type SearchParams = {
  tenantId?: string;
};

export default async function AdminAIAssistantSettingsPage({ searchParams }: { searchParams?: Promise<SearchParams> }) {
  const currentUser = await requireAdminSection("settings");
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
  const settings = await api.aiAssistantSettings(selectedTenantId).catch(() => createDefaultAIAssistantSettings());

  return (
    <AdminShell active="settings" title="AI 模型设置" description="按机构配置 AI 助手使用的 OpenAI 兼容模型 API、模型名称、密钥、温度和系统提示词。">
      <AdminSettingsNav active="ai-assistant" role={currentUser.role} />
      <AIAssistantSettingsForm currentUser={currentUser} selectedTenant={selectedTenant} settings={settings} tenants={tenants} />
    </AdminShell>
  );
}

function resolveSelectedTenantId(currentTenantId: string, role: string, tenants: { id: string }[], requestedTenantId?: string) {
  if (role !== "super_admin") {
    return currentTenantId;
  }
  const normalized = requestedTenantId?.trim() ?? "";
  if (normalized && tenants.some((tenant) => tenant.id === normalized)) {
    return normalized;
  }
  return tenants.some((tenant) => tenant.id === currentTenantId) ? currentTenantId : tenants[0]?.id ?? currentTenantId;
}
