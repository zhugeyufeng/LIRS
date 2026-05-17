import Link from "next/link";
import { AlertTriangle, ClipboardList, Search, ShoppingCart, type LucideIcon } from "lucide-react";
import { AdminShell, requireAdminSection } from "@/components/admin-shell";
import { AppShell } from "@/components/app-shell";
import { MaterialCategoryForm, MaterialCreateForm, MaterialDamageActions, MaterialDamageForm, StockAdjustmentForm } from "@/components/material-management-form";
import { MaterialsNav, type MaterialsNavKey } from "@/components/materials-nav";
import {
  AdminResourceMaterialSection,
  InfoItem,
  ResourceMaterialSection,
  type ResourceProductType,
  filterMaterials,
  materialStatusLabel,
  primaryMaterialCategories,
  resourceTypeSection,
} from "@/components/material-resource-sections";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api, type InventoryLedgerEntry, type MaterialDamage } from "@/lib/api";
import { isMaterialAdminRole } from "@/lib/permissions";

type ResourcePageSearchParams = Promise<{ q?: string; category?: string; status?: string }>;

const resourceNavKeys: Record<ResourceProductType, MaterialsNavKey> = {
  standard: "standards",
  reagent: "reagents",
  consumable: "consumables",
};

export async function MaterialResourceCatalogPage({
  productType,
  searchParams,
}: {
  productType: ResourceProductType;
  searchParams?: ResourcePageSearchParams;
}) {
  const params = (await searchParams) ?? {};
  const [materials, requests, purchases, currentUser, materialCategories] = await Promise.all([
    api.materials(),
    api.materialRequests(),
    api.materialPurchases(),
    api.me(),
    api.materialCategories().catch(() => []),
  ]);
  const section = resourceTypeSection(productType);
  const isAdmin = isMaterialAdminRole(currentUser.role);
  const typeMaterials = materials.filter((item) => item.productType === productType);
  const visibleMaterials = filterMaterials(materials, { ...params, productType });
  const warningMaterials = typeMaterials.filter((item) => ["near_expiry", "expired", "open_expired", "freeze_thaw_exceeded", "low", "damaged"].includes(item.status));
  const materialIds = new Set(typeMaterials.map((item) => item.id));
  const availableUnits = typeMaterials.reduce((sum, item) => sum + (item.units ?? []).filter((unit) => unit.status === "available").length, 0);
  const categories = primaryMaterialCategories(typeMaterials, materialCategories);

  return (
    <AppShell currentUser={currentUser}>
      <div className="mb-6 flex flex-col justify-between gap-4 xl:flex-row xl:items-end">
        <div className="min-w-0">
          <h1 className="text-2xl font-bold">{section.title}目录</h1>
          <p className="mt-1 text-sm text-muted-foreground">{section.description}</p>
        </div>
        <ResourceFilterForm action={`/materials/${section.slug}`} categories={categories} params={params} />
      </div>

      <MaterialsNav active={resourceNavKeys[productType]} />

      <div className="mb-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <Metric label={`${section.title}总数`} value={typeMaterials.length} />
        <Metric label="当前筛选" value={visibleMaterials.length} />
        <Metric label="可领用编号" value={availableUnits} />
        <Metric label="预警数量" value={warningMaterials.length} />
      </div>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_340px]">
        <div className="min-w-0">
          <ResourceMaterialSection description={section.description} items={visibleMaterials} title={`${section.title}目录`} total={typeMaterials.length} />
        </div>

        <aside className="min-w-0 space-y-6">
          {isAdmin ? (
            <Card>
              <CardHeader>
                <CardTitle>管理入口</CardTitle>
              </CardHeader>
              <CardContent>
                <Link className="inline-flex h-10 w-full items-center justify-center rounded-md bg-primary px-4 text-sm font-bold text-white" href={`/admin/materials/${section.slug}`}>
                  进入{section.title}管理
                </Link>
              </CardContent>
            </Card>
          ) : null}
          <Card>
            <CardHeader>
              <CardTitle>资源操作</CardTitle>
            </CardHeader>
            <CardContent className="grid gap-3">
              <SideLink count={requests.filter((item) => materialIds.has(item.materialId)).length} href="/materials/requests" icon={ClipboardList} label="申领记录" />
              <SideLink count={purchases.filter((item) => materialIds.has(item.materialId)).length} href="/materials/purchases" icon={ShoppingCart} label="申购记录" />
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <AlertTriangle className="h-5 w-5 text-amber-600" aria-hidden="true" />
                预警中心
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              {warningMaterials.length === 0 ? <p className="text-sm text-slate-500">暂无预警。</p> : null}
              {warningMaterials.map((item) => (
                <div className="break-words rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800" key={item.id}>
                  <p className="font-bold">
                    {materialStatusLabel(item.status)}：{item.name}
                  </p>
                  <p className="mt-1">
                    库存 {item.stock}
                    {item.unit}，告警线 {item.warningLine}
                    {item.unit}，有效期 {item.expiresAt || "未登记"}。
                  </p>
                </div>
              ))}
            </CardContent>
          </Card>
        </aside>
      </div>
    </AppShell>
  );
}

