"use client";

import { FormEvent, startTransition, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { Search, ShoppingCart } from "lucide-react";
import { browserPatch, browserPost, Material, MaterialPurchase, MaterialPurchasePayload, PurchasableMaterial } from "@/lib/api";
import { confirmTwice } from "@/lib/confirm";
import { AdminDialog } from "@/components/admin-dialog";
import { formatMoney, Info, purchasableMaterialExpired, purchasableMaterialOptionLabel, purchasableMaterialSearchText } from "@/components/material-purchase-shared";
import { Button } from "@/components/ui/button";

export function MaterialPurchaseForm({
  inline = false,
  material,
  materials,
  onSaved,
  purchase,
  purchasableMaterials = [],
}: {
  inline?: boolean;
  material?: Material;
  materials: Material[];
  onSaved?: () => void;
  purchase?: MaterialPurchase;
  purchasableMaterials?: PurchasableMaterial[];
}) {
  const router = useRouter();
  const editing = Boolean(purchase);
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);
  const [selectedPurchasableId, setSelectedPurchasableId] = useState(purchase?.purchasableMaterialId ?? "");
  const [purchasableSearch, setPurchasableSearch] = useState(purchase ? materialPurchaseOptionLabel(purchase) : "");
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
    if (purchase?.monthlyConfirmed) {
      setMessage("该申购单所在月份已确认，不能修改后重新提交。");
      return;
    }
    if (editing && !confirmTwice(`确定修改并重新提交申购单“${purchase?.purchaseSerialNo || purchase?.id}”吗？`, "请再次确认。重新提交后该申购单会回到已登记状态。")) {
      return;
    }
    setPending(true);
    try {
      const saved = editing && purchase
        ? await browserPatch<MaterialPurchase>(`/api/material-purchases/${purchase.id}`, payload)
        : await browserPost<MaterialPurchase>("/api/material-purchases", payload);
      setMessage(`${editing ? "申购已修改并重新提交" : "申购已登记"}：${saved.purchaseSerialNo || saved.id} / ${saved.materialName} x${saved.quantity}`);
      if (!editing) {
        formElement.reset();
        setSelectedPurchasableId("");
        setPurchasableSearch("");
      }
      close?.();
      onSaved?.();
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
        <input className="h-10 min-w-0 rounded-md border bg-white px-3 text-sm" defaultValue={purchase?.quantity} min={1} name="quantity" placeholder="申购数量" required type="number" />
        <input className="h-10 min-w-0 rounded-md border bg-white px-3 text-sm" defaultValue={purchase?.estimatedUnitPrice ?? (material ? undefined : selectedPurchasable?.purchasePrice)} key={`${selectedPurchasable?.id ?? "estimatedUnitPrice"}:${purchase?.id ?? "new"}`} min={0} name="estimatedUnitPrice" placeholder="预计单价" required step="0.01" type="number" />
      </div>
      <input className="h-10 w-full rounded-md border bg-white px-3 text-sm" defaultValue={purchase?.supplier} name="supplier" placeholder="供应商，可留空沿用资源资料" />
      <input className="h-10 w-full rounded-md border bg-white px-3 text-sm" defaultValue={purchase?.reason} name="reason" placeholder="申购原因" required />
      <div className="flex justify-end">
        <Button className="w-full sm:w-auto" disabled={pending} type="submit">
          {pending ? "提交中..." : editing ? "修改并重新提交" : "登记申购"}
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
        description="申购登记后会生成独立流水号，管理员可退回修改、标记下单或到货入库。"
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

function materialPurchaseOptionLabel(item: MaterialPurchase) {
  return `${item.purchaseIdNo} ${item.purchaseItemName || item.materialName} ${item.purchaseBrand} ${item.purchaseSpec}`;
}
