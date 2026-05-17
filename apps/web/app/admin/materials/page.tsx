import Link from "next/link";
import { BarChart3, Boxes, ClipboardList, FlaskConical, PackageCheck, QrCode, ShoppingCart, TestTube2, type LucideIcon } from "lucide-react";
import { AdminShell, requireAdminSection } from "@/components/admin-shell";
import { MaterialCategoryForm, MaterialImportForm, MaterialsExportButton } from "@/components/material-management-form";
import { MaterialsNav } from "@/components/materials-nav";
import { resourceTypeSections, type ResourceProductType } from "@/components/material-resource-sections";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api, type Material } from "@/lib/api";

const resourceIcons = {
  standard: TestTube2,
  reagent: FlaskConical,
  consumable: Boxes,
} satisfies Record<ResourceProductType, typeof TestTube2>;

export default async function AdminMaterialsPage({ searchParams }: { searchParams?: Promise<{ q?: string }> }) {
  await requireAdminSection("materials");
  const params = (await searchParams) ?? {};
  const [materials, requests, purchases, damages, inventoryLedger, analytics, materialCategories] = await Promise.all([
    api.materials().catch(() => []),
    api.materialRequests().catch(() => []),
    api.materialPurchases().catch(() => []),
    api.materialDamages().catch(() => []),
    api.inventoryLedger().catch(() => []),
    api.materialAnalytics().catch(() => null),
    api.materialCategories().catch(() => []),
  ]);
  const warningMaterials = materials.filter((item) => ["near_expiry", "expired", "open_expired", "freeze_thaw_exceeded", "low", "damaged"].includes(item.status));
  const totalStock = materials.reduce((sum, item) => sum + item.stock, 0);
  const totalDamaged = materials.reduce((sum, item) => sum + item.damagedQuantity, 0);

  return (
    <AdminShell active="materials" title="资源管理" description="资源管理总览只维护入口、导出和一级/二级目录；标准品、试剂和耗材在各自独立页面管理。">
      <MaterialsNav active="overview" admin />

      <div className="mb-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-6">
        <Metric label="标准品" value={countByType(materials, "standard")} />
        <Metric label="试剂" value={countByType(materials, "reagent")} />
        <Metric label="耗材" value={countByType(materials, "consumable")} />
        <Metric label="库存总数" value={totalStock} />
        <Metric label="预警数量" value={warningMaterials.length} />
        <Metric label="损毁数量" value={totalDamaged} />
      </div>

      <div className="mb-6 grid gap-4 lg:grid-cols-3">
        {resourceTypeSections.map((section) => (
          <ResourceEntryCard items={materials.filter((item) => item.productType === section.type)} key={section.type} section={section} />
        ))}
      </div>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_380px]">
        <div className="min-w-0 space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>一级/二级目录维护</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="mb-4 text-sm text-slate-500">目录由试剂管理员在资源管理总览统一维护，新增或编辑资源时从这里选择一级目录和二级目录。</p>
              <MaterialCategoryForm categories={materialCategories} />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>数据导出与导入</CardTitle>
            </CardHeader>
            <CardContent className="flex flex-col gap-2 sm:flex-row sm:flex-wrap">
              <MaterialsExportButton filename="lirs-materials.csv" label="导出资源" path="/api/materials/export.csv" />
              <MaterialsExportButton filename="lirs-material-requests.csv" label="导出领用" path="/api/material-requests/export.csv" />
              <MaterialsExportButton filename="lirs-material-damages.csv" label="导出损毁" path="/api/material-damages/export.csv" />
              <MaterialsExportButton filename="lirs-materials-import-template.csv" label="下载模板" path="/api/materials/import-template.csv" />
              <MaterialImportForm />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <BarChart3 className="h-5 w-5 text-primary" aria-hidden="true" />
                统计分析
              </CardTitle>
            </CardHeader>
            <CardContent className="grid gap-4 lg:grid-cols-3">
              <AnalysisList title="月度消耗" rows={(analytics?.monthlyConsumption ?? []).slice(-6).map((item) => [`${item.month}`, `${item.quantity}`])} />
              <AnalysisList title="高频使用资源" rows={(analytics?.topConsumedMaterials ?? []).map((item) => [item.materialName, `${item.quantity}`])} />
              <AnalysisList title="损毁原因" rows={(analytics?.damageByReason ?? []).map((item) => [item.reason, `${item.quantity}`])} />
            </CardContent>
          </Card>
        </div>

        <aside className="min-w-0 space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>流程入口</CardTitle>
            </CardHeader>
            <CardContent className="grid gap-3">
              <SideLink count={requests.length} href="/admin/materials/requests" icon={ClipboardList} label="申领管理" />
              <SideLink count={purchases.length} href="/admin/materials/purchases" icon={ShoppingCart} label="申购管理" />
              <SideLink count={damages.length} href="/api/material-damages/export.csv" icon={PackageCheck} label="损毁导出" />
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle>扫码查询</CardTitle>
            </CardHeader>
            <CardContent>
              <form action="/admin/materials" className="flex min-w-0 items-center gap-3">
                <QrCode className="h-8 w-8 shrink-0 text-primary" aria-hidden="true" />
                <div className="min-w-0 flex-1">
                  <input className="h-10 w-full rounded-md border bg-white px-3 text-sm" defaultValue={params.q ?? ""} name="q" placeholder="输入二维码编码" />
                </div>
                <button className="h-10 rounded-md bg-primary px-3 text-sm font-bold text-white" type="submit">
                  查询
                </button>
              </form>
              {params.q ? <ScanResult materials={materials} query={params.q} /> : null}
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle>最近库存流水</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              {inventoryLedger.length === 0 ? <p className="text-sm text-slate-500">暂无库存流水。</p> : null}
              {inventoryLedger.slice(0, 8).map((entry) => (
                <div className="rounded-lg border p-3 text-sm" key={entry.id}>
                  <div className="flex items-start justify-between gap-3">
                    <div className="min-w-0">
                      <p className="break-words font-bold">{entry.materialName}</p>
                      <p className="mt-1 break-words text-xs text-slate-500">{entry.reason}</p>
                    </div>
                    <span className={entry.changeQty >= 0 ? "shrink-0 font-bold text-emerald-700" : "shrink-0 font-bold text-red-700"}>
                      {entry.changeQty > 0 ? "+" : ""}
                      {entry.changeQty}
                    </span>
                  </div>
                </div>
              ))}
            </CardContent>
          </Card>
        </aside>
      </div>
    </AdminShell>
  );
}