export async function AdminMaterialResourceManagementPage({
  productType,
  searchParams,
}: {
  productType: ResourceProductType;
  searchParams?: ResourcePageSearchParams;
}) {
  await requireAdminSection("materials");
  const params = (await searchParams) ?? {};
  const [materials, requests, purchases, damages, inventoryLedger, materialCategories] = await Promise.all([
    api.materials().catch(() => []),
    api.materialRequests().catch(() => []),
    api.materialPurchases().catch(() => []),
    api.materialDamages().catch(() => []),
    api.inventoryLedger().catch(() => []),
    api.materialCategories().catch(() => []),
  ]);
  const section = resourceTypeSection(productType);
  const typeMaterials = materials.filter((item) => item.productType === productType);
  const visibleMaterials = filterMaterials(materials, { ...params, productType });
  const materialIds = new Set(typeMaterials.map((item) => item.id));
  const typeRequests = requests.filter((item) => materialIds.has(item.materialId));
  const typePurchases = purchases.filter((item) => materialIds.has(item.materialId));
  const typeDamages = damages.filter((item) => materialIds.has(item.materialId));
  const typeLedger = inventoryLedger.filter((item) => materialIds.has(item.materialId));
  const warningMaterials = typeMaterials.filter((item) => ["near_expiry", "expired", "open_expired", "freeze_thaw_exceeded", "low", "damaged"].includes(item.status));
  const availableUnits = typeMaterials.reduce((sum, item) => sum + (item.units ?? []).filter((unit) => unit.status === "available").length, 0);
  const categories = primaryMaterialCategories(typeMaterials, materialCategories);

  return (
    <AdminShell active="materials" title={`${section.title}管理`} description={section.adminDescription}>
      <MaterialsNav active={resourceNavKeys[productType]} admin />

      <div className="mb-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <Metric label={`${section.title}总数`} value={typeMaterials.length} />
        <Metric label="可用编号" value={availableUnits} />
        <Metric label="待处理申领" value={typeRequests.filter((item) => item.status === "pending").length} />
        <Metric label="预警数量" value={warningMaterials.length} />
      </div>

      <div className="mb-6 grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <Card>
          <CardHeader>
            <CardTitle>新增入库</CardTitle>
          </CardHeader>
          <CardContent>
            <MaterialCreateForm categories={materialCategories} productType={productType} />
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>目录维护</CardTitle>
          </CardHeader>
          <CardContent>
            <MaterialCategoryForm categories={materialCategories} />
          </CardContent>
        </Card>
        <SideActionCard count={typeRequests.length} href="/admin/materials/requests" icon={ClipboardList} title="申领管理" />
        <SideActionCard count={typePurchases.length} href="/admin/materials/purchases" icon={ShoppingCart} title="申购管理" />
      </div>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_380px]">
        <div className="min-w-0 space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>{section.title}筛选</CardTitle>
            </CardHeader>
            <CardContent>
              <ResourceFilterForm action={`/admin/materials/${section.slug}`} categories={categories} params={params} />
            </CardContent>
          </Card>

          <AdminResourceMaterialSection
            categories={materialCategories}
            description={section.adminDescription}
            items={visibleMaterials}
            materials={typeMaterials}
            title={`${section.title}列表`}
            total={typeMaterials.length}
          />

          <Card>
            <CardHeader>
              <CardTitle>损毁登记</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              {typeDamages.length === 0 ? <p className="rounded-lg border p-4 text-sm text-slate-500">暂无{section.title}损毁登记。</p> : null}
              {typeDamages.slice(0, 10).map((item) => (
                <DamageCard item={item} key={item.id} />
              ))}
            </CardContent>
          </Card>
        </div>

        <aside className="min-w-0 space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>快捷操作</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <StockAdjustmentForm materials={typeMaterials} />
              <MaterialDamageForm materials={typeMaterials} />
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle>库存流水</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              {typeLedger.length === 0 ? <p className="text-sm text-slate-500">暂无库存流水。</p> : null}
              {typeLedger.slice(0, 12).map((entry) => (
                <LedgerCard entry={entry} key={entry.id} />
              ))}
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <AlertTriangle className="h-5 w-5 text-amber-600" aria-hidden="true" />
                预警中心
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              {warningMaterials.length === 0 ? <p className="text-sm text-slate-500">暂无预警。</p> : null}
              {warningMaterials.map((item) => (
                <div className="break-words rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800" key={item.id}>
                  <p className="font-bold">
                    {materialStatusLabel(item.status)}：{item.name}
                  </p>
                  <p className="mt-1">
                    库存 {item.stock}
                    {item.unit}，告警线 {item.warningLine}
                    {item.unit}，有效期 {item.expiresAt || "未登记"}。
                  </p>
                </div>
              ))}
            </CardContent>
          </Card>
        </aside>
      </div>
    </AdminShell>
  );
}

