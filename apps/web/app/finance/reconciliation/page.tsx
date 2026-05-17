import { FinanceFrame } from "@/components/finance-frame";
import { LedgerExportButton } from "@/components/ledger-export-button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";

export default async function ReconciliationPage() {
  const currentUser = await api.me();
  const [ledger, accounts] =
    currentUser.role === "super_admin" || currentUser.financeEnabled
      ? await Promise.all([api.ledger().catch(() => []), api.financialAccounts().catch(() => [])])
      : [[], []];
  const ledgerTotal = ledger.reduce((sum, item) => sum + item.amount, 0);
  const balanceTotal = accounts.reduce((sum, item) => sum + item.balance, 0);

  return (
    <FinanceFrame active="reconciliation" currentUser={currentUser} title="结算对账" description="核对流水合计、账户余额和导出归档数据。">
      <div className="mb-6 grid gap-4 sm:grid-cols-3">
        <Metric label="流水合计" value={`¥${ledgerTotal.toFixed(2)}`} />
        <Metric label="余额合计" value={`¥${balanceTotal.toFixed(2)}`} />
        <Metric label="账户数量" value={accounts.length} />
      </div>
      <Card>
        <CardHeader className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <CardTitle>对账导出</CardTitle>
          <LedgerExportButton />
        </CardHeader>
        <CardContent className="text-sm leading-6 text-slate-600">
          当前流水与账户均从数据库读取。历史流水不可在前端删除，纠错应通过财务调整流水处理。
        </CardContent>
      </Card>
    </FinanceFrame>
  );
}

function Metric({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="rounded-lg border bg-white p-4">
      <p className="text-sm text-slate-500">{label}</p>
      <p className="mt-2 break-words text-2xl font-bold">{value}</p>
    </div>
  );
}
