import { BusinessConfigList } from "@/components/business-config-list";
import { FinanceFrame } from "@/components/finance-frame";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";
import { isFinanceAdminRole } from "@/lib/permissions";

export default async function InvoicesPage() {
  const currentUser = await api.me();
  const canManageInvoices = isFinanceAdminRole(currentUser.role);
  const items = canManageInvoices ? await api.billingRules("invoices").catch(() => []) : [];

  return (
    <FinanceFrame active="invoices" currentUser={currentUser} title="发票记录" description="维护发票抬头、开票记录和报销关联信息。">
      {canManageInvoices ? (
        <BusinessConfigList
          createTitle="发票记录"
          description="发票记录写入数据库。可在配置 JSON 中记录 invoiceTitle、taxNo、amount、billMonth、reimbursementNo 等字段。"
          items={items}
          path="/api/billing/invoices"
          title="发票记录"
        />
      ) : (
        <Card>
          <CardHeader>
            <CardTitle>发票记录</CardTitle>
          </CardHeader>
          <CardContent className="text-sm leading-6 text-slate-600">发票记录由财务管理员维护，普通用户可通过账单和流水查看个人费用。</CardContent>
        </Card>
      )}
    </FinanceFrame>
  );
}
