"use client";

import { FormEvent, startTransition, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { Pencil, Save, Search, Trash2, Upload } from "lucide-react";
import { browserDelete, browserPatch, browserPost, MaterialImportResult, ProcurementProject, PurchasableMaterial, PurchasableMaterialPayload } from "@/lib/api";
import { confirmTwice } from "@/lib/confirm";
import { AdminDialog } from "@/components/admin-dialog";
import { DownloadButton, Field, formatMoney, purchasableMaterialSearchText } from "@/components/material-purchase-shared";
import { Button } from "@/components/ui/button";

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
    if (item && !confirmTwice(`确定修改可采购物资“${item.projectName}”吗？`, "请再次确认。修改后新的申购会按更新后的目录信息登记。")) {
      return;
    }
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
    const item = items.find((entry) => entry.id === id);
    if (!confirmTwice(`确定删除可采购物资“${item?.projectName ?? id}”吗？`, "请再次确认。删除后该物资不会再出现在新建申购选择中。")) {
      return;
    }
    setPending(id);
    setMessage("");
    try {
      await browserDelete<PurchasableMaterial>(`/api/purchasable-materials/${id}`);
      setMessage("可采购物资已删除。");
      router.refresh();
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

function PurchasableMaterialImportForm() {
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
      const response = await fetch(`/api/purchasable-materials/import?filename=${encodeURIComponent(file.name)}`, {
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
        description="支持 CSV、XLS 和 XLSX；单独一行的采购项目名称及编号会作为后续物资的采购项目字段，直到下一条项目行。"
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
            <input accept=".csv,text/csv,.xls,application/vnd.ms-excel,.xlsx,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" className="w-full rounded-md border bg-white px-3 py-2 text-sm" name="file" required type="file" />
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
