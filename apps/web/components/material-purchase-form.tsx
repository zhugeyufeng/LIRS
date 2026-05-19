"use client";

import { FormEvent, startTransition, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { Download, Pencil, Save, Search, ShoppingCart, Trash2, Upload } from "lucide-react";
import { browserDelete, browserPatch, browserPost, Material, MaterialImportResult, MaterialPurchase, MaterialPurchasePayload, ProcurementProject, ProcurementProjectPayload, PurchasableMaterial, PurchasableMaterialPayload } from "@/lib/api";
import { AdminDialog } from "@/components/admin-dialog";
import { Button } from "@/components/ui/button";

export function MaterialPurchaseForm({
  inline = false,
  material,
  materials,
  purchasableMaterials = [],
}: {
  inline?: boolean;
  material?: Material;
  materials: Material[];
  purchasableMaterials?: PurchasableMaterial[];
}) {
  const router = useRouter();
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);
  const [selectedPurchasableId, setSelectedPurchasableId] = useState("");
  const [purchasableSearch, setPurchasableSearch] = useState("");
  const selectedPurchasable = purchasableMaterials.find((item) => item.id === selectedPurchasableId);
  const hasAvailablePurchasableMaterials = purchasableMaterials.some((item) => !purchasableMaterialExpired(item));
  const visiblePurchasableOptions = useMemo(() => {
    const query = purchasableSearch.trim().toLowerCase();
    const activeItems = purchasableMaterials.filter((item) => !purchasableMaterialExpired(item));
    const source = query === "" ? activeItems : activeItems.filter((item) => purchasableMaterialSearchText(item).includes(query));
    return source.slice(0, 20);
  }, [purchasableMaterials, purchasableSearch]);

  async function submit(event: FormEvent<HTMLFormElement>, close?: () => void) {
    event.preventDefault();
    setMessage("");
    const formElement = event.currentTarget;
    const form = new FormData(formElement);
    const payload: MaterialPurchasePayload = {
      materialId: String(form.get("materialId") ?? ""),
      purchasableMaterialId: String(form.get("purchasableMaterialId") ?? ""),
      quantity: Number(form.get("quantity") ?? 0),
      estimatedUnitPrice: Number(form.get("estimatedUnitPrice") ?? 0),
      supplier: String(form.get("supplier") ?? ""),
      reason: String(form.get("reason") ?? ""),
    };
    if (!material && payload.purchasableMaterialId === "") {
      setMessage("请选择申购物品");
      return;
    }
    setPending(true);
    try {
      const purchase = await browserPost<MaterialPurchase>("/api/material-purchases", payload);
      setMessage(`申购已提交：${purchase.materialName} x${purchase.quantity}`);
      formElement.reset();
      setSelectedPurchasableId("");
      setPurchasableSearch("");
      close?.();
      startTransition(() => {
        router.refresh();
      });
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "提交失败");
    } finally {
      setPending(false);
    }
  }

  const form = (close?: () => void) => (
    <form className="space-y-4" onSubmit={(event) => submit(event, close)}>
      {material ? (
        <div className="rounded-md border bg-slate-50 px-3 py-2 text-sm">
          <input name="materialId" type="hidden" value={material.id} />
          <p className="text-xs text-slate-500">当前申购资源</p>
          <p className="mt-1 font-bold text-slate-900">
            {material.name} / 当前 {material.stock}
            {material.unit}
          </p>
        </div>
      ) : (
        <div className="space-y-2">
          <input name="purchasableMaterialId" type="hidden" value={selectedPurchasableId} />
          <div className="relative">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" aria-hidden="true" />
            <input
              className="h-10 w-full rounded-md border bg-white pl-10 pr-3 text-sm"
              onChange={(event) => setPurchasableSearch(event.currentTarget.value)}
              placeholder="搜索ID号、项目名称、品牌、规格"
              type="search"
              value={purchasableSearch}
            />
          </div>
          <div className="max-h-72 overflow-y-auto rounded-md border bg-white">
            {visiblePurchasableOptions.map((item) => {
              const selected = item.id === selectedPurchasableId;
              return (
                <button
                  className={`grid w-full gap-1 border-b px-3 py-2 text-left text-sm last:border-b-0 hover:bg-slate-50 ${selected ? "bg-primary/10" : ""}`}
                  key={item.id}
                  onClick={() => {
                    setSelectedPurchasableId(item.id);
                    setPurchasableSearch(purchasableMaterialOptionLabel(item));
                  }}
                  type="button"
                >
                  <span className="font-medium text-slate-900">{item.projectName}</span>
                  <span className="text-xs text-slate-500">{item.idNo} / {item.brand} / {item.spec} / {item.unit} / {formatMoney(item.purchasePrice)}</span>
                </button>
              );
            })}
            {visiblePurchasableOptions.length === 0 ? <p className="p-3 text-sm text-slate-500">未找到匹配的申购物品。</p> : null}
          </div>
        </div>
      )}
      {selectedPurchasable ? (
        <div className="grid gap-2 rounded-md border bg-slate-50 p-3 text-sm sm:grid-cols-2">
          <Info className="sm:col-span-2" label="采购项目名称及编号" value={selectedPurchasable.procurementProject || "未登记"} />
          <Info className="sm:col-span-2" label="项目名称" value={selectedPurchasable.projectName} />
          <Info label="序号" value={selectedPurchasable.sequenceNo} />
          <Info label="单位" value={selectedPurchasable.unit} />
          <Info label="采购价" value={formatMoney(selectedPurchasable.purchasePrice)} />
          <Info label="项目有效期" value={selectedPurchasable.procurementExpiresAt || "长期有效"} />
          <Info label="最小规格" value={selectedPurchasable.minSpec || "未登记"} />
        </div>
      ) : null}
      <div className="grid gap-3 sm:grid-cols-2">
        <input className="h-10 min-w-0 rounded-md border bg-white px-3 text-sm" min={1} name="quantity" placeholder="申购数量" required type="number" />
        <input className="h-10 min-w-0 rounded-md border bg-white px-3 text-sm" defaultValue={material ? undefined : selectedPurchasable?.purchasePrice} key={selectedPurchasable?.id ?? "estimatedUnitPrice"} min={0} name="estimatedUnitPrice" placeholder="预计单价" required step="0.01" type="number" />
      </div>
      <input className="h-10 w-full rounded-md border bg-white px-3 text-sm" name="supplier" placeholder="供应商，可留空沿用资源资料" />
      <input className="h-10 w-full rounded-md border bg-white px-3 text-sm" name="reason" placeholder="申购原因" required />
      <div className="flex justify-end">
        <Button className="w-full sm:w-auto" disabled={pending} type="submit">
          {pending ? "提交中..." : "提交申购"}
        </Button>
      </div>
    </form>
  );

  if (inline) {
    return (
      <div className="space-y-2">
        {form()}
        {message ? <p className="text-xs text-slate-500">{message}</p> : null}
      </div>
    );
  }

  return (
    <div className="space-y-2">
      <AdminDialog
        description="申购提交后进入审批流程，审批、下单和入库在申购管理列表中处理。"
        title="新建申购"
        trigger={
          <Button className="w-full" disabled={pending || (!material && !hasAvailablePurchasableMaterials)}>
            <ShoppingCart className="h-4 w-4" aria-hidden="true" />
            新建申购
          </Button>
        }
      >
        {(close) => form(close)}
      </AdminDialog>
      {message ? <p className="text-xs text-slate-500">{message}</p> : null}
    </div>
  );
}

