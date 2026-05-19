import Link from "next/link";
import { notFound } from "next/navigation";
import { ArrowLeft, PackageSearch, ShoppingCart } from "lucide-react";
import { AppShell } from "@/components/app-shell";
import { MaterialQRCodeCard } from "@/components/material-qr-code-card";
import { MaterialRequestDialog } from "@/components/material-request-form";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api, Material } from "@/lib/api";

export default async function MaterialDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const [materials, requests, purchases] = await Promise.all([api.materials(), api.materialRequests(), api.materialPurchases()]);
  const material = materials.find((item) => item.id === id);
  if (!material) {
    notFound();
  }
  const materialRequests = requests.filter((item) => item.materialId === id);
  const materialPurchases = purchases.filter((item) => item.materialId === id);

  return (
    <AppShell>
      <Link className="mb-5 inline-flex items-center gap-2 text-sm text-slate-600 hover:text-primary" href={`/materials/${materialTypePath(material.productType)}`}>
        <ArrowLeft className="h-4 w-4" />
        返回分类目录
      </Link>

      <div className="mb-6 flex flex-col justify-between gap-4 lg:flex-row lg:items-end">
        <div className="min-w-0">
          <p className="text-xs font-bold uppercase tracking-widest text-primary">资源详情</p>
          <h1 className="mt-2 break-words text-2xl font-bold text-slate-900 sm:text-3xl">{material.name}</h1>
          <p className="mt-2 max-w-3xl text-sm leading-6 text-muted-foreground">
            {productTypeLabel(material.productType)} / {material.category}{material.subcategory ? ` / ${material.subcategory}` : ""} / {material.spec} / {material.supplier || "未登记供应商"}
          </p>
        </div>
        <div className="grid gap-2 sm:flex sm:items-center">
          <MaterialRequestDialog buttonClassName="w-full sm:w-auto" material={material} />
          <Button asChild className="w-full sm:w-auto" variant="outline">
            <Link href={`/materials/purchases/new?materialId=${material.id}`}>
              <ShoppingCart className="h-4 w-4" aria-hidden="true" />
              申购
            </Link>
          </Button>
        </div>
      </div>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_360px]">
        <div className="min-w-0 space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <PackageSearch className="h-5 w-5 text-primary" />
                基础资料
              </CardTitle>
            </CardHeader>
            <CardContent className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
              <Info label="资源类型" value={productTypeLabel(material.productType)} />
              <Info label="目录" value={`${material.category}${material.subcategory ? ` / ${material.subcategory}` : ""}`} />
              <Info label="货号" value={material.catalogNo} />
              <Info label="CAS号" value={material.casNo} />
              <Info label="级别/浓度" value={`${material.grade || "未登记"} / ${material.concentration || "未登记"}`} />
              <Info label="规格" value={material.spec} />
              <Info label="单位" value={material.unit} />
              <Info label="保存条件" value={material.storageCondition} />
              <Info label="库位" value={materialLocation(material)} />
              <Info label="批次号" value={material.batchNo} />
              <Info label="有效期" value={material.expiresAt} />
              <Info label="开封日期" value={material.openedAt} />
              <Info label="开封到期" value={material.openExpiresAt} />
              <Info label="冻融次数" value={`${material.freezeThawCount}/${material.freezeThawLimit || "不限"}`} />
              {material.productType !== "standard" ? <Info label="母液/来源" value={material.parentMaterialName || material.parentMaterialId} /> : null}
              {material.productType !== "standard" ? <Info label="稀释倍数" value={material.dilutionFactor} /> : null}
              {material.productType !== "standard" ? <Info label="配制方法" value={material.preparationMethod} /> : null}
              <Info label="二维码" value={material.qrCode} />
              <Info label="审批策略" value={material.approvalRequired ? "申领需要审批" : "默认免审"} />
              <Info label="单价" value={`¥${material.unitPrice.toFixed(2)}`} />
              <Info label="采购项目名称及编号" value={material.tenderContract || material.contractNo} />
              <Info label="状态" value={materialStatusLabel(material.status)} />
              <Info label="备注" value={material.remark} />
              <Info label="资源证书" value={material.certificateUrl} />
              <Info label="标准品/标准物质证书" value={material.standardCertificateUrl} />
            </CardContent>
          </Card>

          {(material.batches ?? []).length > 0 ? (
            <Card>
              <CardHeader>
                <CardTitle>批次与编号</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                {(material.batches ?? []).filter((batch) => batch.status !== "disabled").map((batch) => (
                  <RecordLine
                    key={batch.id}
                    left={`${batch.batchNo} / ${batch.quantity}${material.unit}`}
                    right={batchStatusLabel(batch.status)}
                    sub={[
                      `${batch.expiresAt || "未登记有效期"} / ${batch.location || "未登记库位"}`,
                      ...(batch.units ?? []).map((unit) => `${unit.unitCode} / ${unitStatusLabel(unit.status)}${unit.expiresAt ? ` / ${unit.expiresAt}` : ""}${unit.location ? ` / ${unit.location}` : ""}`),
                    ].join("\n")}
                  />
                ))}
                {(material.batches ?? []).filter((batch) => batch.status !== "disabled").length === 0 ? <p className="text-sm text-slate-500">暂无批次编号。</p> : null}
              </CardContent>
            </Card>
          ) : null}

          {(material.units ?? []).length > 0 ? (
            <Card>
              <CardHeader>
                <CardTitle>最小单位编号</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                {(material.units ?? []).slice(0, 60).map((unit) => (
                  <RecordLine
                    key={unit.id}
                    left={unit.unitCode}
                    right={unitStatusLabel(unit.status)}
                    sub={`${unit.batchNo ? `批次 ${unit.batchNo} / ` : ""}${unit.expiresAt || "未登记有效期"} / ${unit.location || "未登记库位"}`}
                  />
                ))}
              </CardContent>
            </Card>
          ) : null}

          <Card>
            <CardHeader>
              <CardTitle>申领记录</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              {materialRequests.slice(0, 8).map((item) => (
                <RecordLine key={item.id} left={`${item.requester} / ${item.quantity}${material.unit}`} right={requestStatusLabel(item.status)} sub={`${item.unitCode ? `编号 ${item.unitCode} / ` : ""}${item.batchNo ? `批次 ${item.batchNo} / ` : ""}${item.purpose}`} />
              ))}
              {materialRequests.length === 0 ? <p className="text-sm text-slate-500">暂无申领记录。</p> : null}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>申购记录</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              {materialPurchases.slice(0, 8).map((item) => (
                <RecordLine key={item.id} left={`${item.requester} / ${item.quantity}${material.unit}`} right={purchaseStatusLabel(item.status)} sub={item.reason} />
              ))}
              {materialPurchases.length === 0 ? <p className="text-sm text-slate-500">暂无申购记录。</p> : null}
            </CardContent>
          </Card>
        </div>

        <aside className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>库存状态</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <MaterialQRCodeCard
                materialLocation={materialLocation(material)}
                materialName={material.name}
                materialSpec={material.spec}
                qrCode={material.qrCode}
              />
              <div className="rounded-lg border bg-slate-50 p-4">
                <p className="text-sm text-slate-500">当前库存</p>
                <p className="mt-2 text-3xl font-bold">
                  {material.stock}
                  {material.unit}
                </p>
              </div>
              <div className={`rounded-lg border p-4 text-sm ${material.stock <= material.warningLine ? "border-amber-200 bg-amber-50 text-amber-800" : "border-emerald-200 bg-emerald-50 text-emerald-800"}`}>
                库存告警线：{material.warningLine}
                {material.unit}。{material.stock <= material.warningLine ? "当前已触发低库存预警。" : "当前库存高于告警线。"}
              </div>
              {material.damagedQuantity > 0 ? (
                <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-800">
                  已处理损毁数量：{material.damagedQuantity}
                  {material.unit}。
                </div>
              ) : null}
            </CardContent>
          </Card>
        </aside>
      </div>
    </AppShell>
  );
}

