"use client";

import { FormEvent, startTransition, useState } from "react";
import { useRouter } from "next/navigation";
import { ShoppingCart } from "lucide-react";
import { browserPatch, browserPost, Material, MaterialPurchase, MaterialPurchasePayload } from "@/lib/api";
import { AdminDialog } from "@/components/admin-dialog";
import { Button } from "@/components/ui/button";

export function MaterialPurchaseForm({ inline = false, material, materials }: { inline?: boolean; material?: Material; materials: Material[] }) {
  const router = useRouter();
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>, close?: () => void) {
    event.preventDefault();
    setPending(true);
    setMessage("");
    const formElement = event.currentTarget;
    const form = new FormData(formElement);
    const payload: MaterialPurchasePayload = {
      materialId: String(form.get("materialId") ?? ""),
      quantity: Number(form.get("quantity") ?? 0),
      estimatedUnitPrice: Number(form.get("estimatedUnitPrice") ?? 0),
      supplier: String(form.get("supplier") ?? ""),
      reason: String(form.get("reason") ?? ""),
    };
    try {
      const purchase = await browserPost<MaterialPurchase>("/api/material-purchases", payload);
      setMessage(`申购已提交：${purchase.materialName} x${purchase.quantity}`);
      formElement.reset();
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
              <select className="h-10 w-full rounded-md border bg-white px-3 text-sm" name="materialId" required>
                <option value="">选择资源</option>
                {materials.map((item) => (
                  <option key={item.id} value={item.id}>
                    {item.name} / 当前 {item.stock}
                    {item.unit}
                  </option>
                ))}
              </select>
            )}
            <div className="grid gap-3 sm:grid-cols-2">
              <input className="h-10 min-w-0 rounded-md border bg-white px-3 text-sm" min={1} name="quantity" placeholder="申购数量" required type="number" />
              <input className="h-10 min-w-0 rounded-md border bg-white px-3 text-sm" min={0} name="estimatedUnitPrice" placeholder="预计单价" required step="0.01" type="number" />
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
          <Button className="w-full" disabled={pending || materials.length === 0}>
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
