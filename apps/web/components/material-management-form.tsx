"use client";

import { type DragEvent, FormEvent, startTransition, useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { AlertTriangle, Download, GripVertical, PackagePlus, Pencil, Save, SlidersHorizontal, Trash2, Upload } from "lucide-react";
import { browserDelete, browserPatch, browserPost, Material, MaterialAlertAction, MaterialAlertActionPayload, MaterialCategory, MaterialCategoryPayload, MaterialDamage, MaterialDamagePayload, MaterialImportResult, MaterialPayload, PurchasableMaterial, StockAdjustmentPayload } from "@/lib/api";
import { AdminDialog } from "@/components/admin-dialog";
import { Button } from "@/components/ui/button";

type MaterialProductType = "standard" | "reagent" | "consumable";

export function MaterialCreateForm({ categories = [], productType, purchasableMaterials = [] }: { categories?: MaterialCategory[]; productType?: MaterialProductType; purchasableMaterials?: PurchasableMaterial[] }) {
  const router = useRouter();
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>, close?: () => void) {
    event.preventDefault();
    setPending(true);
    setMessage("");
    const formElement = event.currentTarget;
    const payload = materialPayload(new FormData(formElement), "normal");
    try {
      const material = await browserPost<Material>("/api/materials", payload);
      setMessage(`已入库：${material.name}，库存 ${material.stock}${material.unit}`);
      formElement.reset();
      close?.();
      startTransition(() => {
        router.refresh();
      });
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "入库失败");
    } finally {
      setPending(false);
    }
  }

  return (
    <div className="space-y-3">
      <AdminDialog
        description="新增资源会写入标准品/标准物质、试剂或耗材目录，并生成初始库存流水；可登记批次、CAS、库位、证书、开封和审批策略。"
        maxWidth="max-w-4xl"
        title="新增资源入库"
        trigger={
          <Button className="w-full">
            <PackagePlus className="h-4 w-4" aria-hidden="true" />
            新增资源
          </Button>
        }
      >
        {(close) => (
          <form className="space-y-4" onSubmit={(event) => submit(event, close)}>
            <MaterialFields categories={categories} productType={productType} purchasableMaterials={purchasableMaterials} />
            <div className="flex justify-end">
              <Button className="w-full sm:w-auto" disabled={pending} type="submit">
                <Save className="h-4 w-4" aria-hidden="true" />
                {pending ? "保存中..." : "新增入库"}
              </Button>
            </div>
          </form>
        )}
      </AdminDialog>
      {message ? <p className="text-sm text-slate-500">{message}</p> : null}
    </div>
  );
}

export function StockAdjustmentForm({ material, materials }: { material?: Material; materials: Material[] }) {
  const router = useRouter();
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);
  const [selectedMaterialId, setSelectedMaterialId] = useState(material?.id ?? "");
  const selectedMaterial = material ?? materials.find((item) => item.id === selectedMaterialId);
  const isStandard = selectedMaterial?.productType === "standard";
  const availableBatches = isStandard ? (selectedMaterial.batches ?? []).filter((batch) => batch.status === "active" && batch.quantity > 0) : [];
  const availableUnits = (selectedMaterial?.units ?? []).filter((unit) => unit.status === "available");

  async function submit(event: FormEvent<HTMLFormElement>, close?: () => void) {
    event.preventDefault();
    setPending(true);
    setMessage("");
    const formElement = event.currentTarget;
    const form = new FormData(formElement);
    const materialId = material?.id ?? String(form.get("materialId") ?? "");
    const payload: StockAdjustmentPayload = {
      changeQty: Number(form.get("changeQty") ?? 0),
      reason: String(form.get("reason") ?? ""),
      batchId: String(form.get("batchId") ?? ""),
      batchNo: String(form.get("batchNo") ?? ""),
      unitId: String(form.get("unitId") ?? ""),
      expiresAt: String(form.get("expiresAt") ?? ""),
      location: String(form.get("location") ?? ""),
    };
    try {
      const updated = await browserPost<Material>(`/api/materials/${materialId}/stock-adjustments`, payload);
      setMessage(`库存已更新：${updated.name} ${updated.stock}${updated.unit}`);
      formElement.reset();
      setSelectedMaterialId(material?.id ?? "");
      close?.();
      startTransition(() => {
        router.refresh();
      });
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "库存调整失败");
    } finally {
      setPending(false);
    }
  }

  return (
    <div className="space-y-2">
      <AdminDialog
        description="库存调整会写入库存流水。入库为正数，出库为负数。"
        title={material ? `调整库存：${material.name}` : "调整库存"}
        trigger={
          <Button className="w-full sm:w-auto" disabled={pending || materials.length === 0} size="sm" variant="outline">
            <SlidersHorizontal className="h-4 w-4" aria-hidden="true" />
            调整库存
          </Button>
        }
      >
        {(close) => (
          <form className="space-y-4" onSubmit={(event) => submit(event, close)}>
            {material ? (
              <div className="rounded-md border bg-slate-50 px-3 py-2 text-sm">
                <p className="text-xs text-slate-500">当前资源</p>
                <p className="mt-1 font-bold text-slate-900">
                  {material.name} / 当前 {material.stock}
                  {material.unit}
                </p>
              </div>
            ) : (
              <select className="h-10 w-full rounded-md border bg-white px-3 text-sm" name="materialId" onChange={(event) => setSelectedMaterialId(event.target.value)} required value={selectedMaterialId}>
                <option value="">选择资源</option>
                {materials.map((item) => (
                  <option key={item.id} value={item.id}>
                    {item.name} / 当前 {item.stock}
                    {item.unit}
                  </option>
                ))}
              </select>
            )}
            {isStandard ? (
              <div className="grid gap-3 md:grid-cols-2">
                <label className="block space-y-2">
                  <span className="text-sm font-medium">已有批次</span>
                  <select className="h-10 w-full rounded-md border bg-white px-3 text-sm" name="batchId">
                    <option value="">新批次入库</option>
                    {availableBatches.map((batch) => (
                      <option key={batch.id} value={batch.id}>
                        {batch.batchNo} / 当前 {batch.quantity}
                        {selectedMaterial?.unit ?? ""}
                      </option>
                    ))}
                  </select>
                  <FieldHint value="负数出库必须选择已有批次；正数入库可选择已有批次或填写新批次号。" />
                </label>
                <Field label="新批次号" name="batchNo" placeholder="正数入库且未选已有批次时填写" />
                <Field label="批次有效期" name="expiresAt" placeholder="选择该批次有效期" type="date" />
                <Field defaultValue={materialBatchLocation(selectedMaterial)} label="批次库位" name="location" placeholder="填写该批次库位" />
              </div>
            ) : null}
            {selectedMaterial ? (
              <label className="block space-y-2">
                <span className="text-sm font-medium">出库编号</span>
                <select className="h-10 w-full rounded-md border bg-white px-3 text-sm" name="unitId">
                  <option value="">负数出库可自动按有效期扣减</option>
                  {availableUnits.map((unit) => (
                    <option key={unit.id} value={unit.id}>
                      {unit.unitCode}
                      {unit.batchNo ? ` / 批次 ${unit.batchNo}` : ""}
                      {unit.expiresAt ? ` / 有效期 ${unit.expiresAt}` : ""}
                      {unit.location ? ` / ${unit.location}` : ""}
                    </option>
                  ))}
                </select>
                <FieldHint value="选择具体编号时，库存变动数量必须为 -1；不选择编号时会按有效期优先自动扣减。" />
              </label>
            ) : null}
            <Field label="库存变动数量" name="changeQty" placeholder="入库为正数，出库为负数" required type="number" />
            <Field label="调整原因" name="reason" placeholder="填写本次库存调整原因" required />
            <div className="flex justify-end">
              <Button className="w-full sm:w-auto" disabled={pending} type="submit" variant="outline">
                {pending ? "调整中..." : "调整库存"}
              </Button>
            </div>
          </form>
        )}
      </AdminDialog>
      {message ? <p className="text-xs text-slate-500">{message}</p> : null}
    </div>
  );
}