function ResourceEntryCard({
  items,
  section,
}: {
  items: Material[];
  section: (typeof resourceTypeSections)[number];
}) {
  const Icon = resourceIcons[section.type];
  const availableUnits = items.reduce((sum, item) => sum + (item.units ?? []).filter((unit) => unit.status === "available").length, 0);
  const warningCount = items.filter((item) => ["near_expiry", "expired", "open_expired", "freeze_thaw_exceeded", "low", "damaged"].includes(item.status)).length;

  return (
    <Link className="block min-w-0 rounded-lg border bg-white p-4 transition-colors hover:bg-slate-50" href={`/admin/materials/${section.slug}`}>
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0">
          <p className="text-lg font-bold text-slate-900">{section.title}管理</p>
          <p className="mt-1 text-sm leading-6 text-slate-500">{section.adminDescription}</p>
        </div>
        <Icon className="h-8 w-8 shrink-0 text-primary" aria-hidden="true" />
      </div>
      <div className="mt-4 grid grid-cols-3 gap-3 text-sm">
        <InfoMetric label="资源" value={items.length} />
        <InfoMetric label="可用编号" value={availableUnits} />
        <InfoMetric label="预警" value={warningCount} />
      </div>
    </Link>
  );
}

function ScanResult({ materials, query }: { materials: Material[]; query: string }) {
  const normalized = query.trim().toLowerCase();
  const item = materials.find((material) => material.qrCode.toLowerCase() === normalized || material.units.some((unit) => unit.unitCode.toLowerCase() === normalized));
  if (!item) {
    return <p className="mt-3 rounded-md border p-3 text-sm text-slate-500">未找到匹配资源。</p>;
  }
  return (
    <Link className="mt-3 block rounded-md border p-3 text-sm hover:bg-slate-50" href={`/admin/materials/${resourceTypeSections.find((section) => section.type === item.productType)?.slug ?? "consumables"}`} prefetch={false}>
      <p className="font-bold text-primary">{item.name}</p>
      <p className="mt-1 text-slate-500">
        {item.category}
        {item.subcategory ? ` / ${item.subcategory}` : ""} / 库存 {item.stock}
        {item.unit}
      </p>
    </Link>
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

function AnalysisList({ title, rows }: { title: string; rows: string[][] }) {
  return (
    <div className="rounded-lg border bg-white p-4">
      <p className="font-bold text-slate-900">{title}</p>
      <div className="mt-3 space-y-2 text-sm">
        {rows.length === 0 ? <p className="text-slate-500">暂无数据</p> : null}
        {rows.map((row) => (
          <div className="flex items-start justify-between gap-3" key={`${title}-${row[0]}`}>
            <span className="min-w-0 break-words text-slate-600">{row[0]}</span>
            <span className="shrink-0 font-bold text-slate-900">{row[1]}</span>
          </div>
        ))}
      </div>
    </div>
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

function InfoMetric({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-md bg-slate-50 p-2">
      <p className="text-xs text-slate-500">{label}</p>
      <p className="mt-1 font-bold text-slate-900">{value}</p>
    </div>
  );
}

function countByType(materials: Material[], productType: ResourceProductType) {
  return materials.filter((item) => item.productType === productType).length;
}
