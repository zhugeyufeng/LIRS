import { AdminShell, requireAdminSection } from "@/components/admin-shell";
import { BusinessConfigList } from "@/components/business-config-list";
import { api } from "@/lib/api";

export default async function InstrumentBillingRulesPage() {
  await requireAdminSection("finance");
  const items = await api.billingRules("instrument-rules").catch(() => []);

  return (
    <AdminShell active="finance" title="仪器收费规则" description="按时长、样品数、用户类型或仪器类型维护计费规则。">
      <BusinessConfigList
        createTitle="仪器收费规则"
        description="仪器收费规则写入数据库，可在配置 JSON 中设置 billingMode、hourlyRate、sampleRate、freeQuota、userType 等字段。"
        items={items}
        path="/api/billing/instrument-rules"
        title="仪器收费规则"
      />
    </AdminShell>
  );
}
