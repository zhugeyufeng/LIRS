import Link from "next/link";
import { AlertTriangle, ClipboardList, PackageCheck, Search, Settings2, ShoppingCart } from "lucide-react";
import { AppShell } from "@/components/app-shell";
import { MaterialRequestDialog } from "@/components/material-request-form";
import { MaterialsNav } from "@/components/materials-nav";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api, Material } from "@/lib/api";
import { isMaterialAdminRole } from "@/lib/permissions";

const resourceTypeSections = [
  { type: "standard", title: "标准品", description: "标准品按批次和唯一编号独立展示，可直接选择可用编号申领。" },
  { type: "reagent", title: "试剂", description: "试剂按目录、批次、库位和库存状态独立展示。" },
  { type: "consumable", title: "耗材", description: "耗材按最小单位、库位和预警状态独立展示。" },
] as const;

export default async function MaterialsPage({
  searchParams,
}: {
  searchParams?: Promise<{ q?: string; category?: string; productType?: string; status?: string }>;
}) {
  const params = (await searchParams) ?? {};
  const query = (params.q ?? "").trim().toLowerCase();
  const [materials, requests, purchases, currentUser, materialCategories] = await Promise.all([api.materials(), api.materialRequests(), api.materialPurchases(), api.me(), api.materialCategories().catch(() => [])]);
  const isAdmin = isMaterialAdminRole(currentUser.role);
  const visibleMaterials = materials.filter((item) => {
    const matchesSearch =
      query === "" ||
      [item.name, item.category, item.spec, item.supplier, item.manufacturer, item.batchNo, item.catalogNo, item.casNo, item.qrCode, item.tenderContract, item.contractNo, materialLocation(item), materialBatchSummary(item), materialUnitCodes(item)].some((value) => value.toLowerCase().includes(query));
    const matchesCategory = !params.category || item.category === params.category;
    const matchesProductType = !params.productType || item.productType === params.productType;
    const matchesStatus = !params.status || item.status === params.status;
    return matchesSearch && matchesCategory && matchesProductType && matchesStatus;
  });
  const warningMaterials = materials.filter((item) => ["near_expiry", "expired", "open_expired", "freeze_thaw_exceeded", "low", "damaged"].includes(item.status));
  const activeResourceSections = resourceTypeSections.filter((section) => !params.productType || section.type === params.productType);
  const displayedResourceSections = (activeResourceSections.length > 0 ? activeResourceSections : resourceTypeSections).map((section) => ({
    ...section,
    materials: visibleMaterials.filter((item) => item.productType === section.type),
    total: materials.filter((item) => item.productType === section.type).length,
  }));
  const categories = Array.from(
    new Set([
      ...materialCategories.filter((item) => item.status === "active" && item.parentName.trim() === "").map((item) => item.name),
      ...materials.map((item) => item.category).filter(Boolean),
    ]),
  ).sort((a, b) => a.localeCompare(b, "zh-CN"));

  return (
    <AppShell currentUser={currentUser}>
      <div className="mb-6 flex flex-col justify-between gap-4 xl:flex-row xl:items-end">
        <div className="min-w-0">
          <h1 className="text-2xl font-bold">资源目录</h1>
          <p className="mt-1 text-sm text-muted-foreground">标准品、试剂和耗材分区展示；一级目录、二级目录、批次、库位和唯一编号保持独立，可直接在资源目录发起申领。</p>
        </div>
        <form action="/materials" className="grid w-full gap-2 sm:grid-cols-2 xl:flex xl:w-auto xl:flex-row">
          <div className="relative sm:col-span-2 xl:col-span-1 xl:w-80">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" aria-hidden="true" />
            <input className="h-10 w-full rounded-lg border bg-white pl-10 pr-3 text-sm" defaultValue={params.q ?? ""} name="q" placeholder="搜索名称、CAS、货号、批号、库位、合同、供应商" />
          </div>
          <select className="h-10 min-w-0 rounded-lg border bg-white px-3 text-sm" defaultValue={params.category ?? ""} name="category">
            <option value="">全部一级目录</option>
            {categories.map((category) => (
              <option key={category} value={category}>
                {category}
              </option>
            ))}
          </select>
          <select className="h-10 min-w-0 rounded-lg border bg-white px-3 text-sm" defaultValue={params.productType ?? ""} name="productType">
            <option value="">全部资源类型</option>
            <option value="standard">标准品</option>
            <option value="reagent">试剂</option>
            <option value="consumable">耗材</option>
          </select>
          <select className="h-10 min-w-0 rounded-lg border bg-white px-3 text-sm" defaultValue={params.status ?? ""} name="status">
            <option value="">全部状态</option>
            <option value="normal">正常</option>
            <option value="near_expiry">临期</option>
            <option value="low">低库存</option>
            <option value="expired">已过期</option>
            <option value="open_expired">开封超期</option>
            <option value="freeze_thaw_exceeded">冻融超限</option>
            <option value="damaged">损毁</option>
            <option value="disabled">停用</option>
          </select>
          <button className="inline-flex h-10 w-full min-w-20 items-center justify-center whitespace-nowrap rounded-lg bg-primary px-4 text-sm font-bold text-white sm:col-span-2 xl:col-span-1 xl:w-auto" type="submit">
            筛选
          </button>
        </form>
      </div>

      <MaterialsNav active="overview" />

      <div className="mb-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-6">
        <Metric label="标准品" value={materials.filter((item) => item.productType === "standard").length} />
        <Metric label="试剂" value={materials.filter((item) => item.productType === "reagent").length} />
        <Metric label="耗材" value={materials.filter((item) => item.productType === "consumable").length} />
        <Metric label="预警数量" value={warningMaterials.length} />
        <Metric label="待处理申领" value={requests.filter((item) => item.status === "pending").length} />
        <Metric label="待处理申购" value={purchases.filter((item) => item.status === "pending").length} />
      </div>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_360px]">
        <div className="min-w-0 space-y-6">
          {displayedResourceSections.map((section) => (
            <MaterialSection description={section.description} items={section.materials} key={section.type} title={section.title} total={section.total} />
          ))}
        </div>

        <aside className="min-w-0 space-y-6">
          {isAdmin ? (
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Settings2 className="h-5 w-5 text-primary" />
                  管理入口
                </CardTitle>
              </CardHeader>
              <CardContent>
                <Link className="inline-flex h-10 w-full items-center justify-center rounded-md bg-primary px-4 text-sm font-bold text-white" href="/admin/materials">
                  进入资源管理
                </Link>
              </CardContent>
            </Card>
          ) : null}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <PackageCheck className="h-5 w-5 text-primary" />
                资源操作
              </CardTitle>
            </CardHeader>
            <CardContent className="grid gap-3">
              <Link className="inline-flex min-h-11 items-center justify-between gap-3 rounded-md border px-3 py-2 text-sm font-bold text-slate-700 hover:bg-slate-50" href="/materials/requests">
                <span className="inline-flex min-w-0 items-center gap-2">
                  <ClipboardList className="h-4 w-4 shrink-0 text-primary" aria-hidden="true" />
                  <span className="truncate">申领记录</span>
                </span>
                <span className="shrink-0 rounded bg-slate-100 px-2 py-1 text-xs">{requests.length}</span>
              </Link>
              <Link className="inline-flex min-h-11 items-center justify-between gap-3 rounded-md border px-3 py-2 text-sm font-bold text-slate-700 hover:bg-slate-50" href="/materials/purchases">
                <span className="inline-flex min-w-0 items-center gap-2">
                  <ShoppingCart className="h-4 w-4 shrink-0 text-primary" aria-hidden="true" />
                  <span className="truncate">进入申购</span>
                </span>
                <span className="shrink-0 rounded bg-slate-100 px-2 py-1 text-xs">{purchases.length}</span>
              </Link>
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <AlertTriangle className="h-5 w-5 text-amber-600" />
                预警中心
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              {warningMaterials.length === 0 ? <p className="text-sm text-slate-500">暂无预警。</p> : null}
              {warningMaterials.map((item) => (
                <div className="break-words rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800" key={item.id}>
                  <p className="font-bold">{materialStatusLabel(item.status)}：{item.name}</p>
                  <p className="mt-1">库存 {item.stock}{item.unit}，告警线 {item.warningLine}{item.unit}，有效期 {item.expiresAt || "未登记"}。</p>
                </div>
              ))}
            </CardContent>
          </Card>
        </aside>
      </div>
    </AppShell>
  );
}

