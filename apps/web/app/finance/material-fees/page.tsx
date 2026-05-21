import { FinanceFrame } from "@/components/finance-frame";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";
import { materialFeeStatusLabel } from "@/lib/status-labels";

export default async function MaterialFeesPage() {
  const currentUser = await api.me();
  const [materials, requests, purchases] =
    currentUser.role === "super_admin" || currentUser.financeEnabled
      ? await Promise.all([api.materials().catch(() => []), api.materialRequests().catch(() => []), api.materialPurchases().catch(() => [])])
      : [[], [], []];
  const materialPrice = new Map(materials.map((item) => [item.id, item.unitPrice]));
  const requestRows = requests.map((item) => ({
    id: item.id,
    type: "申领",
    name: item.materialName,
    requester: item.requester,
    quantity: item.quantity,
    amount: (materialPrice.get(item.materialId) ?? 0) * item.quantity,
    status: item.status,
    createdAt: item.createdAt,
  }));
  const purchaseRows = purchases.map((item) => ({
    id: item.id,
    type: "申购",
    name: item.materialName,
    requester: item.requester,
    quantity: item.quantity,
    amount: item.estimatedUnitPrice * item.quantity,
    status: item.status,
    createdAt: item.createdAt,
  }));
  const rows = [...requestRows, ...purchaseRows].sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime());
  const total = rows.reduce((sum, item) => sum + item.amount, 0);

  return (
    <FinanceFrame active="material-fees" currentUser={currentUser} title="耗材费用" description="根据耗材出库和申购记录汇总耗材费用。">
      <div className="mb-6 grid gap-4 sm:grid-cols-3">
        <Metric label="费用记录" value={rows.length} />
        <Metric label="费用估算" value={`¥${total.toFixed(2)}`} />
        <Metric label="计价依据" value="目录价/申购价" />
      </div>
      <Card>
        <CardHeader>
          <CardTitle>耗材费用记录</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {rows.map((item) => (
            <article className="rounded-lg border bg-white p-4" key={`${item.type}-${item.id}`}>
              <div className="flex flex-col justify-between gap-3 sm:flex-row sm:items-start">
                <div className="min-w-0">
                  <p className="text-xs font-bold text-primary">{item.type}</p>
                  <h2 className="mt-1 break-words font-bold text-slate-900">{item.name}</h2>
                  <p className="mt-1 break-words text-xs text-slate-500">{item.requester} / {statusLabel(item.status)}</p>
                </div>
                <span className="w-fit rounded-full bg-slate-100 px-2 py-1 text-xs font-bold text-slate-700">¥{item.amount.toFixed(2)}</span>
              </div>
              <p className="mt-3 text-sm text-slate-600">数量：{item.quantity} / 时间：{formatDateTime(item.createdAt)}</p>
            </article>
          ))}
          {rows.length === 0 ? <p className="rounded-lg border border-dashed p-4 text-sm text-slate-500">暂无耗材费用记录。</p> : null}
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

function statusLabel(status: string) {
  return materialFeeStatusLabel(status);
}

function formatDateTime(value: string) {
  return new Date(value).toLocaleString("zh-CN", { month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit", timeZone: "Asia/Shanghai" });
}
