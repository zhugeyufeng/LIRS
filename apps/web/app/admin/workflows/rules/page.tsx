import { AdminShell, requireAdminSection } from "@/components/admin-shell";
import { BusinessConfigList } from "@/components/business-config-list";
import { WorkflowTabs } from "@/components/workflow-tabs";
import { api } from "@/lib/api";

export default async function WorkflowRulesPage() {
  await requireAdminSection("workflows");
  const items = await api.workflowConfigs("rules").catch(() => []);

  return (
    <AdminShell active="workflows" title="审批规则" description="按金额、仪器、耗材类型、部门或团队配置审批流触发规则。">
      <WorkflowTabs active="rules" />
      <BusinessConfigList
        createTitle="审批规则"
        description="审批规则用于决定某条申请进入哪个流程模板。建议在配置 JSON 中写入 threshold、resourceType、department、materialCategory 等条件。"
        items={items}
        path="/api/workflows/rules"
        title="审批规则"
      />
    </AdminShell>
  );
}
