import { FinanceFrame } from "@/components/finance-frame";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";

export default async function InstrumentFeesPage() {
  const currentUser = await api.me();
  const ledger = currentUser.role === "super_admin" || currentUser.financeEnabled ? await api.ledger().catch(() => []) : [];
  const rows = ledger.filter((item) => item.reservationId || item.entryType === "debit");
  const total = rows.reduce((sum, item) => sum + (item.amount ?? 0), 0);

  return (
    <FinanceFrame active="instrument-fees" currentUser={currentUser} title="机时费用" description="根据预约与实际使用生成机时费用流水。">
      <div className="mb-6 grid gap-4 sm:grid-cols-3">
        <Metric label="费用记录" value={rows.length} />
        <Metric label="费用合计" value={`¥${total.toFixed(2)}`} />
        <Metric label="数据来源" value="流水账" />
      </div>
      <FeeTable rows={rows} title="机时费用流水" />
    </FinanceFrame>
  );
}

function FeeTable({ rows, title }: { rows: { id: string; description: string; userName?: string; groupName: string; amount: number; createdAt: string }[]; title: string }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="grid gap-3 xl:hidden">
          {rows.map((item) => (
            <article className="rounded-lg border p-4 text-sm" key={item.id}>
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0">
                  <p className="break-words font-bold text-slate-900">{item.description}</p>
                  <p className="mt-1 text-xs text-slate-500">{item.userName || item.groupName}</p>
                </div>
                <span className="shrink-0 font-bold text-primary">¥{(item.amount ?? 0).toFixed(2)}</span>
              </div>
              <p className="mt-3 text-xs text-slate-500">{formatDateTime(item.createdAt)}</p>
            </article>
          ))}
        </div>
        <div className="hidden overflow-x-auto xl:block">
          <table className="w-full text-left text-sm">
            <thead className="border-b text-slate-500">
              <tr>
                <th className="py-3 pr-4">说明</th>
                <th className="py-3 pr-4">人员</th>
                <th className="py-3 pr-4">时间</th>
                <th className="py-3 text-right">金额</th>
              </tr>
            </thead>
            <tbody className="divide-y">
              {rows.map((item) => (
                <tr key={item.id}>
                  <td className="py-3 pr-4">{item.description}</td>
                  <td className="py-3 pr-4">{item.userName || item.groupName}</td>
                  <td className="py-3 pr-4 text-xs text-slate-500">{formatDateTime(item.createdAt)}</td>
                  <td className="py-3 text-right font-bold">¥{(item.amount ?? 0).toFixed(2)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {rows.length === 0 ? <p className="rounded-lg border border-dashed p-4 text-sm text-slate-500">暂无机时费用流水。</p> : null}
      </CardContent>
    </Card>
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

function formatDateTime(value: string) {
  return new Date(value).toLocaleString("zh-CN", { month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit", timeZone: "Asia/Shanghai" });
}
