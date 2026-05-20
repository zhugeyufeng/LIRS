import Link from "next/link";
import { MaterialDamageForm, MaterialEditForm, StockAdjustmentForm } from "@/components/material-management-form";
import { MaterialRequestDialog } from "@/components/material-request-form";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Material, MaterialCategory } from "@/lib/api";

export type ResourceProductType = "standard" | "reagent" | "consumable";

export const resourceTypeSections: { type: ResourceProductType; slug: string; title: string; description: string; adminDescription: string }[] = [
  {
    type: "standard",
    slug: "standards",
    title: "标准品/标准物质",
    description: "标准品/标准物质按批次和唯一编号独立展示，可直接选择可用编号申领。",
    adminDescription: "标准品/标准物质按批次、唯一编号、有效期、库位和证书独立维护。",
  },
  {
    type: "reagent",
    slug: "reagents",
    title: "试剂",
    description: "试剂按目录、批次、库位和库存状态独立展示。",
    adminDescription: "试剂按目录、批次、库位和库存预警独立维护。",
  },
  {
    type: "consumable",
    slug: "consumables",
    title: "耗材",
    description: "耗材按最小单位、库位和预警状态独立展示。",
    adminDescription: "耗材按最小单位、库位、库存和损毁闭环独立维护。",
  },
];

export function resourceTypeSection(type: string) {
  return resourceTypeSections.find((section) => section.type === type) ?? resourceTypeSections[0];
}

export function resourceTypeFromSlug(slug: string): ResourceProductType {
  return (resourceTypeSections.find((section) => section.slug === slug) ?? resourceTypeSections[0]).type;
}

export function filterMaterials(
  materials: Material[],
  params: { query?: string; q?: string; category?: string; productType?: string; status?: string },
) {
  const query = (params.query ?? params.q ?? "").trim().toLowerCase();
  return materials.filter((item) => {
    const matchesSearch = query === "" || materialSearchValues(item).some((value) => value.toLowerCase().includes(query));
    const matchesCategory = !params.category || item.category === params.category;
    const matchesProductType = !params.productType || item.productType === params.productType;
    const matchesStatus = !params.status || item.status === params.status;
    return matchesSearch && matchesCategory && matchesProductType && matchesStatus;
  });
}

function materialSearchValues(item: Material) {
  return [
    item.name,
    item.category,
    item.subcategory,
    item.spec,
    item.unit,
    item.supplier,
    item.manufacturer,
    item.batchNo,
    item.catalogNo,
    item.casNo,
    item.grade,
    item.concentration,
    item.preparationMethod,
    item.storageCondition,
    item.tenderContract,
    item.contractNo,
    item.remark,
    item.certificateUrl,
    item.standardCertificateUrl,
    item.attachmentUrl,
    item.qrCode,
    item.expiresAt,
    item.openedAt,
    materialLocation(item),
    materialBatchSummary(item),
    materialUnitCodes(item),
    ...(item.batches ?? []).flatMap((batch) => [batch.batchNo, batch.expiresAt, batch.location]),
    ...(item.units ?? []).flatMap((unit) => [unit.unitCode, unit.batchNo, unit.expiresAt, unit.location]),
  ].map((value) => String(value ?? ""));
}

export function primaryMaterialCategories(materials: Material[], materialCategories: MaterialCategory[]) {
  return Array.from(
    new Set([
      ...materialCategories.filter((item) => item.status === "active" && item.parentName.trim() === "").map((item) => item.name),
      ...materials.map((item) => item.category).filter(Boolean),
    ]),
  ).sort((a, b) => a.localeCompare(b, "zh-CN"));
}