const purchasableMaterialPageSize = 50;

export function PurchasableMaterialManager({ items, projects }: { items: PurchasableMaterial[]; projects: ProcurementProject[] }) {
  const router = useRouter();
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState("");
  const [query, setQuery] = useState("");
  const [page, setPage] = useState(1);
  const filteredItems = useMemo(() => {
    const normalizedQuery = query.trim().toLowerCase();
    if (normalizedQuery === "") {
      return items;
    }
    return items.filter((item) => purchasableMaterialSearchText(item).includes(normalizedQuery));
  }, [items, query]);
  const totalPages = Math.max(Math.ceil(filteredItems.length / purchasableMaterialPageSize), 1);
  const currentPage = Math.min(page, totalPages);
  const pagedItems = filteredItems.slice((currentPage - 1) * purchasableMaterialPageSize, currentPage * purchasableMaterialPageSize);

  async function submit(event: FormEvent<HTMLFormElement>, close?: () => void, item?: PurchasableMaterial) {
    event.preventDefault();
    const pendingKey = item ? `edit:${item.id}` : "save";
    setPending(pendingKey);
    setMessage("");
    const form = new FormData(event.currentTarget);
    const payload: PurchasableMaterialPayload = {
      idNo: String(form.get("idNo") ?? ""),
      sequenceNo: String(form.get("sequenceNo") ?? ""),
      procurementProjectId: String(form.get("procurementProjectId") ?? ""),
      procurementProject: String(form.get("procurementProject") ?? ""),
      procurementExpiresAt: "",
      projectName: String(form.get("projectName") ?? ""),
      brand: String(form.get("brand") ?? ""),
      spec: String(form.get("spec") ?? ""),
      unit: String(form.get("unit") ?? ""),
      purchasePrice: Number(form.get("purchasePrice") ?? 0),
      remark: String(form.get("remark") ?? ""),
      technicalRequirement: String(form.get("technicalRequirement") ?? ""),
      minSpec: String(form.get("minSpec") ?? ""),
    };
    try {
      if (item) {
        await browserPatch<PurchasableMaterial>(`/api/purchasable-materials/${item.id}`, payload);
      } else {
        await browserPost<PurchasableMaterial>("/api/purchasable-materials", payload);
      }
      setMessage(item ? "可采购物资已修改。" : "可采购物资已保存。");
      close?.();
      startTransition(() => router.refresh());
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "保存失败");
    } finally {
      setPending("");
    }
  }

  async function remove(id: string) {
    setPending(id);
    setMessage("");
    try {
      await browserDelete<PurchasableMaterial>(`/api/purchasable-materials/${id}`);
      setMessage("可采购物资已删除。");
      startTransition(() => router.refresh());
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "删除失败");
    } finally {
      setPending("");
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-2 sm:flex-row sm:flex-wrap">
        <AdminDialog
          description="标星字段必须填写，保存后可供用户申购选择。"
          maxWidth="max-w-4xl"
          title="新增可采购物资"
          trigger={
            <Button className="w-full sm:w-auto" type="button">
              <Save className="h-4 w-4" aria-hidden="true" />
              新增可采购物资
            </Button>
          }
        >
          {(close) => (
            <form className="space-y-4" onSubmit={(event) => submit(event, close)}>
              <PurchasableMaterialFields projects={projects} />
              <div className="flex justify-end">
                <Button disabled={pending === "save"} type="submit">
                  {pending === "save" ? "保存中..." : "保存"}
                </Button>
              </div>
            </form>
          )}
        </AdminDialog>
        <PurchasableMaterialImportForm />
        <DownloadButton filename="lirs-purchasable-materials-import-template.csv" label="下载模板" path="/api/purchasable-materials/import-template.csv" />
        <DownloadButton filename="lirs-purchasable-materials.csv" label="导出目录" path="/api/purchasable-materials/export.csv" />
      </div>
      <div className="grid gap-3 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-center">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" aria-hidden="true" />
          <input
            className="h-10 w-full rounded-md border bg-white pl-10 pr-3 text-sm"
            onChange={(event) => {
              setQuery(event.currentTarget.value);
              setPage(1);
            }}
            placeholder="搜索ID号、序号、项目、品牌、规格、单位"
            type="search"
            value={query}
          />
        </div>
        <p className="text-sm text-slate-500">共 {filteredItems.length} 条，每页 {purchasableMaterialPageSize} 条，第 {currentPage} / {totalPages} 页</p>
      </div>
      <div className="overflow-x-auto rounded-lg border">
        <table className="w-full min-w-[880px] text-left text-sm">
          <thead className="bg-slate-50 text-slate-500">
            <tr>
              <th className="p-3">ID号</th>
              <th className="p-3">序号</th>
              <th className="p-3">项目名称</th>
              <th className="p-3">品牌</th>
              <th className="p-3">规格</th>
              <th className="p-3">单位</th>
              <th className="p-3">采购价</th>
              <th className="w-44 p-3">操作</th>
            </tr>
          </thead>
          <tbody className="divide-y">
            {pagedItems.map((item) => (
              <tr key={item.id}>
                <td className="break-words p-3 align-top">{item.idNo}</td>
                <td className="break-words p-3 align-top">{item.sequenceNo}</td>
                <td className="break-words p-3 align-top">{item.projectName}</td>
                <td className="break-words p-3 align-top">{item.brand}</td>
                <td className="break-words p-3 align-top">{item.spec}</td>
                <td className="break-words p-3 align-top">{item.unit}</td>
                <td className="p-3 align-top font-bold">{formatMoney(item.purchasePrice)}</td>
                <td className="p-3 align-top">
                  <div className="flex flex-wrap gap-2">
                    <AdminDialog
                      description="采购项目名称及编号只在修改时展示，不在目录列表中展开。"
                      maxWidth="max-w-4xl"
                      title="修改可采购物资"
                      trigger={
                        <Button disabled={pending === `edit:${item.id}`} size="sm" type="button" variant="ghost">
                          <Pencil className="h-4 w-4" aria-hidden="true" />
                          修改
                        </Button>
                      }
                    >
                      {(close) => (
                        <form className="space-y-4" onSubmit={(event) => submit(event, close, item)}>
                          <PurchasableMaterialFields item={item} projects={projects} />
                          <div className="flex justify-end">
                            <Button disabled={pending === `edit:${item.id}`} type="submit">
                              {pending === `edit:${item.id}` ? "保存中..." : "保存修改"}
                            </Button>
                          </div>
                        </form>
                      )}
                    </AdminDialog>
                    <Button disabled={pending === item.id} onClick={() => remove(item.id)} size="sm" type="button" variant="ghost">
                      <Trash2 className="h-4 w-4" aria-hidden="true" />
                      删除
                    </Button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <div className="flex flex-col justify-between gap-3 sm:flex-row sm:items-center">
        <p className="text-sm text-slate-500">当前显示 {pagedItems.length} 条</p>
        <div className="flex gap-2">
          <Button disabled={currentPage <= 1} onClick={() => setPage((value) => Math.max(value - 1, 1))} size="sm" type="button" variant="outline">
            上一页
          </Button>
          <Button disabled={currentPage >= totalPages} onClick={() => setPage((value) => Math.min(value + 1, totalPages))} size="sm" type="button" variant="outline">
            下一页
          </Button>
        </div>
      </div>
      {filteredItems.length === 0 ? <p className="rounded-lg border p-4 text-sm text-slate-500">暂无匹配的可采购物资。</p> : null}
      {message ? <p className="text-sm text-slate-500">{message}</p> : null}
    </div>
  );
}

export function ProcurementProjectManager({ projects }: { projects: ProcurementProject[] }) {
  const router = useRouter();
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState("");
  async function submit(event: FormEvent<HTMLFormElement>, close?: () => void, project?: ProcurementProject) {
    event.preventDefault();
    const pendingKey = project ? `project:${project.id}` : "project:new";
    setPending(pendingKey);
    setMessage("");
    const form = new FormData(event.currentTarget);
    const payload: ProcurementProjectPayload = {
      name: String(form.get("name") ?? ""),
      expiresAt: String(form.get("expiresAt") ?? ""),
      status: String(form.get("status") ?? "active"),
    };
    try {
      if (project) {
        await browserPatch<ProcurementProject>(`/api/procurement-projects/${project.id}`, payload);
      } else {
        await browserPost<ProcurementProject>("/api/procurement-projects", payload);
      }
      setMessage(project ? "采购项目已修改。" : "采购项目已新增。");
      close?.();
      startTransition(() => router.refresh());
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "保存失败");
    } finally {
      setPending("");
    }
  }

  async function remove(id: string) {
    setPending(`project-delete:${id}`);
    setMessage("");
    try {
      await browserDelete<ProcurementProject>(`/api/procurement-projects/${id}`);
      setMessage("采购项目已停用。");
      startTransition(() => router.refresh());
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "停用失败");
    } finally {
      setPending("");
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-2 sm:flex-row sm:flex-wrap">
        <AdminDialog
          description="有效期为空表示长期有效；超过有效期后，关联物资不能再被申购。"
          maxWidth="max-w-3xl"
          title="新增采购项目"
          trigger={
            <Button className="w-full sm:w-auto" type="button">
              <Save className="h-4 w-4" aria-hidden="true" />
              新增采购项目
            </Button>
          }
        >
          {(close) => (
            <form className="space-y-4" onSubmit={(event) => submit(event, close)}>
              <ProcurementProjectFields />
              <div className="flex justify-end">
                <Button disabled={pending === "project:new"} type="submit">
                  {pending === "project:new" ? "保存中..." : "保存"}
                </Button>
              </div>
            </form>
          )}
        </AdminDialog>
      </div>
      <div className="overflow-x-auto rounded-lg border">
        <table className="w-full min-w-[720px] text-left text-sm">
          <thead className="bg-slate-50 text-slate-500">
            <tr>
              <th className="p-3">采购项目名称及编号</th>
              <th className="p-3">有效期</th>
              <th className="p-3">状态</th>
              <th className="w-44 p-3">操作</th>
            </tr>
          </thead>
          <tbody className="divide-y">
            {projects.map((project) => (
              <tr key={project.id}>
                <td className="break-words p-3 align-top">{project.name}</td>
                <td className="p-3 align-top">{project.expiresAt || "长期有效"}</td>
                <td className="p-3 align-top">
                  <span className={`rounded px-2 py-1 text-xs font-bold ${project.status !== "active" || procurementProjectExpired(project) ? "bg-amber-100 text-amber-800" : "bg-emerald-100 text-emerald-800"}`}>
                    {project.status !== "active" ? "已停用" : procurementProjectExpired(project) ? "已过期" : "可申购"}
                  </span>
                </td>
                <td className="p-3 align-top">
                  <div className="flex flex-wrap gap-2">
                    <AdminDialog
                      description="有效期为空表示长期有效；超过有效期后，关联物资不能再被申购。"
                      maxWidth="max-w-3xl"
                      title="修改采购项目"
                      trigger={
                        <Button disabled={pending === `project:${project.id}`} size="sm" type="button" variant="ghost">
                          <Pencil className="h-4 w-4" aria-hidden="true" />
                          修改
                        </Button>
                      }
                    >
                      {(close) => (
                        <form className="space-y-4" onSubmit={(event) => submit(event, close, project)}>
                          <ProcurementProjectFields project={project} />
                          <div className="flex justify-end">
                            <Button disabled={pending === `project:${project.id}`} type="submit">
                              {pending === `project:${project.id}` ? "保存中..." : "保存修改"}
                            </Button>
                          </div>
                        </form>
                      )}
                    </AdminDialog>
                    <Button disabled={pending === `project-delete:${project.id}`} onClick={() => remove(project.id)} size="sm" type="button" variant="ghost">
                      <Trash2 className="h-4 w-4" aria-hidden="true" />
                      停用
                    </Button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {projects.length === 0 ? <p className="rounded-lg border p-4 text-sm text-slate-500">暂无采购项目。</p> : null}
      {message ? <p className="text-sm text-slate-500">{message}</p> : null}
    </div>
  );
}

function PurchasableMaterialImportForm() {
  const router = useRouter();
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>, close?: () => void) {
    event.preventDefault();
    const file = new FormData(event.currentTarget).get("file");
    if (!(file instanceof File) || file.size === 0) {
      setMessage("请选择 CSV 或 XLSX 文件");
      return;
    }
    setPending(true);
    setMessage("");
    try {
      const response = await fetch(`/api/purchasable-materials/import?filename=${encodeURIComponent(file.name)}`, {
        body: await file.arrayBuffer(),
        credentials: "include",
        headers: { "Content-Type": file.name.endsWith(".xlsx") ? "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" : "text/csv; charset=utf-8" },
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
      startTransition(() => router.refresh());
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "导入失败");
    } finally {
      setPending(false);
    }
  }

  return (
    <div className="space-y-1">
      <AdminDialog
        description="支持 CSV 和 XLSX；单独一行的采购项目名称及编号会作为后续物资的采购项目字段，直到下一条项目行。"
        title="批量导入可采购物资"
        trigger={
          <Button className="w-full sm:w-auto" type="button" variant="outline">
            <Upload className="h-4 w-4" aria-hidden="true" />
            批量导入
          </Button>
        }
      >
        {(close) => (
          <form className="space-y-4" onSubmit={(event) => submit(event, close)}>
            <input accept=".csv,text/csv,.xlsx,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" className="w-full rounded-md border bg-white px-3 py-2 text-sm" name="file" required type="file" />
            <div className="flex justify-end">
              <Button disabled={pending} type="submit">
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

function PurchasableMaterialFields({ item, projects = [] }: { item?: PurchasableMaterial; projects?: ProcurementProject[] }) {
  const listId = `procurement-projects-${item?.id ?? "new"}`;
  return (
    <div className="grid gap-3 md:grid-cols-2">
      <Field defaultValue={item?.idNo} label="ID号*" name="idNo" required />
      <Field defaultValue={item?.sequenceNo} label="序号*" name="sequenceNo" required />
      <input name="procurementProjectId" type="hidden" value="" />
      <Field className="md:col-span-2" defaultValue={item?.procurementProject} label="采购项目名称及编号" list={listId} name="procurementProject" />
      <datalist id={listId}>
        {projects.map((project) => (
          <option key={project.id} value={project.name} />
        ))}
      </datalist>
      <Field className="md:col-span-2" defaultValue={item?.projectName} label="项目名称*" name="projectName" required />
      <Field defaultValue={item?.brand} label="品牌*" name="brand" required />
      <Field defaultValue={item?.spec} label="规格*" name="spec" required />
      <Field defaultValue={item?.unit} label="单位*" name="unit" required />
      <Field defaultValue={item?.purchasePrice} label="采购价（元）*" name="purchasePrice" required step="0.01" type="number" />
      <Field defaultValue={item?.minSpec} label="最小规格" name="minSpec" />
      <Field className="md:col-span-2" defaultValue={item?.remark} label="备注" name="remark" />
      <Field className="md:col-span-2" defaultValue={item?.technicalRequirement} label="技术要求" name="technicalRequirement" />
    </div>
  );
}

function ProcurementProjectFields({ project }: { project?: ProcurementProject }) {
  return (
    <div className="grid gap-3 md:grid-cols-2">
      <Field className="md:col-span-2" defaultValue={project?.name} label="采购项目名称及编号*" name="name" required />
      <Field defaultValue={project?.expiresAt} label="有效期" name="expiresAt" type="date" />
      <label className="block space-y-2">
        <span className="text-sm font-medium">状态</span>
        <select className="h-10 w-full rounded-md border bg-white px-3 text-sm" defaultValue={project?.status ?? "active"} name="status">
          <option value="active">启用</option>
          <option value="disabled">停用</option>
        </select>
      </label>
    </div>
  );
}

function Field({ className = "", defaultValue = "", label, list, name, required = false, step, type = "text" }: { className?: string; defaultValue?: string | number; label: string; list?: string; name: string; required?: boolean; step?: string; type?: string }) {
  return (
    <label className={`block space-y-2 ${className}`}>
      <span className="text-sm font-medium">{label}</span>
      <input className="h-10 w-full rounded-md border bg-white px-3 text-sm" defaultValue={defaultValue} list={list} min={type === "number" ? 0 : undefined} name={name} required={required} step={step} type={type} />
    </label>
  );
}

function Info({ className = "", label, value }: { className?: string; label: string; value: string }) {
  return (
    <div className={`min-w-0 ${className}`}>
      <p className="text-xs text-slate-500">{label}</p>
      <p className="mt-1 break-words font-medium text-slate-800">{value}</p>
    </div>
  );
}

function formatMoney(value: number) {
  return `¥${value.toFixed(2)}`;
}

function purchasableMaterialOptionLabel(item: PurchasableMaterial) {
  return `${item.idNo} ${item.projectName} ${item.brand} ${item.spec}`;
}

function purchasableMaterialExpired(item: PurchasableMaterial) {
  return item.procurementProjectStatus === "disabled" || dateExpired(item.procurementExpiresAt);
}

function procurementProjectExpired(project: ProcurementProject) {
  return dateExpired(project.expiresAt);
}

function dateExpired(value: string) {
  if (!value) {
    return false;
  }
  return value < new Date().toISOString().slice(0, 10);
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
  ]
    .join(" ")
    .toLowerCase();
}

function DownloadButton({ filename, label, path }: { filename: string; label: string; path: string }) {
  async function download() {
    const response = await fetch(path, { credentials: "include" });
    if (!response.ok) {
      return;
    }
    const blob = await response.blob();
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = filename;
    link.click();
    URL.revokeObjectURL(url);
  }

  return (
    <Button className="w-full sm:w-auto" onClick={download} type="button" variant="outline">
      <Download className="h-4 w-4" aria-hidden="true" />
      {label}
    </Button>
  );
}

export function MaterialPurchaseActions({
  id,
  status,
  canReview = false,
  canOrder = false,
  canReceive = false,
  canCancel = true,
}: {
  id: string;
  status: string;
  canReview?: boolean;
  canOrder?: boolean;
  canReceive?: boolean;
  canCancel?: boolean;
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
      const item = await browserPatch<MaterialPurchase>(path, payload);
      setMessage(`已更新为 ${purchaseStatusLabel(item.status)}`);
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
            description="填写审批意见后通过该申购单。"
            title="通过申购"
            trigger={
              <Button className="h-10 w-full sm:h-8 sm:w-auto" disabled={pending} onClick={() => setApproveComment("")} size="sm">
                通过
              </Button>
            }
          >
            {(close) => (
              <div className="space-y-4">
                <label className="block space-y-2">
                  <span className="text-sm font-medium">审批意见</span>
                  <input
                    className="h-10 w-full rounded border bg-white px-3 text-sm"
                    onChange={(event) => setApproveComment(event.target.value)}
                    placeholder="填写该申购单的审批意见"
                    value={approveComment}
                  />
                  <span className="block break-all text-xs text-slate-500">当前申购单 ID：{id}</span>
                </label>
                <div className="flex justify-end">
                  <Button className="w-full sm:w-auto" disabled={pending} onClick={() => patch(`/api/material-purchases/${id}/approve`, { comment: approveComment }, close)} type="button">
                    确认通过
                  </Button>
                </div>
              </div>
            )}
          </AdminDialog>
          <AdminDialog
            description="填写拒绝原因后驳回该申购单。"
            title="拒绝申购"
            trigger={
              <Button className="h-10 w-full sm:h-8 sm:w-auto" disabled={pending} onClick={() => setRejectComment("")} size="sm" variant="outline">
                拒绝
              </Button>
            }
          >
            {(close) => (
              <div className="space-y-4">
                <label className="block space-y-2">
                  <span className="text-sm font-medium">拒绝原因</span>
                  <input
                    className="h-10 w-full rounded border bg-white px-3 text-sm"
                    onChange={(event) => setRejectComment(event.target.value)}
                    placeholder="填写该申购单的拒绝原因"
                    value={rejectComment}
                  />
                  <span className="block break-all text-xs text-slate-500">当前申购单 ID：{id}</span>
                </label>
                <div className="flex justify-end">
                  <Button className="w-full sm:w-auto" disabled={pending} onClick={() => patch(`/api/material-purchases/${id}/reject`, { comment: rejectComment }, close)} type="button" variant="outline">
                    确认拒绝
                  </Button>
                </div>
              </div>
            )}
          </AdminDialog>
        </>
      ) : null}
      {status === "approved" && canOrder ? (
        <Button className="h-10 w-full sm:h-8 sm:w-auto" disabled={pending} onClick={() => patch(`/api/material-purchases/${id}/order`)} size="sm" variant="outline">
          标记下单
        </Button>
      ) : null}
      {(status === "approved" || status === "ordered") && canReceive ? (
        <Button className="h-10 w-full sm:h-8 sm:w-auto" disabled={pending} onClick={() => patch(`/api/material-purchases/${id}/receive`)} size="sm">
          到货入库
        </Button>
      ) : null}
      {canCancel && (status === "pending" || status === "approved" || status === "ordered") ? (
        <Button className="h-10 w-full sm:h-8 sm:w-auto" disabled={pending} onClick={() => patch(`/api/material-purchases/${id}/cancel`)} size="sm" variant="ghost">
          取消
        </Button>
      ) : null}
      {message ? <span className="text-xs text-slate-500 sm:basis-full">{message}</span> : null}
    </div>
  );
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
