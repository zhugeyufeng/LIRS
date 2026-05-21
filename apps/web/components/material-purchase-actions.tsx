"use client";

import { startTransition, useOptimistic, useState } from "react";
import { useRouter } from "next/navigation";
import { browserPatch, MaterialPurchase, PurchasableMaterial } from "@/lib/api";
import { confirmTwice } from "@/lib/confirm";
import { materialPurchaseStatusLabel } from "@/lib/status-labels";
import { AdminDialog } from "@/components/admin-dialog";
import { MaterialPurchaseForm } from "@/components/material-purchase-form";
import { Button } from "@/components/ui/button";

export function MaterialPurchaseActions({
  id,
  purchase,
  purchasableMaterials = [],
  status,
  canReview = false,
  canOrder = false,
  canReceive = false,
  canCancel = true,
}: {
  id: string;
  purchase: MaterialPurchase;
  purchasableMaterials?: PurchasableMaterial[];
  status: string;
  canReview?: boolean;
  canOrder?: boolean;
  canReceive?: boolean;
  canCancel?: boolean;
}) {
  const router = useRouter();
  const [message, setMessage] = useState("");
  const [returnComment, setReturnComment] = useState("");
  const [pending, setPending] = useState(false);
  const [visibleStatus, setVisibleStatus] = useOptimistic(status, (_currentStatus: string, nextStatus: string) => nextStatus);

  async function patch(path: string, payload: unknown | undefined, nextStatus: string, close: (() => void) | undefined, firstConfirm: string, secondConfirm: string) {
    if (!confirmTwice(firstConfirm, secondConfirm)) {
      return;
    }
    setPending(true);
    startTransition(() => {
      setVisibleStatus(nextStatus);
    });
    setMessage(`正在更新为 ${materialPurchaseStatusLabel(nextStatus)}`);
    try {
      const item = await browserPatch<MaterialPurchase>(path, payload);
      setMessage(`已更新为 ${materialPurchaseStatusLabel(item.status)}`);
      close?.();
      startTransition(() => {
        router.refresh();
      });
    } catch (error) {
      startTransition(() => {
        setVisibleStatus(status);
      });
      setMessage(error instanceof Error ? error.message : "操作失败");
    } finally {
      setPending(false);
    }
  }

  const locked = purchase.monthlyConfirmed;

  return (
    <div className="grid w-full gap-2 sm:flex sm:flex-wrap sm:items-center">
      <span className="w-fit rounded bg-slate-100 px-2 py-1 text-xs font-bold">{materialPurchaseStatusLabel(visibleStatus)}</span>
      {visibleStatus === "registered" && canReview && !locked ? (
        <AdminDialog
          description="退回后申请人可以修改申购物品、数量、单价、供应商和原因，再重新提交。"
          title="退回修改"
          trigger={
            <Button className="h-10 w-full sm:h-8 sm:w-auto" disabled={pending} onClick={() => setReturnComment("")} size="sm" variant="outline">
              退回修改
            </Button>
          }
        >
          {(close) => (
            <div className="space-y-4">
              <label className="block space-y-2">
                <span className="text-sm font-medium">退回说明</span>
                <input
                  className="h-10 w-full rounded border bg-white px-3 text-sm"
                  onChange={(event) => setReturnComment(event.target.value)}
                  placeholder="填写需要申请人修改的内容"
                  value={returnComment}
                />
                <span className="block break-all text-xs text-slate-500">申购流水号：{purchase.purchaseSerialNo || id}</span>
              </label>
              <div className="flex justify-end">
                <Button className="w-full sm:w-auto" disabled={pending} onClick={() => patch(`/api/material-purchases/${id}/return`, { comment: returnComment }, "returned", close, `确定退回申购单“${purchase.purchaseSerialNo || id}”吗？`, "请再次确认。退回后申请人需要修改并重新提交。")} type="button" variant="outline">
                  确认退回
                </Button>
              </div>
            </div>
          )}
        </AdminDialog>
      ) : null}
      {visibleStatus === "returned" && !locked && (canCancel || canReview) ? (
        <AdminDialog
          description="退回后的申购单可以修改内容并重新登记，仍保留原流水号。"
          maxWidth="max-w-3xl"
          title="修改退回申购"
          trigger={
            <Button className="h-10 w-full sm:h-8 sm:w-auto" disabled={pending} size="sm" variant="outline">
              修改
            </Button>
          }
        >
          {(close) => <MaterialPurchaseForm inline materials={[]} onSaved={close} purchase={purchase} purchasableMaterials={purchasableMaterials} />}
        </AdminDialog>
      ) : null}
      {(visibleStatus === "registered" || visibleStatus === "approved") && canOrder ? (
        <Button className="h-10 w-full sm:h-8 sm:w-auto" disabled={pending} onClick={() => patch(`/api/material-purchases/${id}/order`, undefined, "ordered", undefined, `确定将申购单“${purchase.purchaseSerialNo || id}”标记为已下单吗？`, "请再次确认。该申购单会进入已下单状态。")} size="sm" variant="outline">
          标记下单
        </Button>
      ) : null}
      {(visibleStatus === "registered" || visibleStatus === "approved" || visibleStatus === "ordered") && canReceive ? (
        <Button className="h-10 w-full sm:h-8 sm:w-auto" disabled={pending} onClick={() => patch(`/api/material-purchases/${id}/receive`, undefined, "received", undefined, `确定将申购单“${purchase.purchaseSerialNo || id}”到货入库吗？`, "请再次确认。系统会按关联资源写入库存流水。")} size="sm">
          到货入库
        </Button>
      ) : null}
      {canCancel && !locked && (visibleStatus === "registered" || visibleStatus === "approved" || visibleStatus === "returned" || visibleStatus === "ordered") ? (
        <Button className="h-10 w-full sm:h-8 sm:w-auto" disabled={pending} onClick={() => patch(`/api/material-purchases/${id}/cancel`, undefined, "cancelled", undefined, `确定取消申购单“${purchase.purchaseSerialNo || id}”吗？`, "请再次确认。取消后该申购单不能继续下单或入库。")} size="sm" variant="ghost">
          取消
        </Button>
      ) : null}
      {locked && (visibleStatus === "registered" || visibleStatus === "approved" || visibleStatus === "returned" || visibleStatus === "ordered") ? <span className="text-xs text-slate-500">本月已确认，不能退回、取消或修改。</span> : null}
      {message ? <span className="text-xs text-slate-500 sm:basis-full">{message}</span> : null}
    </div>
  );
}