export function ResourceMaterialSection({ title, description, items, total }: { title: string; description: string; items: Material[]; total: number }) {
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
                <InfoItem label="采购项目名称及编号" value={materialProcurementProject(item)} />
                <InfoItem label="备注" value={item.remark} />
                {item.productType !== "standard" ? <InfoItem label="来源/稀释" value={`${item.parentMaterialName || "无"} / ${item.dilutionFactor || "未登记"}`} /> : null}
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
                    <p className="mt-1">采购项目：{materialProcurementProject(item) || "未登记"}</p>
                    {item.remark ? <p className="mt-1">备注：{item.remark}</p> : null}
                    <p className="mt-1">开封：{item.openedAt || "未开封"}{item.openExpiresAt ? ` / 到期 ${item.openExpiresAt}` : ""}</p>
                    {item.productType !== "standard" && (item.parentMaterialName || item.dilutionFactor) ? <p className="mt-1">来源：{item.parentMaterialName || item.parentMaterialId || "未登记"} / {item.dilutionFactor || "未登记"}</p> : null}
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

export function AdminResourceMaterialSection({
  title,
  description,
  items,
  total,
  materials,
  categories,
}: {
  title: string;
  description: string;
  items: Material[];
  total: number;
  materials: Material[];
  categories: MaterialCategory[];
}) {
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
            <div className="rounded-lg border bg-white p-4" key={item.id}>
              <div className="flex flex-col justify-between gap-3 sm:flex-row sm:items-start">
                <div className="min-w-0">
                  <p className="break-words font-bold text-primary">{item.name}</p>
                  <p className="mt-1 break-words text-xs text-slate-500">
                    {item.category}{item.subcategory ? ` / ${item.subcategory}` : ""} / {item.supplier || "未登记供应商"}
                  </p>
                </div>
                <StatusPill status={item.status} />
              </div>
              <div className="mt-4 grid gap-3 text-sm sm:grid-cols-2">
                <InfoItem label="规格" value={item.spec} />
                <InfoItem label="CAS/级别" value={`${item.casNo || "未登记"} / ${item.grade || "未登记"}`} />
                <InfoItem label="货号" value={item.catalogNo} />
                <InfoItem label="编号/批次" value={materialBatchSummary(item)} />
                <InfoItem label="库位" value={materialLocation(item)} />
                <InfoItem label="开封/冻融" value={`${item.openedAt || "未开封"} / ${item.freezeThawCount}/${item.freezeThawLimit || "不限"}`} />
                <InfoItem label="库存告警线" value={`${item.warningLine}${item.unit}`} />
                <InfoItem label="采购项目名称及编号" value={materialProcurementProject(item)} />
                <InfoItem label="备注" value={item.remark} />
                {item.productType !== "standard" ? <InfoItem label="来源/稀释" value={`${item.parentMaterialName || "无"} / ${item.dilutionFactor || "未登记"}`} /> : null}
                <InfoItem label="单价" value={`¥${item.unitPrice.toFixed(2)}`} />
                <InfoItem label="二维码" value={item.qrCode} />
              </div>
              <div className="mt-4 flex flex-col gap-2 sm:flex-row">
                <MaterialRequestDialog buttonClassName="w-full sm:w-auto" material={item} />
                <MaterialEditForm categories={categories} material={item} />
                <StockAdjustmentForm material={item} materials={materials} />
                <MaterialDamageForm material={item} materials={materials} />
              </div>
            </div>
          ))}
        </div>

        <div className="hidden overflow-x-auto rounded-lg border xl:block">
          <table className="w-full table-fixed text-left text-sm">
            <thead className="bg-slate-50 text-slate-500">
              <tr>
                <th className="p-3">名称</th>
                <th className="p-3">规格</th>
                <th className="p-3">库存</th>
                <th className="p-3">编号/批次</th>
                <th className="p-3">状态</th>
                <th className="p-3 text-right">单价</th>
                <th className="w-52 p-3">操作</th>
              </tr>
            </thead>
            <tbody className="divide-y">
              {items.map((item) => (
                <tr key={item.id}>
                  <td className="p-3 align-top">
                    <p className="break-words font-bold text-primary">{item.name}</p>
                    <p className="break-words text-xs text-slate-500">
                      {item.category}{item.subcategory ? ` / ${item.subcategory}` : ""} / {item.supplier || "未登记供应商"}
                    </p>
                    <p className="mt-1 break-words text-xs text-slate-500">CAS：{item.casNo || "未登记"} / 货号：{item.catalogNo || "未登记"}</p>
                  </td>
                  <td className="break-words p-3 align-top">
                    <p>{item.spec}</p>
                    <p className="mt-1 text-xs text-slate-500">{item.storageCondition || "未登记保存条件"}</p>
                  </td>
                  <td className="p-3 align-top">
                    <span className={item.stock <= item.warningLine ? "font-bold text-amber-700" : "font-bold"}>
                      {item.stock}
                      {item.unit}
                    </span>
                    {item.damagedQuantity > 0 ? <p className="mt-1 text-xs text-red-700">损毁 {item.damagedQuantity}{item.unit}</p> : null}
                  </td>
                  <td className="break-words p-3 align-top text-xs text-slate-500">
                    <p>{materialBatchSummary(item)}</p>
                    <p className="mt-1">{materialLocation(item) || "未登记库位"}</p>
                    <p className="mt-1">采购项目：{materialProcurementProject(item) || "未登记"}</p>
                    {item.remark ? <p className="mt-1">备注：{item.remark}</p> : null}
                    <p className="mt-1">开封：{item.openedAt || "未开封"}{item.openExpiresAt ? ` / 到期 ${item.openExpiresAt}` : ""}</p>
                    <p className="mt-1">冻融：{item.freezeThawCount}/{item.freezeThawLimit || "不限"}</p>
                    {item.productType !== "standard" && (item.parentMaterialName || item.dilutionFactor) ? <p className="mt-1">来源：{item.parentMaterialName || item.parentMaterialId || "未登记"} / {item.dilutionFactor || "未登记"}</p> : null}
                  </td>
                  <td className="p-3 align-top"><StatusPill status={item.status} /></td>
                  <td className="whitespace-nowrap p-3 text-right align-top font-bold">¥{item.unitPrice.toFixed(2)}</td>
                  <td className="p-3 align-top">
                    <div className="flex flex-col gap-2">
                      <MaterialRequestDialog buttonClassName="h-8 w-full" material={item} />
                      <MaterialEditForm categories={categories} material={item} />
                      <StockAdjustmentForm material={item} materials={materials} />
                      <MaterialDamageForm material={item} materials={materials} />
                    </div>
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

export function InfoItem({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-xs text-slate-500">{label}</p>
      <p className="mt-1 break-words font-medium text-slate-800">{value || "未登记"}</p>
    </div>
  );
}

export function StatusPill({ status }: { status: string }) {
  const warning = ["near_expiry", "low", "open_expired", "freeze_thaw_exceeded", "damaged"].includes(status);
  const danger = ["expired", "disabled"].includes(status);
  const className = danger
    ? "w-fit shrink-0 rounded bg-red-50 px-2 py-1 text-xs font-bold text-red-700"
    : warning
      ? "w-fit shrink-0 rounded bg-amber-50 px-2 py-1 text-xs font-bold text-amber-700"
      : "w-fit shrink-0 rounded bg-emerald-50 px-2 py-1 text-xs font-bold text-emerald-700";
  return <span className={className}>{materialStatusLabel(status)}</span>;
}

export function materialLocation(item: Pick<Material, "storageRoom" | "storageCabinet" | "storageLayer" | "storageSlot">) {
  return [item.storageRoom, item.storageCabinet, item.storageLayer, item.storageSlot].filter(Boolean).join(" / ");
}

export function materialBatchSummary(item: Material) {
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

export function materialUnitCodes(item: Material) {
  return (item.units ?? []).map((unit) => unit.unitCode).join(" ");
}

function materialProcurementProject(item: Material) {
  return item.tenderContract || item.contractNo;
}

export function materialStatusLabel(status: string) {
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