export function MaterialEditForm({ material, categories = [] }: { material: Material; categories?: MaterialCategory[] }) {
  const router = useRouter();
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>, close?: () => void) {
    event.preventDefault();
    setPending(true);
    setMessage("");
    const payload = materialPayload(new FormData(event.currentTarget), material.status);
    try {
      const updated = await browserPatch<Material>(`/api/materials/${material.id}`, payload);
      setMessage(`已保存：${updated.name} / ${statusLabel(updated.status)}`);
      close?.();
      startTransition(() => {
        router.refresh();
      });
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "保存失败");
    } finally {
      setPending(false);
    }
  }

  async function deleteMaterial(close?: () => void) {
    if (!confirm(`确定删除“${material.name}”吗？删除后该资源会从默认列表和申领入口移除。`)) {
      return;
    }
    if (!confirm("请再次确认删除资源。历史库存流水、申领和损毁记录会保留。")) {
      return;
    }
    setPending(true);
    setMessage("");
    try {
      const deleted = await browserDelete<Material>(`/api/materials/${material.id}`);
      setMessage(`已删除：${deleted.name}`);
      close?.();
      startTransition(() => {
        router.refresh();
      });
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "删除失败");
    } finally {
      setPending(false);
    }
  }

  return (
    <div className="space-y-2">
      <AdminDialog
        description="修改资料不会直接改变库存数量；库存、损毁和出入库数量通过流水动作处理。"
        maxWidth="max-w-4xl"
        title={`修改产品：${material.name}`}
        trigger={
          <Button className="w-full sm:w-auto" disabled={pending} size="sm" variant="outline">
            <Pencil className="h-4 w-4" aria-hidden="true" />
            修改
          </Button>
        }
      >
        {(close) => (
          <form className="space-y-4" onSubmit={(event) => submit(event, close)}>
            <MaterialFields categories={categories} material={material} />
            <div className="flex flex-col gap-2 sm:flex-row sm:justify-end">
              <Button className="w-full sm:w-auto" disabled={pending} type="submit" variant="outline">
                <Save className="h-4 w-4" aria-hidden="true" />
                {pending ? "保存中..." : "保存资源"}
              </Button>
              <Button className="w-full sm:w-auto" disabled={pending} onClick={() => deleteMaterial(close)} type="button" variant="destructive">
                <Trash2 className="h-4 w-4" aria-hidden="true" />
                删除资源
              </Button>
            </div>
          </form>
        )}
      </AdminDialog>
      {message ? <p className="text-xs text-slate-500">{message}</p> : null}
    </div>
  );
}

export function MaterialDamageForm({ material, materials }: { material?: Material; materials: Material[] }) {
  const router = useRouter();
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);
  const [selectedMaterialId, setSelectedMaterialId] = useState(material?.id ?? "");
  const selectedMaterial = material ?? materials.find((item) => item.id === selectedMaterialId);
  const availableUnits = (selectedMaterial?.units ?? []).filter((unit) => unit.status === "available");

  async function submit(event: FormEvent<HTMLFormElement>, close?: () => void) {
    event.preventDefault();
    setPending(true);
    setMessage("");
    const formElement = event.currentTarget;
    const form = new FormData(formElement);
    const materialId = material?.id ?? String(form.get("materialId") ?? "");
    const payload: MaterialDamagePayload = {
      materialId,
      unitId: String(form.get("unitId") ?? ""),
      quantity: 1,
      reason: String(form.get("reason") ?? ""),
      photoUrl: String(form.get("photoUrl") ?? ""),
      attachmentUrl: String(form.get("attachmentUrl") ?? ""),
    };
    try {
      const item = await browserPost<MaterialDamage>("/api/material-damages", payload);
      setMessage(`损毁已登记：${item.materialName} / ${item.unitCode || "未返回编号"}`);
      formElement.reset();
      setSelectedMaterialId(material?.id ?? "");
      close?.();
      startTransition(() => {
        router.refresh();
      });
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "损毁登记失败");
    } finally {
      setPending(false);
    }
  }

  return (
    <div className="space-y-2">
      <AdminDialog
        description="损毁登记必须选择唯一编号；通过后由试剂管理员处理并按该编号扣减库存、写入损毁流水。"
        title={material ? `登记损毁：${material.name}` : "登记损毁"}
        trigger={
          <Button className="w-full sm:w-auto" disabled={pending || materials.length === 0} size="sm" variant="outline">
            <AlertTriangle className="h-4 w-4" aria-hidden="true" />
            损毁登记
          </Button>
        }
      >
        {(close) => (
          <form className="space-y-4" onSubmit={(event) => submit(event, close)}>
            {material ? (
              <div className="rounded-md border bg-slate-50 px-3 py-2 text-sm">
                <p className="text-xs text-slate-500">当前资源</p>
                <p className="mt-1 font-bold text-slate-900">
                  {material.name} / 库存 {material.stock}
                  {material.unit}
                </p>
              </div>
            ) : (
              <select className="h-10 w-full rounded-md border bg-white px-3 text-sm" name="materialId" onChange={(event) => setSelectedMaterialId(event.target.value)} required value={selectedMaterialId}>
                <option value="">选择资源</option>
                {materials.map((item) => (
                  <option key={item.id} value={item.id}>
                    {item.name} / 当前 {item.stock}
                    {item.unit}
                  </option>
                ))}
              </select>
            )}
            {selectedMaterial ? (
              <label className="block space-y-2">
                <span className="text-sm font-medium">损毁编号</span>
                <select className="h-10 w-full rounded-md border bg-white px-3 text-sm" disabled={availableUnits.length === 0} name="unitId" required>
                  <option value="">选择唯一编号</option>
                  {availableUnits.map((unit) => (
                    <option key={unit.id} value={unit.id}>
                      {unit.unitCode}
                      {unit.batchNo ? ` / 批次 ${unit.batchNo}` : ""}
                      {unit.expiresAt ? ` / 有效期 ${unit.expiresAt}` : ""}
                      {unit.location ? ` / ${unit.location}` : ""}
                    </option>
                  ))}
                </select>
                <span className="block text-xs text-slate-500">{availableUnits.length === 0 ? "暂无可登记损毁的可用编号。" : `该资源共有 ${availableUnits.length} 个可登记损毁编号，单次登记 1 个最小单位。`}</span>
              </label>
            ) : null}
            <input name="quantity" type="hidden" value="1" />
            <Field label="损毁原因" name="reason" placeholder="填写损毁原因" required />
            <div className="grid gap-3 md:grid-cols-2">
              <Field label="照片地址" name="photoUrl" placeholder="填写损毁照片地址" />
              <Field label="说明文件地址" name="attachmentUrl" placeholder="填写说明文件地址" />
            </div>
            <div className="flex justify-end">
              <Button className="w-full sm:w-auto" disabled={pending || !selectedMaterial || availableUnits.length === 0} type="submit" variant="outline">
                {pending ? "登记中..." : "登记损毁"}
              </Button>
            </div>
          </form>
        )}
      </AdminDialog>
      {message ? <p className="text-xs text-slate-500">{message}</p> : null}
    </div>
  );
}

