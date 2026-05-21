import { Search } from "lucide-react";
import { AppShell } from "@/components/app-shell";
import { MaterialPurchaseActions } from "@/components/material-purchase-actions";
import { MaterialPurchaseForm } from "@/components/material-purchase-form";
import { MaterialsNav } from "@/components/materials-nav";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api, MaterialPurchase } from "@/lib/api";
import { isMaterialAdminRole } from "@/lib/permissions";

export default async function MaterialPurchasesPage({
  searchParams,
}: {
  searchParams?: Promise<{ q?: string; status?: string }>;
}) {
  const params = (await searchParams) ?? {};
  const query = (params.q ?? "").trim().toLowerCase();
  const [materials, purchasableMaterials, purchases, currentUser] = await Promise.all([api.materials(), api.purchasableMaterials(), api.materialPurchases(), api.me()]);
  const isAdmin = isMaterialAdminRole(currentUser.role);
  const canManageActions = isAdmin;
  const visiblePurchases = purchases.filter((item) => {
    const matchesSearch =
      query === "" ||
      [item.purchaseSerialNo, item.materialName, item.purchaseIdNo, item.purchaseProjectName, item.purchaseItemName, item.purchaseBrand, item.purchaseSpec, item.requester, item.groupName, item.supplier, item.reason].some((value) => value.toLowerCase().includes(query));
    const matchesStatus = !params.status || item.status === params.status;
    return matchesSearch && matchesStatus;
  });

  return (
    <AppShell currentUser={currentUser}>
      <div className="mb-6">
        <h1 className="text-2xl font-bold">资源申购</h1>
        <p className="mt-1 text-sm text-muted-foreground">登记标准品/标准物质、试剂或耗材申购，跟踪退回修改、下单和到货入库状态。</p>
      </div>

      <MaterialsNav active="purchases" />

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_360px]">
        <Card className="min-w-0">
          <CardHeader>
            <CardTitle>申购单</CardTitle>
          </CardHeader>
          <CardContent>
            <form action="/materials/purchases" className="mb-4 grid gap-3 md:grid-cols-[minmax(0,1fr)_180px_auto]">
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
                <MaterialPurchaseCard currentUserId={currentUser.id} isAdmin={isAdmin} item={item} key={item.id} canManageActions={canManageActions} purchasableMaterials={purchasableMaterials} />
              ))}
            </div>

            <div className="hidden overflow-x-auto rounded-lg border xl:block">
              <table className="w-full table-fixed text-left text-sm">
                <thead className="bg-slate-50 text-slate-500">
                  <tr>
                    <th className="p-3">资源</th>
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
                        <MaterialPurchaseActions
                          canCancel={isAdmin || item.requesterId === currentUser.id}
                          canOrder={canManageActions}
                          canReceive={canManageActions && Boolean(item.materialId)}
                          canReview={canManageActions}
                          id={item.id}
                          purchase={item}
                          purchasableMaterials={purchasableMaterials}
                          status={item.status}
                        />
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            {visiblePurchases.length === 0 ? <p className="rounded-lg border p-4 text-sm text-slate-500">当前筛选下暂无申购单。</p> : null}
          </CardContent>
        </Card>

        <aside className="min-w-0">
          <Card>
            <CardHeader>
              <CardTitle>新建申购</CardTitle>
            </CardHeader>
            <CardContent>
              <MaterialPurchaseForm materials={materials} purchasableMaterials={purchasableMaterials} />
            </CardContent>
          </Card>
        </aside>
      </div>
    </AppShell>
  );
}

function MaterialPurchaseCard({
  item,
  canManageActions,
  currentUserId,
  isAdmin,
  purchasableMaterials,
}: {
  item: MaterialPurchase;
  canManageActions: boolean;
  currentUserId: string;
  isAdmin: boolean;
  purchasableMaterials: Awaited<ReturnType<typeof api.purchasableMaterials>>;
}) {
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
        <MaterialPurchaseActions
          canCancel={isAdmin || item.requesterId === currentUserId}
          canOrder={canManageActions}
          canReceive={canManageActions && Boolean(item.materialId)}
          canReview={canManageActions}
          id={item.id}
          purchase={item}
          purchasableMaterials={purchasableMaterials}
          status={item.status}
        />
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

function purchaseStatusLabel(status: string) {
  const labels: Record<string, string> = {
    registered: "已登记",
    approved: "已通过",
    rejected: "已拒绝",
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
