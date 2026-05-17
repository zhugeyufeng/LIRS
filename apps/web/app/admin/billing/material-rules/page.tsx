import { AdminShell, requireAdminSection } from "@/components/admin-shell";
import { BusinessConfigList } from "@/components/business-config-list";
import { api } from "@/lib/api";

export default async function MaterialBillingRulesPage() {
  await requireAdminSection("finance");
  const items = await api.billingRules("material-rules").catch(() => []);

  return (
    <AdminShell active="finance" title="耗材收费规则" description="按合同价、成本价、加成比例或免费规则维护耗材计费口径。">
      <BusinessConfigList
        createTitle="耗材收费规则"
        description="耗材收费规则写入数据库，可在配置 JSON 中设置 priceSource、markupRate、freeCategories、contractRequired 等字段。"
        items={items}
        path="/api/billing/material-rules"
        title="耗材收费规则"
      />
    </AdminShell>
  );
}