export function MaterialDamageActions({
  id,
  status,
  canReview,
  canProcess,
}: {
  id: string;
  status: string;
  canReview: boolean;
  canProcess: boolean;
}) {
  const router = useRouter();
  const [message, setMessage] = useState("");
  const [approveComment, setApproveComment] = useState("");
  const [rejectComment, setRejectComment] = useState("");
  const [pending, setPending] = useState(false);

  async function patch(path: string, payload?: unknown, close?: () => void) {
    setPending(true);
    setMessage("");
    try {
      const item = await browserPatch<MaterialDamage>(path, payload);
      setMessage(`已更新为 ${damageStatusLabel(item.status)}`);
      close?.();
      startTransition(() => {
        router.refresh();
      });
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "操作失败");
    } finally {
      setPending(false);
    }
  }

  return (
    <div className="grid w-full gap-2 sm:flex sm:flex-wrap sm:items-center">
      {status === "pending" && canReview ? (
        <>
          <AdminDialog
            description="通过后还需执行损毁处理，库存才会自动扣减。"
            title="通过损毁登记"
            trigger={
              <Button className="h-10 w-full sm:h-8 sm:w-auto" disabled={pending} onClick={() => setApproveComment("")} size="sm">
                通过
              </Button>
            }
          >
            {(close) => (
              <div className="space-y-4">
                <label className="block min-w-0 space-y-2">
                  <span className="text-sm font-medium">审核备注</span>
                  <input className="h-10 w-full min-w-0 rounded-md border bg-white px-3 text-sm" onChange={(event) => setApproveComment(event.target.value)} placeholder="填写审核备注" value={approveComment} />
                </label>
                <div className="flex justify-end">
                  <Button className="w-full sm:w-auto" disabled={pending} onClick={() => patch(`/api/material-damages/${id}/approve`, { comment: approveComment }, close)} type="button">
                    确认通过
                  </Button>
                </div>
              </div>
            )}
          </AdminDialog>
          <AdminDialog
            description="拒绝后不会扣减库存。"
            title="拒绝损毁登记"
            trigger={
              <Button className="h-10 w-full sm:h-8 sm:w-auto" disabled={pending} onClick={() => setRejectComment("")} size="sm" variant="outline">
                拒绝
              </Button>
            }
          >
            {(close) => (
              <div className="space-y-4">
                <label className="block min-w-0 space-y-2">
                  <span className="text-sm font-medium">拒绝原因</span>
                  <input className="h-10 w-full min-w-0 rounded-md border bg-white px-3 text-sm" onChange={(event) => setRejectComment(event.target.value)} placeholder="填写拒绝原因" value={rejectComment} />
                </label>
                <div className="flex justify-end">
                  <Button className="w-full sm:w-auto" disabled={pending} onClick={() => patch(`/api/material-damages/${id}/reject`, { comment: rejectComment }, close)} type="button" variant="outline">
                    确认拒绝
                  </Button>
                </div>
              </div>
            )}
          </AdminDialog>
        </>
      ) : null}
      {status === "approved" && canProcess ? (
        <Button className="h-10 w-full sm:h-8 sm:w-auto" disabled={pending} onClick={() => patch(`/api/material-damages/${id}/process`)} size="sm">
          处理扣减
        </Button>
      ) : null}
      {message ? <span className="text-xs text-slate-500 sm:basis-full">{message}</span> : null}
    </div>
  );
}

export function MaterialsExportButton({ path, filename, label }: { path: string; filename: string; label: string }) {
  const [message, setMessage] = useState("");

  async function download() {
    setMessage("");
    try {
      const response = await fetch(path, { credentials: "include" });
      if (!response.ok) {
        setMessage(`导出失败：${response.status}`);
        return;
      }
      const blob = await response.blob();
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = filename;
      link.click();
      URL.revokeObjectURL(url);
    } catch {
      setMessage("导出失败，请稍后重试");
    }
  }

  return (
    <div className="space-y-1">
      <Button className="h-10 w-full sm:h-8 sm:w-auto" onClick={download} size="sm" type="button" variant="outline">
        <Download className="h-4 w-4" aria-hidden="true" />
        {label}
      </Button>
      {message ? <p className="text-xs text-destructive">{message}</p> : null}
    </div>
  );
}

export function MaterialImportForm() {
  const router = useRouter();
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>, close?: () => void) {
    event.preventDefault();
    const file = new FormData(event.currentTarget).get("file");
    if (!(file instanceof File) || file.size === 0) {
      setMessage("请选择 CSV、XLS 或 XLSX 文件");
      return;
    }
    const lowerName = file.name.toLowerCase();
    const contentType = lowerName.endsWith(".xlsx")
      ? "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
      : lowerName.endsWith(".xls")
        ? "application/vnd.ms-excel"
        : "text/csv; charset=utf-8";
    setPending(true);
    setMessage("");
    try {
      const response = await fetch(`/api/materials/import?filename=${encodeURIComponent(file.name)}`, {
        body: await file.arrayBuffer(),
        credentials: "include",
        headers: { "Content-Type": contentType },
        method: "POST",
      });
      const result = await response.json().catch(() => null) as (MaterialImportResult & { error?: string }) | null;
      if (!response.ok) {
        setMessage(result?.error ? `导入失败：${result.error}` : `导入失败：${response.status}`);
        return;
      }
      if (!result) {
        setMessage("导入失败：后端未返回导入结果");
        return;
      }
      const errorPreview = result.errors.length > 0 ? `；错误示例：${result.errors.slice(0, 3).join("；")}` : "";
      setMessage(`${result.message || `导入完成：新增 ${result.created}，更新 ${result.updated}，跳过 ${result.skipped}`}${errorPreview}`);
      close?.();
      startTransition(() => {
        router.refresh();
      });
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "导入失败");
    } finally {
      setPending(false);
    }
  }

  return (
    <div className="space-y-1">
      <AdminDialog
        description="支持 CSV、XLS 和 XLSX；相同二维码或相同资源名称加批号会更新现有记录。"
        title="导入资源"
        trigger={
          <Button className="h-10 w-full sm:h-8 sm:w-auto" size="sm" type="button" variant="outline">
            <Upload className="h-4 w-4" aria-hidden="true" />
            导入资源
          </Button>
        }
      >
        {(close) => (
          <form className="space-y-4" onSubmit={(event) => submit(event, close)}>
            <input accept=".csv,text/csv,.xls,application/vnd.ms-excel,.xlsx,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" className="w-full rounded-md border bg-white px-3 py-2 text-sm" name="file" required type="file" />
            <div className="flex justify-end">
              <Button className="w-full sm:w-auto" disabled={pending} type="submit">
                {pending ? "导入中..." : "开始导入"}
              </Button>
            </div>
          </form>
        )}
      </AdminDialog>
      {message ? <p className="text-xs text-slate-500">{message}</p> : null}
    </div>
  );
}

