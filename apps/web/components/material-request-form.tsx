"use client";

import { FormEvent, startTransition, useOptimistic, useState } from "react";
import { useRouter } from "next/navigation";
import { ClipboardList } from "lucide-react";
import { browserPatch, browserPost, Material, MaterialRequest, MaterialRequestPayload } from "@/lib/api";
import { AdminDialog } from "@/components/admin-dialog";
import { Button } from "@/components/ui/button";

export function MaterialRequestDialog({ buttonClassName, material }: { buttonClassName?: string; material: Material }) {
  return (
    <AdminDialog
      description="选择唯一编号并填写实验用途后提交。若该资源需要审批，将进入待审批状态。"
      title={`申领资源：${material.name}`}
      trigger={
        <Button className={buttonClassName ?? "w-full sm:w-auto"}>
          <ClipboardList className="h-4 w-4" aria-hidden="true" />
          申领
        </Button>
      }
    >
      {(close) => <MaterialRequestForm close={close} material={material} materials={[material]} />}
    </AdminDialog>
  );
}

export function MaterialRequestForm({ close, material, materials }: { close?: () => void; material?: Material; materials: Material[] }) {
  const router = useRouter();
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);
  const [selectedMaterialId, setSelectedMaterialId] = useState(material?.id ?? "");
  const selectedMaterial = material ?? materials.find((item) => item.id === selectedMaterialId);
  const availableUnits = (selectedMaterial?.units ?? []).filter(isRequestableUnit);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setMessage("");
    const formElement = event.currentTarget;
    const form = new FormData(formElement);
    const payload: MaterialRequestPayload = {
      materialId: String(form.get("materialId") ?? ""),
      unitId: String(form.get("unitId") ?? ""),
      quantity: 1,
      purpose: String(form.get("purpose") ?? ""),
    };
    try {
      const request = await browserPost<MaterialRequest>("/api/material-requests", payload);
      setMessage(`申领已提交：${request.materialName} x${request.quantity}`);
      formElement.reset();
      setSelectedMaterialId(material?.id ?? "");
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

  return (
    <form className="space-y-3" onSubmit={submit}>
      {material ? (
        <div className="rounded-md border bg-slate-50 px-3 py-2 text-sm">
          <input name="materialId" type="hidden" value={material.id} />
          <p className="text-xs text-slate-500">当前申领资源</p>
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
              {item.name} / 库存 {item.stock}
              {item.unit}
            </option>
          ))}
        </select>
      )}
      {selectedMaterial ? (
        <label className="block space-y-2">
          <span className="text-sm font-medium">领用编号</span>
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
          <span className="block text-xs text-slate-500">{availableUnits.length === 0 ? "暂无可领用编号，请联系试剂管理员入库。" : `该资源共有 ${availableUnits.length} 个可领用编号，单次申领 1 个最小单位。`}</span>
        </label>
      ) : null}
      <input name="quantity" type="hidden" value="1" />
      <input className="h-10 w-full rounded-md border bg-white px-3 text-sm" name="purpose" placeholder="实验用途" required />
      <Button className="w-full sm:w-auto" disabled={pending || !selectedMaterial || availableUnits.length === 0} type="submit">
        {pending ? "提交中..." : "提交申领"}
      </Button>
      {message ? <p className="text-sm text-slate-500">{message}</p> : null}
    </form>
  );
}

function isRequestableUnit(unit: Material["units"][number]) {
  if (unit.status !== "available") {
    return false;
  }
  if (!unit.expiresAt) {
    return true;
  }
  return unit.expiresAt >= new Date().toISOString().slice(0, 10);
}

export function MaterialRequestActions({
  id,
  status,
  canReview = false,
  canOutbound = false,
  canCancel = true,
}: {
  id: string;
  status: string;
  canReview?: boolean;
  canOutbound?: boolean;
  canCancel?: boolean;
}) {
  const router = useRouter();
  const [message, setMessage] = useState("");
  const [approveComment, setApproveComment] = useState("");
  const [rejectComment, setRejectComment] = useState("");
  const [pending, setPending] = useState(false);
  const [visibleStatus, setVisibleStatus] = useOptimistic(status, (_currentStatus: string, nextStatus: string) => nextStatus);

  async function patch(path: string, nextStatus: string, payload?: unknown, close?: () => void) {
    setPending(true);
    startTransition(() => {
      setVisibleStatus(nextStatus);
    });
    setMessage(`正在更新为 ${materialRequestStatusLabel(nextStatus)}`);
    try {
      const item = await browserPatch<MaterialRequest>(path, payload);
      setMessage(`已更新为 ${materialRequestStatusLabel(item.status)}`);
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

  return (
    <div className="grid w-full gap-2 sm:flex sm:flex-wrap sm:items-center">
      <span className="w-fit rounded bg-slate-100 px-2 py-1 text-xs font-bold">{materialRequestStatusLabel(visibleStatus)}</span>
      {visibleStatus === "pending" && canReview ? (
        <>
          <AdminDialog
            description="填写审批意见后通过该申领单。"
            title="通过申领"
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
                    placeholder="填写该申领单的审批意见"
                    value={approveComment}
                  />
                  <span className="block break-all text-xs text-slate-500">当前申领单 ID：{id}</span>
                </label>
                <div className="flex justify-end">
                  <Button className="w-full sm:w-auto" disabled={pending} onClick={() => patch(`/api/material-requests/${id}/approve`, "approved", { comment: approveComment }, close)} type="button">
                    确认通过
                  </Button>
                </div>
              </div>
            )}
          </AdminDialog>
          <AdminDialog
            description="填写拒绝原因后驳回该申领单。"
            title="拒绝申领"
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
                    placeholder="填写该申领单的拒绝原因"
                    value={rejectComment}
                  />
                  <span className="block break-all text-xs text-slate-500">当前申领单 ID：{id}</span>
                </label>
                <div className="flex justify-end">
                  <Button className="w-full sm:w-auto" disabled={pending} onClick={() => patch(`/api/material-requests/${id}/reject`, "rejected", { comment: rejectComment }, close)} type="button" variant="outline">
                    确认拒绝
                  </Button>
                </div>
              </div>
            )}
          </AdminDialog>
        </>
      ) : null}
      {visibleStatus === "approved" && canOutbound ? (
        <Button className="h-10 w-full sm:h-8 sm:w-auto" disabled={pending} onClick={() => patch(`/api/material-requests/${id}/outbound`, "outbound")} size="sm">
          出库
        </Button>
      ) : null}
      {canCancel && (visibleStatus === "pending" || visibleStatus === "approved") ? (
        <Button className="h-10 w-full sm:h-8 sm:w-auto" disabled={pending} onClick={() => patch(`/api/material-requests/${id}/cancel`, "cancelled")} size="sm" variant="ghost">
          取消
        </Button>
      ) : null}
      {message ? <span className="text-xs text-slate-500 sm:basis-full">{message}</span> : null}
    </div>
  );
}

function materialRequestStatusLabel(status: string) {
  const labels: Record<string, string> = {
    pending: "待审批",
    approved: "已通过",
    rejected: "已拒绝",
    outbound: "已出库",
    cancelled: "已取消",
  };
  return labels[status] ?? status;
}