function Info({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border bg-slate-50/40 p-4">
      <p className="text-xs text-slate-500">{label}</p>
      <p className="mt-2 break-words text-sm font-bold text-slate-900">{value || "未登记"}</p>
    </div>
  );
}

function RecordLine({ left, right, sub }: { left: string; right: string; sub: string }) {
  return (
    <div className="rounded-lg border p-3 text-sm">
      <div className="flex flex-col justify-between gap-2 sm:flex-row sm:items-center">
        <p className="font-bold text-slate-900">{left}</p>
        <span className="w-fit rounded bg-slate-100 px-2 py-1 text-xs font-bold text-slate-600">{right}</span>
      </div>
      <p className="mt-2 whitespace-pre-line break-words text-slate-500">{sub}</p>
    </div>
  );
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

function productTypeLabel(productType: string) {
  const labels: Record<string, string> = {
    consumable: "耗材",
    reagent: "试剂",
    standard: "标准品/标准物质",
  };
  return labels[productType] ?? productType;
}

function materialTypePath(productType: string) {
  if (productType === "standard") {
    return "standards";
  }
  if (productType === "reagent") {
    return "reagents";
  }
  return "consumables";
}

function materialLocation(item: Pick<Material, "storageRoom" | "storageCabinet" | "storageLayer" | "storageSlot">) {
  return [item.storageRoom, item.storageCabinet, item.storageLayer, item.storageSlot].filter(Boolean).join(" / ");
}

function requestStatusLabel(status: string) {
  const labels: Record<string, string> = {
    pending: "待审批",
    approved: "已通过",
    rejected: "已拒绝",
    outbound: "已出库",
    cancelled: "已取消",
  };
  return labels[status] ?? status;
}

function purchaseStatusLabel(status: string) {
  const labels: Record<string, string> = {
    pending: "待审批",
    approved: "已通过",
    rejected: "已拒绝",
    ordered: "已下单",
    received: "已入库",
    cancelled: "已取消",
  };
  return labels[status] ?? status;
}

function batchStatusLabel(status: string) {
  const labels: Record<string, string> = {
    active: "可用",
    depleted: "已用尽",
    disabled: "停用",
  };
  return labels[status] ?? status;
}

function unitStatusLabel(status: string) {
  const labels: Record<string, string> = {
    available: "可领用",
    reserved: "已预留",
    used: "已领用",
    damaged: "已损毁",
    disabled: "停用",
  };
  return labels[status] ?? status;
}