export function MaterialCategoryForm({ categories }: { categories: MaterialCategory[] }) {
  const router = useRouter();
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);
  const [sorting, setSorting] = useState(false);
  const [editingID, setEditingID] = useState("");
  const [orderedCategories, setOrderedCategories] = useState<MaterialCategory[]>(() => sortCategoryGroup(categories));
  const [dragging, setDragging] = useState<CategoryDragState | null>(null);

  useEffect(() => {
    setOrderedCategories(sortCategoryGroup(categories));
    setEditingID((current) => categories.some((item) => item.id === current) ? current : "");
  }, [categories]);

  const primaryCategories = useMemo(
    () => sortCategoryGroup(orderedCategories.filter((item) => categoryParentName(item) === "")),
    [orderedCategories],
  );
  const childCategories = useMemo(() => {
    const groups = new Map<string, MaterialCategory[]>();
    for (const primary of primaryCategories) {
      groups.set(primary.name, categorySiblings(orderedCategories, primary.name));
    }
    return groups;
  }, [orderedCategories, primaryCategories]);
  const disabled = pending || sorting;

  async function submit(event: FormEvent<HTMLFormElement>, close?: () => void) {
    event.preventDefault();
    setPending(true);
    setMessage("");
    const form = new FormData(event.currentTarget);
    const parentName = String(form.get("parentName") ?? "").trim();
    const nextDisplayOrder = categorySiblings(orderedCategories, parentName).reduce((max, item) => Math.max(max, item.displayOrder), 0) + 1;
    const payload: MaterialCategoryPayload = {
      name: String(form.get("name") ?? "").trim(),
      parentName,
      displayOrder: nextDisplayOrder,
      status: "active",
    };
    try {
      const item = await browserPost<MaterialCategory>("/api/materials/categories", payload);
      setMessage(`已保存分类：${item.name}`);
      close?.();
      startTransition(() => {
        router.refresh();
      });
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "分类保存失败");
    } finally {
      setPending(false);
    }
  }

  async function updateCategory(event: FormEvent<HTMLFormElement>, item: MaterialCategory) {
    event.preventDefault();
    setPending(true);
    setMessage("");
    const form = new FormData(event.currentTarget);
    const parentName = String(form.get("parentName") ?? "").trim();
    const payload: MaterialCategoryPayload = {
      name: String(form.get("name") ?? "").trim(),
      parentName,
      displayOrder: item.displayOrder,
      status: String(form.get("status") ?? item.status),
    };
    try {
      const updated = await browserPatch<MaterialCategory>(`/api/materials/categories/${item.id}`, payload);
      setMessage(`已修改目录：${updated.name}`);
      setEditingID("");
      startTransition(() => {
        router.refresh();
      });
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "目录修改失败");
    } finally {
      setPending(false);
    }
  }

  async function deleteCategory(item: MaterialCategory) {
    if (!confirm(`确定删除目录“${item.name}”吗？删除后该目录不会出现在新增资源的目录选项中。`)) {
      return;
    }
    const children = categorySiblings(orderedCategories, item.name);
    if (children.length > 0 && !confirm(`该一级目录下还有 ${children.length} 个二级目录，请再次确认删除。`)) {
      return;
    }
    setPending(true);
    setMessage("");
    try {
      const deleted = await browserDelete<MaterialCategory>(`/api/materials/categories/${item.id}`);
      setMessage(`已删除目录：${deleted.name}`);
      setEditingID("");
      startTransition(() => {
        router.refresh();
      });
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "目录删除失败");
    } finally {
      setPending(false);
    }
  }

  function dragStart(event: DragEvent<HTMLDivElement>, item: MaterialCategory) {
    if (disabled) {
      event.preventDefault();
      return;
    }
    const state = { id: item.id, parentName: categoryParentName(item) };
    setDragging(state);
    event.dataTransfer.effectAllowed = "move";
    event.dataTransfer.setData("text/plain", item.id);
  }

  function dragOver(event: DragEvent<HTMLDivElement>, item: MaterialCategory) {
    if (!dragging || dragging.id === item.id || dragging.parentName !== categoryParentName(item)) {
      return;
    }
    event.preventDefault();
    event.dataTransfer.dropEffect = "move";
  }

  function drop(event: DragEvent<HTMLDivElement>, target: MaterialCategory) {
    event.preventDefault();
    if (!dragging || dragging.id === target.id || dragging.parentName !== categoryParentName(target)) {
      setDragging(null);
      return;
    }
    const siblings = categorySiblings(orderedCategories, dragging.parentName);
    const fromIndex = siblings.findIndex((item) => item.id === dragging.id);
    const toIndex = siblings.findIndex((item) => item.id === target.id);
    if (fromIndex < 0 || toIndex < 0) {
      setDragging(null);
      return;
    }
    const reordered = moveCategory(siblings, fromIndex, toIndex).map((item, index) => ({ ...item, displayOrder: index + 1 }));
    const reorderedByID = new Map(reordered.map((item) => [item.id, item]));
    setOrderedCategories((items) => sortCategoryGroup(items.map((item) => reorderedByID.get(item.id) ?? item)));
    setDragging(null);
    void saveCategoryOrder(reordered);
  }

  async function saveCategoryOrder(items: MaterialCategory[]) {
    setSorting(true);
    setMessage("");
    try {
      await Promise.all(items.map((item) => browserPatch<MaterialCategory>(`/api/materials/categories/${item.id}`, categoryPayload(item))));
      setMessage("目录排序已保存");
      startTransition(() => {
        router.refresh();
      });
    } catch (error) {
      setOrderedCategories(sortCategoryGroup(categories));
      setMessage(error instanceof Error ? error.message : "目录排序保存失败");
    } finally {
      setSorting(false);
    }
  }

  return (
    <div className="space-y-1">
      <AdminDialog
        description="目录用于资源检索和统计，由试剂管理员维护一级目录与二级目录。"
        title="目录维护"
        trigger={
          <Button className="h-10 w-full sm:h-8 sm:w-auto" size="sm" type="button" variant="outline">
            目录维护
          </Button>
        }
      >
        {(close) => (
          <div className="space-y-4">
            <form className="space-y-3" onSubmit={(event) => submit(event, close)}>
              <Field label="目录名称" name="name" placeholder="填写目录名称" required />
              <select className="h-10 w-full rounded-md border bg-white px-3 text-sm" name="parentName">
                <option value="">一级目录</option>
                {primaryCategories.filter((item) => item.status === "active").map((item) => (
                  <option key={item.id} value={item.name}>
                    {item.name}
                  </option>
                ))}
              </select>
              <div className="flex justify-end">
                <Button className="w-full sm:w-auto" disabled={disabled} type="submit">
                  {pending ? "保存中..." : "保存分类"}
                </Button>
              </div>
            </form>
            <div className="max-h-80 space-y-3 overflow-y-auto pr-1">
              {primaryCategories.length === 0 ? (
                <p className="rounded-md border border-dashed px-3 py-4 text-center text-sm text-slate-500">暂无目录</p>
              ) : (
                primaryCategories.map((primary) => {
                  const children = childCategories.get(primary.name) ?? [];
                  return (
                    <div className="space-y-2" key={primary.id}>
                      <CategorySortRow disabled={disabled} draggingID={dragging?.id} editingID={editingID} item={primary} level="primary" onCancelEdit={() => setEditingID("")} onDelete={deleteCategory} onDragEnd={() => setDragging(null)} onDragOver={dragOver} onDragStart={dragStart} onDrop={drop} onEdit={() => setEditingID(primary.id)} onSubmitEdit={updateCategory} primaryCategories={primaryCategories} />
                      <div className="space-y-2">
                        {children.map((child) => (
                          <CategorySortRow disabled={disabled} draggingID={dragging?.id} editingID={editingID} item={child} key={child.id} level="secondary" onCancelEdit={() => setEditingID("")} onDelete={deleteCategory} onDragEnd={() => setDragging(null)} onDragOver={dragOver} onDragStart={dragStart} onDrop={drop} onEdit={() => setEditingID(child.id)} onSubmitEdit={updateCategory} primaryCategories={primaryCategories} />
                        ))}
                      </div>
                    </div>
                  );
                })
              )}
            </div>
          </div>
        )}
      </AdminDialog>
      {message ? <p className="text-xs text-slate-500">{message}</p> : null}
    </div>
  );
}

