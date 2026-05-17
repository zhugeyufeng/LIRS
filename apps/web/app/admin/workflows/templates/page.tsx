import { AdminShell, requireAdminSection } from "@/components/admin-shell";
import { BusinessConfigList } from "@/components/business-config-list";
import { WorkflowTabs } from "@/components/workflow-tabs";
import { api } from "@/lib/api";

export default async function WorkflowTemplatesPage() {
  await requireAdminSection("workflows");
  const items = await api.workflowConfigs("templates").catch(() => []);

  return (
    <AdminShell active="workflows" title="审批流配置" description="按业务类型配置预约、申领、申购、维修和授权等流程模板。">
      <WorkflowTabs active="templates" />
      <BusinessConfigList
        createTitle="流程模板"
        description="流程模板用于定义节点顺序、超时时间和处理角色，例如课题组负责人、仪器管理员、试剂管理员或财务管理员。"
        items={items}
        path="/api/workflows/templates"
        title="流程模板"
      />
    </AdminShell>
  );
}
