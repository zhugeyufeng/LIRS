import { AdminShell, requireAdminSection } from "@/components/admin-shell";
import { BusinessConfigList } from "@/components/business-config-list";
import { WorkflowTabs } from "@/components/workflow-tabs";
import { api } from "@/lib/api";

export default async function WorkflowApproversPage() {
  await requireAdminSection("workflows");
  const items = await api.workflowConfigs("approvers").catch(() => []);

  return (
    <AdminShell active="workflows" title="审批人配置" description="配置课题组负责人、仪器管理员、试剂管理员和财务管理员等节点处理人。">
      <WorkflowTabs active="approvers" />
      <BusinessConfigList
        createTitle="审批人配置"
        description="审批人配置用于把组织角色、具体用户或管理员角色绑定到流程节点。配置变更只影响后续新申请。"
        items={items}
        path="/api/workflows/approvers"
        title="审批人配置"
      />
    </AdminShell>
  );
}