type CategoryDragState = {
  id: string;
  parentName: string;
};

function CategorySortRow({
  disabled,
  draggingID,
  editingID,
  item,
  level,
  onCancelEdit,
  onDelete,
  onDragEnd,
  onDragOver,
  onDragStart,
  onDrop,
  onEdit,
  onSubmitEdit,
  primaryCategories,
}: {
  disabled: boolean;
  draggingID?: string;
  editingID: string;
  item: MaterialCategory;
  level: "primary" | "secondary";
  onCancelEdit: () => void;
  onDelete: (item: MaterialCategory) => void;
  onDragEnd: () => void;
  onDragOver: (event: DragEvent<HTMLDivElement>, item: MaterialCategory) => void;
  onDragStart: (event: DragEvent<HTMLDivElement>, item: MaterialCategory) => void;
  onDrop: (event: DragEvent<HTMLDivElement>, item: MaterialCategory) => void;
  onEdit: () => void;
  onSubmitEdit: (event: FormEvent<HTMLFormElement>, item: MaterialCategory) => void;
  primaryCategories: MaterialCategory[];
}) {
  const isDragging = draggingID === item.id;
  const editing = editingID === item.id;
  const rowClass = [
    "rounded-md border bg-white px-3 py-2 text-sm transition",
    level === "secondary" ? "ml-6 border-slate-200" : "border-slate-300",
    disabled || editing ? "opacity-70" : "hover:border-slate-400",
    isDragging ? "border-slate-500 bg-slate-50 opacity-60" : "",
  ].filter(Boolean).join(" ");

  if (editing) {
    return (
      <form className={`${rowClass} cursor-auto space-y-3`} onSubmit={(event) => onSubmitEdit(event, item)}>
        <div className="grid gap-2 sm:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_120px]">
          <Field defaultValue={item.name} label="目录名称" name="name" placeholder="填写目录名称" required />
          <label className="space-y-1 text-sm">
            <span className="text-sm font-medium">上级目录</span>
            <select className="h-10 w-full rounded-md border bg-white px-3 text-sm" defaultValue={categoryParentName(item)} name="parentName">
              <option value="">一级目录</option>
              {primaryCategories.filter((primary) => primary.id !== item.id && primary.status === "active").map((primary) => (
                <option key={primary.id} value={primary.name}>
                  {primary.name}
                </option>
              ))}
            </select>
          </label>
          <label className="space-y-1 text-sm">
            <span className="text-sm font-medium">状态</span>
            <select className="h-10 w-full rounded-md border bg-white px-3 text-sm" defaultValue={item.status} name="status">
              <option value="active">启用</option>
              <option value="disabled">停用</option>
            </select>
          </label>
        </div>
        <div className="flex flex-col gap-2 sm:flex-row sm:justify-end">
          <Button className="w-full sm:w-auto" disabled={disabled} type="button" variant="outline" onClick={onCancelEdit}>
            取消
          </Button>
          <Button className="w-full sm:w-auto" disabled={disabled} type="submit">
            保存修改
          </Button>
        </div>
      </form>
    );
  }

  return (
    <div
      aria-disabled={disabled}
      className={rowClass}
      draggable={!disabled}
      onDragEnd={onDragEnd}
      onDragOver={(event) => onDragOver(event, item)}
      onDragStart={(event) => onDragStart(event, item)}
      onDrop={(event) => onDrop(event, item)}
    >
      <div className="flex items-center gap-2">
        <GripVertical className="h-4 w-4 flex-none text-slate-400" aria-hidden="true" />
        <div className="min-w-0 flex-1">
          <p className="truncate font-bold">{item.name}</p>
          <p className="text-xs text-slate-500">{level === "primary" ? "一级目录" : item.parentName} / {item.status === "active" ? "启用" : "停用"}</p>
        </div>
        <span className="flex-none rounded border px-2 py-1 text-xs text-slate-500">{item.displayOrder}</span>
      </div>
      <div className="mt-2 flex flex-col gap-2 sm:flex-row sm:justify-end">
        <Button className="w-full sm:w-auto" disabled={disabled} size="sm" type="button" variant="outline" onClick={onEdit}>
          <Pencil className="h-4 w-4" aria-hidden="true" />
          修改
        </Button>
        <Button className="w-full sm:w-auto" disabled={disabled} size="sm" type="button" variant="destructive" onClick={() => onDelete(item)}>
          <Trash2 className="h-4 w-4" aria-hidden="true" />
          删除
        </Button>
      </div>
    </div>
  );
}

export function MaterialAlertActionForm({ material, alertType }: { material: Material; alertType: string }) {
  const router = useRouter();
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);

  async function submit(action: "handled" | "ignored") {
    setPending(true);
    setMessage("");
    const payload: MaterialAlertActionPayload = {
      action,
      alertType,
      comment: action === "handled" ? "已处理预警" : "已忽略预警",
    };
    try {
      const item = await browserPost<MaterialAlertAction>(`/api/materials/${material.id}/alert-actions`, payload);
      setMessage(item.action === "handled" ? "已标记处理" : "已忽略");
      startTransition(() => {
        router.refresh();
      });
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "预警处理失败");
    } finally {
      setPending(false);
    }
  }

  return (
    <div className="mt-2 flex flex-wrap items-center gap-2">
      <Button disabled={pending} onClick={() => submit("handled")} size="sm" type="button" variant="outline">
        标记处理
      </Button>
      <Button disabled={pending} onClick={() => submit("ignored")} size="sm" type="button" variant="outline">
        忽略
      </Button>
      {message ? <span className="text-xs text-slate-500">{message}</span> : null}
    </div>
  );
}

