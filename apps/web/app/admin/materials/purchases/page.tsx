import { Search } from "lucide-react";
import { AdminShell, requireAdminSection } from "@/components/admin-shell";
import { MaterialPurchaseAdminNav } from "@/components/material-purchase-admin-nav";
import { MaterialPurchaseActions, MaterialPurchaseForm, MaterialPurchaseMonthConfirmButton } from "@/components/material-purchase-form";
import { MaterialsNav } from "@/components/materials-nav";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api, MaterialPurchase } from "@/lib/api";
import { isTenantAdminRole } from "@/lib/permissions";

export default async function AdminMaterialPurchasesPage({
  searchParams,
}: {
  searchParams?: Promise<{ q?: string; status?: string }>;
}) {
  const currentUser = await requireAdminSection("materials");
  const params = (await searchParams) ?? {};
  const query = (params.q ?? "").trim().toLowerCase();
  const [materials, purchasableMaterials, purchases] = await Promise.all([
    api.materials().catch(() => []),
    api.purchasableMaterials().catch(() => []),
    api.materialPurchases().catch(() => []),
  ]);
  const canManageActions = isTenantAdminRole(currentUser.role) || currentUser.role === "material_admin";
  const visiblePurchases = purchases.filter((item) => {
    const matchesSearch =
      query === "" ||
      [item.purchaseSerialNo, item.materialName, item.purchaseIdNo, item.purchaseProjectName, item.purchaseItemName, item.purchaseBrand, item.purchaseSpec, item.requester, item.groupName, item.supplier, item.reason].some((value) => value.toLowerCase().includes(query));
    const matchesStatus = !params.status || item.status === params.status;
    return matchesSearch && matchesStatus;
  });
  return (
    <AdminShell active="materials" title="资源申购管理" description="独立处理标准品/标准物质、试剂和耗材申购登记、退回修改、下单和到货入库；到货会自动增加库存并写入库存流水。">
      <MaterialsNav active="purchases" admin />
      <MaterialPurchaseAdminNav active="orders" />

      <div className="mb-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <Metric label="全部申购" value={purchases.length} />
        <Metric label="可采购目录" value={purchasableMaterials.length} />
        <Metric label="已登记" value={purchases.filter((item) => item.status === "registered").length} />
        <Metric label="待入库" value={purchases.filter((item) => item.materialId && (item.status === "registered" || item.status === "ordered")).length} />
      </div>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_380px]">
        <Card className="min-w-0">
          <CardHeader>
            <CardTitle>申购单列表</CardTitle>
          </CardHeader>
          <CardContent>
            <form action="/admin/materials/purchases" className="mb-4 grid gap-3 md:grid-cols-[minmax(0,1fr)_180px_auto]">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" aria-hidden="true" />
                <input className="h-10 w-full rounded-md border bg-white pl-10 pr-3 text-sm" defaultValue={params.q ?? ""} name="q" placeholder="搜索ID号、项目、品牌、规格、申请人" />
              </div>
              <select className="h-10 min-w-0 rounded-md border bg-white px-3 text-sm" defaultValue={params.status ?? ""} name="status">
                <option value="">全部状态</option>
                <option value="registered">已登记</option>
                <option value="returned">退回修改</option>
                <option value="ordered">已下单</option>
                <option value="received">已入库</option>
                <option value="cancelled">已取消</option>
              </select>
              <button className="inline-flex h-10 w-full items-center justify-center rounded-md bg-primary px-4 text-sm font-bold text-white md:w-auto" type="submit">
                筛选
              </button>
            </form>

            <div className="grid gap-3 xl:hidden">
              {visiblePurchases.map((item) => (
                <MaterialPurchaseCard canManageActions={canManageActions} item={item} key={item.id} purchasableMaterials={purchasableMaterials} />
              ))}
            </div>

            <div className="hidden overflow-x-auto rounded-lg border xl:block">
              <table className="w-full table-fixed text-left text-sm">
                <thead className="bg-slate-50 text-slate-500">
                  <tr>
                    <th className="p-3">采购物资</th>
                    <th className="p-3">申请人</th>
                    <th className="p-3">金额</th>
                    <th className="p-3">原因</th>
                    <th className="w-28 p-3">状态</th>
                    <th className="w-[24rem] p-3">操作</th>
                  </tr>
                </thead>
                <tbody className="divide-y">
                  {visiblePurchases.map((item) => (
                    <tr key={item.id}>
                      <td className="break-words p-3 align-top">
                        <p className="font-bold">{purchaseTitle(item)}</p>
                        <p className="mt-1 text-xs font-medium text-slate-500">{item.purchaseSerialNo || item.id}</p>
                        <p className="mt-1 text-xs text-slate-500">数量 {item.quantity} / {item.purchaseBrand || item.supplier || "未指定品牌"}</p>
                        {item.purchaseProjectName ? <p className="mt-1 break-words text-xs text-slate-500">{item.purchaseProjectName}</p> : null}
                      </td>
                      <td className="break-words p-3 align-top">
                        <p className="font-medium">{item.requester}</p>
                        <p className="mt-1 text-xs text-slate-500">{item.groupName}</p>
                      </td>
                      <td className="p-3 align-top font-bold">{formatMoney(item.estimatedUnitPrice * item.quantity)}</td>
                      <td className="break-words p-3 align-top">{item.reason}</td>
                      <td className="p-3 align-top">
                        <span className="rounded bg-slate-100 px-2 py-1 text-xs font-bold">{purchaseStatusLabel(item.status)}</span>
                      </td>
                      <td className="p-3 align-top">
                        <MaterialPurchaseActions canCancel={canManageActions} canOrder={canManageActions} canReceive={canManageActions && Boolean(item.materialId)} canReview={canManageActions} id={item.id} purchase={item} purchasableMaterials={purchasableMaterials} status={item.status} />
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            {visiblePurchases.length === 0 ? <p className="rounded-lg border p-4 text-sm text-slate-500">当前筛选下暂无申购单。</p> : null}
          </CardContent>
        </Card>

        <aside className="min-w-0 space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>新建申购与汇总</CardTitle>
            </CardHeader>
            <CardContent>
              <MaterialPurchaseForm materials={materials} purchasableMaterials={purchasableMaterials} />
              <div className="mt-4">
                <DownloadMonthlyButton />
              </div>
              <div className="mt-4 border-t pt-4">
                <MaterialPurchaseMonthConfirmButton />
              </div>
            </CardContent>
          </Card>
        </aside>
      </div>
    </AdminShell>
  );
}

function MaterialPurchaseCard({ canManageActions, item, purchasableMaterials }: { canManageActions: boolean; item: MaterialPurchase; purchasableMaterials: Awaited<ReturnType<typeof api.purchasableMaterials>> }) {
  return (
    <div className="rounded-lg border bg-white p-4">
      <div className="flex flex-col justify-between gap-3 sm:flex-row sm:items-start">
        <div className="min-w-0">
          <p className="break-words font-bold text-slate-900">{purchaseTitle(item)} x{item.quantity}</p>
          <p className="mt-1 break-words text-xs font-medium text-slate-500">{item.purchaseSerialNo || item.id}</p>
          <p className="mt-1 break-words text-sm text-slate-500">{item.requester} / {item.groupName}</p>
          {item.purchaseProjectName ? <p className="mt-1 break-words text-xs text-slate-500">{item.purchaseProjectName}</p> : null}
        </div>
        <span className="w-fit shrink-0 rounded bg-slate-100 px-2 py-1 text-xs font-bold">{purchaseStatusLabel(item.status)}</span>
      </div>
      <div className="mt-3 grid gap-3 text-sm sm:grid-cols-2">
        <InfoItem label="预计金额" value={formatMoney(item.estimatedUnitPrice * item.quantity)} />
        <InfoItem label="品牌/规格" value={`${item.purchaseBrand || "未指定品牌"} / ${item.purchaseSpec || "未登记规格"}`} />
      </div>
      <p className="mt-3 break-words text-sm text-slate-600">{item.reason}</p>
      <div className="mt-3">
        <MaterialPurchaseActions canCancel={canManageActions} canOrder={canManageActions} canReceive={canManageActions && Boolean(item.materialId)} canReview={canManageActions} id={item.id} purchase={item} purchasableMaterials={purchasableMaterials} status={item.status} />
      </div>
    </div>
  );
}

function InfoItem({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-xs text-slate-500">{label}</p>
      <p className="mt-1 break-words font-medium text-slate-800">{value}</p>
    </div>
  );
}

function Metric({ label, value }: { label: string; value: number | string }) {
  return (
    <div className="rounded-lg border bg-white p-4">
      <p className="text-sm text-slate-500">{label}</p>
      <p className="mt-2 text-2xl font-bold">{value}</p>
    </div>
  );
}

function purchaseStatusLabel(status: string) {
  const labels: Record<string, string> = {
    pending: "已登记",
    registered: "已登记",
    returned: "退回修改",
    ordered: "已下单",
    received: "已入库",
    cancelled: "已取消",
  };
  return labels[status] ?? status;
}

function formatMoney(value: number) {
  return `¥${value.toFixed(2)}`;
}

function purchaseTitle(item: MaterialPurchase) {
  return item.purchaseItemName || item.materialName || item.purchaseProjectName;
}

function DownloadMonthlyButton() {
  const month = new Date().toISOString().slice(0, 7);
  return (
    <a className="inline-flex h-10 w-full items-center justify-center rounded-md border px-4 text-sm font-bold text-slate-700 hover:bg-slate-50" href={`/api/material-purchases/monthly-export.csv?month=${month}`}>
      导出本月申购清单
    </a>
  );
}