function ResourceFilterForm({ action, categories, params }: { action: string; categories: string[]; params: { q?: string; category?: string; status?: string } }) {
  return (
    <form action={action} className="grid w-full gap-2 sm:grid-cols-2 xl:flex xl:w-auto xl:flex-row">
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

function SideLink({ count, href, icon: Icon, label }: { count: number; href: string; icon: LucideIcon; label: string }) {
  return (
    <Link className="inline-flex min-h-11 items-center justify-between gap-3 rounded-md border px-3 py-2 text-sm font-bold text-slate-700 hover:bg-slate-50" href={href} prefetch={false}>
      <span className="inline-flex min-w-0 items-center gap-2">
        <Icon className="h-4 w-4 shrink-0 text-primary" aria-hidden="true" />
        <span className="truncate">{label}</span>
      </span>
      <span className="shrink-0 rounded bg-slate-100 px-2 py-1 text-xs">{count}</span>
    </Link>
  );
}

function SideActionCard({ count, href, icon: Icon, title }: { count: number; href: string; icon: LucideIcon; title: string }) {
  return (
    <Link className="flex min-w-0 items-center justify-between gap-4 rounded-lg border bg-white p-4 transition-colors hover:bg-slate-50" href={href} prefetch={false}>
      <div className="min-w-0">
        <p className="font-bold text-slate-900">{title}</p>
        <p className="mt-1 text-sm text-slate-500">当前资源类型关联 {count} 条记录。</p>
      </div>
      <Icon className="h-8 w-8 shrink-0 text-primary" aria-hidden="true" />
    </Link>
  );
}

function DamageCard({ item }: { item: MaterialDamage }) {
  return (
    <div className="rounded-lg border bg-white p-4 text-sm">
      <div className="flex flex-col justify-between gap-3 md:flex-row md:items-start">
        <div className="min-w-0">
          <p className="break-words font-bold text-primary">{item.materialName}</p>
          <p className="mt-1 text-xs text-slate-500">
            {item.reporter} / {item.groupName} / {formatDateTime(item.createdAt)}
          </p>
        </div>
        <span className="w-fit rounded bg-slate-100 px-2 py-1 text-xs font-bold text-slate-700">{damageStatusLabel(item.status)}</span>
      </div>
      <div className="mt-3 grid gap-3 md:grid-cols-3">
        <InfoItem label="唯一编号" value={item.unitCode || item.unitId || "未登记"} />
        <InfoItem label="批次" value={item.batchNo || item.batchId || "未登记"} />
        <InfoItem label="原因" value={item.reason} />
      </div>
      <div className="mt-4">
        <MaterialDamageActions canProcess canReview id={item.id} status={item.status} />
      </div>
    </div>
  );
}

function LedgerCard({ entry }: { entry: InventoryLedgerEntry }) {
  return (
    <div className="rounded-lg border p-3 text-sm">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="break-words font-bold">{entry.materialName}</p>
          <p className="mt-1 break-words text-xs text-slate-500">
            {entry.reason} / {formatDateTime(entry.createdAt)}
          </p>
        </div>
        <span className={entry.changeQty >= 0 ? "shrink-0 font-bold text-emerald-700" : "shrink-0 font-bold text-red-700"}>
          {entry.changeQty > 0 ? "+" : ""}
          {entry.changeQty}
        </span>
      </div>
    </div>
  );
}

function damageStatusLabel(status: string) {
  const labels: Record<string, string> = {
    pending: "待审核",
    approved: "已通过",
    rejected: "已拒绝",
    processed: "已处理",
    cancelled: "已取消",
  };
  return labels[status] ?? status;
}

function formatDateTime(value: string) {
  return new Date(value).toLocaleString("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    timeZone: "Asia/Shanghai",
  });
}