function MaterialFields({ material, categories, productType, purchasableMaterials = [] }: { material?: Material; categories: MaterialCategory[]; productType?: MaterialProductType; purchasableMaterials?: PurchasableMaterial[] }) {
  const [selectedCategory, setSelectedCategory] = useState(material?.category ?? "");
  const [selectedSubcategory, setSelectedSubcategory] = useState(material?.subcategory ?? "");
  const [selectedPurchasableId, setSelectedPurchasableId] = useState("");
  const [purchasableSearch, setPurchasableSearch] = useState("");
  const [standardCertificateUrl, setStandardCertificateUrl] = useState(material?.standardCertificateUrl ?? "");
  const [certificateUploadMessage, setCertificateUploadMessage] = useState("");
  const [certificateUploading, setCertificateUploading] = useState(false);
  const selectedPurchasable = purchasableMaterials.find((item) => item.id === selectedPurchasableId);
  const purchasableQuery = purchasableSearch.trim().toLowerCase();
  const visiblePurchasableMaterials = purchasableQuery === ""
    ? purchasableMaterials.slice(0, 8)
    : purchasableMaterials.filter((item) => purchasableMaterialSearchText(item).includes(purchasableQuery)).slice(0, 8);
  const purchasableKey = material?.id ?? selectedPurchasable?.id ?? "manual";
  const primaryDirectories = materialDirectoryOptions(categories, "", material?.category);
  const secondaryDirectories = selectedCategory ? materialDirectoryOptions(categories, selectedCategory, material?.subcategory) : material?.subcategory ? [material.subcategory] : [];
  const resolvedProductType = material?.productType ?? productType ?? "consumable";
  const isStandard = resolvedProductType === "standard";

  async function uploadStandardCertificate(file: File | undefined) {
    if (!file || file.size === 0) {
      return;
    }
    setCertificateUploading(true);
    setCertificateUploadMessage("");
    try {
      const body = new FormData();
      body.append("file", file);
      const response = await fetch("/api/uploads/material-certificates", {
        body,
        credentials: "include",
        method: "POST",
      });
      const result = await response.json().catch(() => null) as { url?: string; error?: string } | null;
      if (!response.ok || !result?.url) {
        setCertificateUploadMessage(result?.error ? `上传失败：${result.error}` : `上传失败：${response.status}`);
        return;
      }
      setStandardCertificateUrl(result.url);
      setCertificateUploadMessage("证书已上传，保存资源后生效。");
    } catch (error) {
      setCertificateUploadMessage(error instanceof Error ? error.message : "上传失败");
    } finally {
      setCertificateUploading(false);
    }
  }

  return (
    <>
      {!material && purchasableMaterials.length > 0 ? (
        <div className="space-y-3 rounded-lg border bg-slate-50 p-3">
          <label className="block min-w-0 space-y-2">
            <span className="text-sm font-medium">从可采购物资目录带出</span>
            <input
              className="h-10 w-full rounded-md border bg-white px-3 text-sm"
              onChange={(event) => setPurchasableSearch(event.target.value)}
              placeholder="搜索ID号、序号、项目、品牌、规格、采购项目"
              value={purchasableSearch}
            />
          </label>
          <div className="grid max-h-56 gap-2 overflow-y-auto">
            <button className="rounded-md border bg-white px-3 py-2 text-left text-sm hover:bg-slate-100" onClick={() => setSelectedPurchasableId("")} type="button">
              手动填写
            </button>
            {visiblePurchasableMaterials.map((item) => (
              <button
                className={`rounded-md border px-3 py-2 text-left text-sm hover:bg-slate-100 ${item.id === selectedPurchasableId ? "border-primary bg-primary/5" : "bg-white"}`}
                key={item.id}
                onClick={() => {
                  setSelectedPurchasableId(item.id);
                  setPurchasableSearch(purchasableMaterialOptionLabel(item));
                }}
                type="button"
              >
                <span className="block break-words font-medium text-slate-900">{purchasableMaterialOptionLabel(item)}</span>
                <span className="mt-1 block break-words text-xs text-slate-500">{item.procurementProject || "未登记采购项目名称及编号"}</span>
              </button>
            ))}
          </div>
          {selectedPurchasable ? <FieldHint value={`已选择：${purchasableMaterialOptionLabel(selectedPurchasable)}；采购项目名称及编号：${selectedPurchasable.procurementProject || "未登记"}`} /> : null}
        </div>
      ) : null}
      {!material ? (
        <Field label="申购流水号" name="purchaseSerialNo" placeholder="填写申购单ID、ID号或序号后自动匹配采购人和采购目录" />
      ) : null}
      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
        <Field defaultValue={material?.name ?? selectedPurchasable?.projectName} key={`name-${purchasableKey}`} label="资源名称" name="name" placeholder="填写资源名称" required />
        <label className="block min-w-0 space-y-2">
          <span className="text-sm font-medium">资源类型</span>
          <select className="h-10 w-full rounded-md border bg-white px-3 text-sm" defaultValue={resolvedProductType} disabled={!material && Boolean(productType)} name="productType">
            <option value="consumable">耗材</option>
            <option value="reagent">试剂</option>
            <option value="standard">标准品/标准物质</option>
          </select>
          {!material && productType ? <input name="productType" type="hidden" value={productType} /> : null}
          {material ? <FieldHint value={`当前：${productTypeLabel(material.productType)}`} /> : null}
        </label>
        <label className="block min-w-0 space-y-2">
          <span className="text-sm font-medium">一级目录</span>
          <select
            className="h-10 w-full rounded-md border bg-white px-3 text-sm"
            name="category"
            onChange={(event) => {
              setSelectedCategory(event.target.value);
              setSelectedSubcategory("");
            }}
            required
            value={selectedCategory}
          >
            <option value="">选择一级目录</option>
            {primaryDirectories.map((category) => (
              <option key={category} value={category}>
                {category}
              </option>
            ))}
          </select>
          {primaryDirectories.length === 0 ? <FieldHint value="请先通过目录维护新增一级目录" /> : null}
        </label>
        <label className="block min-w-0 space-y-2">
          <span className="text-sm font-medium">二级目录</span>
          <select className="h-10 w-full rounded-md border bg-white px-3 text-sm" disabled={!selectedCategory || secondaryDirectories.length === 0} name="subcategory" onChange={(event) => setSelectedSubcategory(event.target.value)} value={selectedSubcategory}>
            <option value="">选择二级目录</option>
            {secondaryDirectories.map((subcategory) => (
              <option key={subcategory} value={subcategory}>
                {subcategory}
              </option>
            ))}
          </select>
          {selectedCategory && secondaryDirectories.length === 0 ? <FieldHint value="该一级目录暂未维护二级目录" /> : null}
        </label>
      </div>
      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
        <Field defaultValue={material?.spec ?? selectedPurchasable?.spec} key={`spec-${purchasableKey}`} label="规格" name="spec" placeholder="填写规格" required />
        <Field defaultValue={material?.unit ?? selectedPurchasable?.unit} key={`unit-${purchasableKey}`} label="单位" name="unit" placeholder="填写库存单位" required />
        <Field defaultValue={material?.supplier ?? selectedPurchasable?.brand} key={`supplier-${purchasableKey}`} label="供应商" name="supplier" placeholder="填写供应商" />
      </div>
      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
        <Field defaultValue={material?.manufacturer ?? selectedPurchasable?.brand} key={`manufacturer-${purchasableKey}`} label="生产商" name="manufacturer" placeholder="填写生产商" />
        <Field defaultValue={material?.catalogNo ?? selectedPurchasable?.idNo} key={`catalogNo-${purchasableKey}`} label="货号" name="catalogNo" placeholder="填写货号" />
        <Field defaultValue={material?.casNo} label="CAS号" name="casNo" placeholder="填写 CAS 号" />
      </div>
      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
        <Field defaultValue={material?.grade} label="级别" name="grade" placeholder="例如 AR / HPLC / CRM" />
        <Field defaultValue={material?.concentration} label="浓度" name="concentration" placeholder="填写浓度或含量" />
        <Field defaultValue={material?.storageCondition} label="保存条件" name="storageCondition" placeholder="例如 2-8°C 避光" />
      </div>
      {!isStandard ? (
        <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
          <Field defaultValue={material?.parentMaterialId} label="母液/来源资源ID" name="parentMaterialId" placeholder="填写来源资源 ID" />
          <Field defaultValue={material?.dilutionFactor} label="稀释倍数" name="dilutionFactor" placeholder="例如 1:10" />
          <Field defaultValue={material?.preparationMethod} label="配制方法" name="preparationMethod" placeholder="填写工作液或混标配制方法" />
        </div>
      ) : null}
      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
        {material ? (
          <div className="rounded-md border bg-slate-50 px-3 py-2 text-sm">
            <p className="text-xs text-slate-500">当前库存</p>
            <p className="font-bold">
              {material.stock}
              {material.unit}
            </p>
            <input name="stock" type="hidden" value={material.stock} />
          </div>
        ) : (
          <Field label="初始库存" min={0} name="stock" placeholder="填写初始库存数量" required type="number" />
        )}
        <Field defaultValue={material?.warningLine} label="库存告警线" min={0} name="warningLine" placeholder="低于或等于该数量触发告警" required type="number" />
        <Field defaultValue={material?.unitPrice ?? selectedPurchasable?.purchasePrice} key={`unitPrice-${purchasableKey}`} label="单价" min={0} name="unitPrice" placeholder="填写单价" required step="0.01" type="number" />
      </div>
      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
        <Field defaultValue={material?.batchNo} label="批次号" name="batchNo" placeholder="填写批次号" />
        <Field defaultValue={material?.expiresAt} label="有效期" name="expiresAt" placeholder="选择有效期" type="date" />
        <Field defaultValue={material?.nearExpiryDays ?? 30} label="临期预警天数" min={0} name="nearExpiryDays" placeholder="到期前多少天进入临期" type="number" />
      </div>
      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
        <Field defaultValue={material?.storageRoom} label="库房/冰箱" name="storageRoom" placeholder="填写库房或冰箱" />
        <Field defaultValue={material?.storageCabinet} label="柜/架" name="storageCabinet" placeholder="填写柜或架" />
        <Field defaultValue={material?.storageLayer} label="层/盒" name="storageLayer" placeholder="填写层或盒" />
        <Field defaultValue={material?.storageSlot} label="孔位" name="storageSlot" placeholder="填写孔位" />
      </div>
      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
        <Field defaultValue={material?.openedAt} label="开封日期" name="openedAt" placeholder="选择开封日期" type="date" />
        <Field defaultValue={material?.openExpireDays} label="开封有效天数" min={0} name="openExpireDays" placeholder="0 表示不限制" type="number" />
        <Field defaultValue={material?.freezeThawCount} label="冻融次数" min={0} name="freezeThawCount" placeholder="当前冻融次数" type="number" />
        <Field defaultValue={material?.freezeThawLimit} label="冻融上限" min={0} name="freezeThawLimit" placeholder="0 表示不限制" type="number" />
      </div>
      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
        <Field defaultValue={material?.tenderContract || material?.contractNo || selectedPurchasable?.procurementProject} key={`procurementProject-${purchasableKey}`} label="采购项目名称及编号" name="procurementProject" placeholder="填写采购项目名称及编号" />
        <Field defaultValue={material?.qrCode} label="二维码编码" name="qrCode" placeholder="留空时由系统按资源编号生成" />
        <label className="flex min-h-10 items-center gap-2 rounded-md border bg-white px-3 text-sm">
          <input defaultChecked={material?.approvalRequired ?? false} name="approvalRequired" type="checkbox" value="true" />
          特殊资源申领需要审批
        </label>
      </div>
      <div className="grid gap-3 md:grid-cols-2">
        <Field defaultValue={material?.remark ?? selectedPurchasable?.remark} key={`remark-${purchasableKey}`} label="备注" name="remark" placeholder="填写备注信息" />
        <Field defaultValue={material?.certificateUrl} label="资源证书地址" name="certificateUrl" placeholder="填写证书查看或下载地址" />
      </div>
      <div className="grid gap-3 md:grid-cols-2">
        {isStandard ? (
          <label className="block min-w-0 space-y-2">
            <span className="text-sm font-medium">标准品/标准物质证书</span>
            <input
              className="h-10 w-full min-w-0 rounded-md border bg-white px-3 text-sm"
              onChange={(event) => uploadStandardCertificate(event.currentTarget.files?.[0])}
              type="file"
              accept=".pdf,application/pdf"
            />
            <input className="h-10 w-full min-w-0 rounded-md border bg-white px-3 text-sm" name="standardCertificateUrl" onChange={(event) => setStandardCertificateUrl(event.target.value)} placeholder="上传 PDF 后自动填入，也可手动填写证书地址" value={standardCertificateUrl} />
            {certificateUploadMessage ? <FieldHint value={certificateUploadMessage} /> : null}
            {certificateUploading ? <FieldHint value="证书上传中..." /> : null}
          </label>
        ) : (
          <Field defaultValue={material?.standardCertificateUrl} label="标准证书地址" name="standardCertificateUrl" placeholder="标准证书地址" />
        )}
        <Field defaultValue={material?.attachmentUrl} label="附件地址" name="attachmentUrl" placeholder="填写说明文件或附件地址" />
      </div>
      {material ? (
        <label className="block min-w-0 space-y-2">
          <span className="text-sm font-medium">状态</span>
          <select className="h-10 w-full rounded-md border bg-white px-3 text-sm" defaultValue={["low", "near_expiry", "open_expired", "freeze_thaw_exceeded", "damaged"].includes(material.status) ? "normal" : material.status} name="status">
            <option value="normal">正常</option>
            <option value="expired">已过期</option>
            <option value="damaged">损毁</option>
            <option value="disabled">停用</option>
          </select>
          <FieldHint value={`当前：${statusLabel(material.status)}`} />
        </label>
      ) : null}
    </>
  );
}