function Metric({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-lg border bg-white p-4">
      <p className="text-sm text-slate-500">{label}</p>
      <p className="mt-2 text-2xl font-bold">{value}</p>
    </div>
  );
}

function MaterialSection({ title, description, items, total }: { title: string; description: string; items: Material[]; total: number }) {
  return (
    <Card className="min-w-0">
      <CardHeader className="border-b bg-slate-50/50">
        <div className="flex flex-col justify-between gap-2 sm:flex-row sm:items-start">
          <div className="min-w-0">
            <CardTitle>{title}</CardTitle>
            <p className="mt-1 text-sm text-slate-500">{description}</p>
          </div>
          <span className="w-fit rounded bg-white px-2 py-1 text-xs font-bold text-slate-600">
            {items.length}/{total}
          </span>
        </div>
      </CardHeader>
      <CardContent className="pt-6">
        <div className="grid gap-3 xl:hidden">
          {items.map((item) => (
            <div className="rounded-lg border bg-white p-4 text-sm" key={item.id}>
              <div className="flex flex-col justify-between gap-3 sm:flex-row sm:items-start">
                <div className="min-w-0">
                  <Link className="break-words font-bold text-primary hover:underline" href={`/materials/${item.id}`}>
                    {item.name}
                  </Link>
                  <p className="mt-1 break-words text-xs text-slate-500">
                    {item.category}{item.subcategory ? ` / ${item.subcategory}` : ""} / {item.supplier || "未登记供应商"}
                  </p>
                </div>
                <StatusPill status={item.status} />
              </div>
              <div className="mt-4 grid gap-3 sm:grid-cols-2">
                <InfoItem label="规格" value={item.spec} />
                <InfoItem label="CAS/级别" value={`${item.casNo || "未登记"} / ${item.grade || "未登记"}`} />
                <InfoItem label="货号" value={item.catalogNo} />
                <InfoItem label="编号/批次" value={materialBatchSummary(item)} />
                <InfoItem label="库位" value={materialLocation(item)} />
                <InfoItem label="开封/冻融" value={`${item.openedAt || "未开封"} / ${item.freezeThawCount}/${item.freezeThawLimit || "不限"}`} />
                <InfoItem label="招标合同" value={item.tenderContract} />
                <InfoItem label="合同序号" value={item.contractNo} />
                <InfoItem label="来源/稀释" value={`${item.parentMaterialName || "无"} / ${item.dilutionFactor || "未登记"}`} />
                <InfoItem label="状态" value={materialStatusLabel(item.status)} />
                <InfoItem label="单价" value={`¥${item.unitPrice.toFixed(2)}`} />
              </div>
              <div className="mt-4">
                <MaterialRequestDialog buttonClassName="w-full sm:w-auto" material={item} />
              </div>
            </div>
          ))}
        </div>
        <div className="hidden overflow-x-auto rounded-lg border xl:block">
          <table className="w-full table-fixed text-left text-sm">
            <thead className="border-b text-slate-500">
              <tr>
                <th className="px-3 py-3">名称</th>
                <th className="px-3 py-3">规格</th>
                <th className="px-3 py-3">库存</th>
                <th className="px-3 py-3">编号/批次</th>
                <th className="px-3 py-3 text-right">单价</th>
                <th className="w-28 px-3 py-3">操作</th>
              </tr>
            </thead>
            <tbody className="divide-y">
              {items.map((item) => (
                <tr key={item.id}>
                  <td className="px-3 py-3 align-top">
                    <Link className="break-words font-bold text-slate-900 hover:text-primary hover:underline" href={`/materials/${item.id}`}>
                      {item.name}
                    </Link>
                    <p className="break-words text-xs text-slate-500">
                      {item.category}{item.subcategory ? ` / ${item.subcategory}` : ""} / {item.supplier || "未登记供应商"}
                    </p>
                    <p className="mt-1 break-words text-xs text-slate-500">CAS：{item.casNo || "未登记"} / 货号：{item.catalogNo || "未登记"}</p>
                  </td>
                  <td className="break-words px-3 py-3 align-top">
                    <p>{item.spec}</p>
                    <p className="mt-1 text-xs text-slate-500">{item.storageCondition || "未登记保存条件"}</p>
                  </td>
                  <td className="px-3 py-3 align-top">
                    <span className={item.stock <= item.warningLine ? "font-bold text-amber-700" : "font-bold"}>
                      {item.stock}{item.unit}
                    </span>
                  </td>
                  <td className="break-words px-3 py-3 align-top text-xs text-slate-500">
                    <p>{materialBatchSummary(item)}</p>
                    <p className="mt-1">{materialLocation(item) || "未登记库位"}</p>
                    <p className="mt-1">开封：{item.openedAt || "未开封"}{item.openExpiresAt ? ` / 到期 ${item.openExpiresAt}` : ""}</p>
                    {item.parentMaterialName || item.dilutionFactor ? <p className="mt-1">来源：{item.parentMaterialName || item.parentMaterialId || "未登记"} / {item.dilutionFactor || "未登记"}</p> : null}
                  </td>
                  <td className="whitespace-nowrap px-3 py-3 text-right align-top font-bold">¥{item.unitPrice.toFixed(2)}</td>
                  <td className="px-3 py-3 align-top">
                    <MaterialRequestDialog buttonClassName="h-9 w-full" material={item} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {items.length === 0 ? <p className="rounded-lg border p-4 text-sm text-slate-500">当前筛选下暂无{title}。</p> : null}
      </CardContent>
    </Card>
  );
}

function InfoItem({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-xs text-slate-500">{label}</p>
      <p className="mt-1 break-words font-medium text-slate-800">{value || "未登记"}</p>
    </div>
  );
}

function StatusPill({ status }: { status: string }) {
  const warning = ["near_expiry", "low", "open_expired", "freeze_thaw_exceeded", "damaged"].includes(status);
  const danger = ["expired", "disabled"].includes(status);
  const className = danger
    ? "w-fit shrink-0 rounded bg-red-50 px-2 py-1 text-xs font-bold text-red-700"
    : warning
      ? "w-fit shrink-0 rounded bg-amber-50 px-2 py-1 text-xs font-bold text-amber-700"
      : "w-fit shrink-0 rounded bg-emerald-50 px-2 py-1 text-xs font-bold text-emerald-700";
  return <span className={className}>{materialStatusLabel(status)}</span>;
}

function materialLocation(item: Pick<Material, "storageRoom" | "storageCabinet" | "storageLayer" | "storageSlot">) {
  return [item.storageRoom, item.storageCabinet, item.storageLayer, item.storageSlot].filter(Boolean).join(" / ");
}

function materialBatchSummary(item: Material) {
  const units = item.units ?? [];
  const batches = (item.batches ?? []).filter((batch) => batch.status !== "disabled");
  if (batches.length > 0) {
    return `编号 ${units.length} 个 / 批次 ${batches.length} 个 / 可用 ${units.filter((unit) => unit.status === "available").length}${item.unit}`;
  }
  if (units.length > 0) {
    return `编号 ${units.length} 个 / 可用 ${units.filter((unit) => unit.status === "available").length}${item.unit}`;
  }
  return `${item.batchNo || "未登记批次"} / ${item.expiresAt || "未登记有效期"}`;
}

function materialUnitCodes(item: Material) {
  return (item.units ?? []).map((unit) => unit.unitCode).join(" ");
}

function materialStatusLabel(status: string) {
  const labels: Record<string, string> = {
    normal: "正常",
    near_expiry: "临期",
    low: "低库存",
    expired: "过期",
    open_expired: "开封超期",
    freeze_thaw_exceeded: "冻融超限",
    damaged: "损毁",
    disabled: "停用",
  };
  return labels[status] ?? status;
}
