import { FinanceFrame } from "@/components/finance-frame";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";

export default async function BillsPage() {
  const currentUser = await api.me();
  const ledger = currentUser.role === "super_admin" || currentUser.financeEnabled ? await api.ledger().catch(() => []) : [];
  const bills = Array.from(
    ledger.reduce((map, entry) => {
      const month = new Date(entry.createdAt).toLocaleDateString("zh-CN", { year: "numeric", month: "2-digit", timeZone: "Asia/Shanghai" });
      const key = `${month}-${entry.userId || entry.groupName}`;
      const current = map.get(key) ?? { key, month, owner: entry.userName || entry.groupName, count: 0, amount: 0 };
      current.count += 1;
      current.amount += entry.amount ?? 0;
      map.set(key, current);
      return map;
    }, new Map<string, { key: string; month: string; owner: string; count: number; amount: number }>()),
  ).map(([, value]) => value);

  return (
    <FinanceFrame active="bills" currentUser={currentUser} title="月度账单" description="按月份和个人汇总费用流水，作为课题组或财务确认的账单基础。">
      <Card>
        <CardHeader>
          <CardTitle>月度账单列表</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {bills.map((bill) => (
            <article className="rounded-lg border bg-white p-4" key={bill.key}>
              <div className="flex flex-col justify-between gap-3 sm:flex-row sm:items-start">
                <div>
                  <p className="text-xs font-bold text-primary">{bill.month}</p>
                  <h2 className="mt-1 font-bold text-slate-900">{bill.owner}</h2>
                  <p className="mt-1 text-xs text-slate-500">流水 {bill.count} 条</p>
                </div>
                <span className="w-fit rounded-full bg-emerald-50 px-2 py-1 text-xs font-bold text-emerald-700">已生成</span>
              </div>
              <p className="mt-3 text-lg font-bold text-slate-900">¥{bill.amount.toFixed(2)}</p>
            </article>
          ))}
          {bills.length === 0 ? <p className="rounded-lg border border-dashed p-4 text-sm text-slate-500">暂无月度账单。</p> : null}
        </CardContent>
      </Card>
    </FinanceFrame>
  );
}