function Field({
  defaultValue,
  label,
  min,
  name,
  placeholder,
  required = false,
  step,
  type = "text",
}: {
  defaultValue?: string | number;
  label?: string;
  min?: number;
  name: string;
  placeholder: string;
  required?: boolean;
  step?: string;
  type?: string;
}) {
  const input = (
    <input
      className="h-10 w-full min-w-0 rounded-md border bg-white px-3 text-sm"
      defaultValue={defaultValue ?? ""}
      min={min}
      name={name}
      placeholder={placeholder}
      required={required}
      step={step}
      type={type}
    />
  );
  if (!label) {
    return input;
  }
  return (
    <label className="block min-w-0 space-y-2">
      <span className="text-sm font-medium">{label}</span>
      {input}
      {defaultValue !== undefined ? <FieldHint value={`当前：${formatCurrent(defaultValue)}`} /> : null}
    </label>
  );
}

function FieldHint({ value }: { value: string }) {
  return <span className="block break-words text-xs text-slate-500">{value}</span>;
}

function formatCurrent(value: string | number) {
  const text = String(value).trim();
  return text === "" ? "未设置" : text;
}

function materialBatchLocation(material?: Material) {
  if (!material) {
    return "";
  }
  return [material.storageRoom, material.storageCabinet, material.storageLayer, material.storageSlot].filter(Boolean).join(" / ");
}

function materialPayload(form: FormData, fallbackStatus: string): MaterialPayload {
  const productType = String(form.get("productType") ?? "consumable");
  const procurementProject = String(form.get("procurementProject") ?? form.get("tenderContract") ?? "");
  return {
    name: String(form.get("name") ?? ""),
    productType,
    category: String(form.get("category") ?? ""),
    subcategory: String(form.get("subcategory") ?? ""),
    spec: String(form.get("spec") ?? ""),
    unit: String(form.get("unit") ?? ""),
    unitPrice: Number(form.get("unitPrice") ?? 0),
    stock: Number(form.get("stock") ?? 0),
    warningLine: Number(form.get("warningLine") ?? 0),
    supplier: String(form.get("supplier") ?? ""),
    manufacturer: String(form.get("manufacturer") ?? ""),
    batchNo: String(form.get("batchNo") ?? ""),
    catalogNo: String(form.get("catalogNo") ?? ""),
    casNo: String(form.get("casNo") ?? ""),
    grade: String(form.get("grade") ?? ""),
    concentration: String(form.get("concentration") ?? ""),
    parentMaterialId: productType === "standard" ? "" : String(form.get("parentMaterialId") ?? ""),
    dilutionFactor: productType === "standard" ? "" : String(form.get("dilutionFactor") ?? ""),
    preparationMethod: productType === "standard" ? "" : String(form.get("preparationMethod") ?? ""),
    storageCondition: String(form.get("storageCondition") ?? ""),
    storageRoom: String(form.get("storageRoom") ?? ""),
    storageCabinet: String(form.get("storageCabinet") ?? ""),
    storageLayer: String(form.get("storageLayer") ?? ""),
    storageSlot: String(form.get("storageSlot") ?? ""),
    tenderContract: procurementProject,
    contractNo: procurementProject,
    remark: String(form.get("remark") ?? ""),
    certificateUrl: String(form.get("certificateUrl") ?? ""),
    standardCertificateUrl: String(form.get("standardCertificateUrl") ?? ""),
    attachmentUrl: String(form.get("attachmentUrl") ?? ""),
    qrCode: String(form.get("qrCode") ?? ""),
    purchaseSerialNo: String(form.get("purchaseSerialNo") ?? ""),
    expiresAt: String(form.get("expiresAt") ?? ""),
    openedAt: String(form.get("openedAt") ?? ""),
    openExpireDays: Number(form.get("openExpireDays") ?? 0),
    freezeThawCount: Number(form.get("freezeThawCount") ?? 0),
    freezeThawLimit: Number(form.get("freezeThawLimit") ?? 0),
    approvalRequired: form.get("approvalRequired") === "true",
    nearExpiryDays: Number(form.get("nearExpiryDays") ?? 30),
    status: String(form.get("status") ?? fallbackStatus),
  };
}

function statusLabel(status: string) {
  const labels: Record<string, string> = {
    normal: "正常",
    near_expiry: "临期",
    low: "低库存",
    expired: "已过期",
    open_expired: "开封超期",
    freeze_thaw_exceeded: "冻融超限",
    damaged: "损毁",
    disabled: "停用",
  };
  return labels[status] ?? status;
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

function productTypeLabel(productType: string) {
  const labels: Record<string, string> = {
    consumable: "耗材",
    reagent: "试剂",
    standard: "标准品/标准物质",
  };
  return labels[productType] ?? productType;
}

function purchasableMaterialOptionLabel(item: PurchasableMaterial) {
  return [item.idNo, item.sequenceNo, item.projectName, item.brand, item.spec, item.unit].filter(Boolean).join(" / ");
}

function purchasableMaterialSearchText(item: PurchasableMaterial) {
  return [
    item.idNo,
    item.sequenceNo,
    item.procurementProject,
    item.projectName,
    item.brand,
    item.spec,
    item.unit,
    item.remark,
    item.technicalRequirement,
    item.minSpec,
  ].join(" ").toLowerCase();
}

function materialDirectoryOptions(categories: MaterialCategory[], parentName: string, current?: string) {
  const normalizedParent = parentName.trim();
  const options = sortCategoryGroup(categories)
    .filter((item) => item.status === "active" && item.parentName.trim() === normalizedParent)
    .map((item) => item.name.trim())
    .filter(Boolean);
  const currentValue = current?.trim();
  if (currentValue && !options.includes(currentValue)) {
    options.push(currentValue);
  }
  return Array.from(new Set(options));
}

function categoryParentName(item: MaterialCategory) {
  return item.parentName.trim();
}

function sortCategoryGroup(items: MaterialCategory[]) {
  return [...items].sort((left, right) => {
    const orderDiff = left.displayOrder - right.displayOrder;
    if (orderDiff !== 0) {
      return orderDiff;
    }
    const parentDiff = categoryParentName(left).localeCompare(categoryParentName(right), "zh-Hans-CN");
    if (parentDiff !== 0) {
      return parentDiff;
    }
    return left.name.trim().localeCompare(right.name.trim(), "zh-Hans-CN");
  });
}

function categorySiblings(categories: MaterialCategory[], parentName: string) {
  const normalizedParent = parentName.trim();
  return sortCategoryGroup(categories.filter((item) => categoryParentName(item) === normalizedParent));
}

function moveCategory(items: MaterialCategory[], fromIndex: number, toIndex: number) {
  const next = [...items];
  const [moved] = next.splice(fromIndex, 1);
  next.splice(toIndex, 0, moved);
  return next;
}

function categoryPayload(item: MaterialCategory): MaterialCategoryPayload {
  return {
    name: item.name,
    parentName: item.parentName,
    displayOrder: item.displayOrder,
    status: item.status,
  };
}
