"use client";

import { startTransition, useOptimistic, useState } from "react";
import { useRouter } from "next/navigation";
import { browserPatch, Reservation } from "@/lib/api";
import { reservationStatusLabel } from "@/lib/status-labels";
import { AdminDialog } from "@/components/admin-dialog";
import { Button } from "@/components/ui/button";

export function ReservationActions({
  id,
  status,
  canReview = true,
  canCheckIn = true,
  canCheckOut = true,
  canCancel = true,
}: {
  id: string;
  status: string;
  canReview?: boolean;
  canCheckIn?: boolean;
  canCheckOut?: boolean;
  canCancel?: boolean;
}) {
  const router = useRouter();
  const [message, setMessage] = useState("");
  const [approveComment, setApproveComment] = useState("");
  const [rejectComment, setRejectComment] = useState("");
  const [cancelReason, setCancelReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [visibleStatus, setVisibleStatus] = useOptimistic(status, (_currentStatus: string, nextStatus: string) => nextStatus);

  async function patch(path: string, nextStatus: string, payload?: unknown, close?: () => void) {
    setBusy(true);
    startTransition(() => {
      setVisibleStatus(nextStatus);
    });
    setMessage(`正在更新为 ${reservationStatusLabel(nextStatus)}`);
    try {
      const reservation = await browserPatch<Reservation>(path, payload);
      setMessage(`已更新为 ${reservationStatusLabel(reservation.status)}`);
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
      setBusy(false);
    }
  }

  return (
    <div className="flex flex-wrap items-center gap-2">
      <span className="w-fit rounded bg-slate-100 px-2 py-1 text-xs font-bold">{reservationStatusLabel(visibleStatus)}</span>
      {visibleStatus === "pending" && canReview ? (
        <>
          <AdminDialog
            description="填写审批意见后通过该预约。"
            title="通过预约"
            trigger={
              <Button className="h-10 w-full sm:h-8 sm:w-auto" disabled={busy} onClick={() => setApproveComment("")} size="sm">
                通过
              </Button>
            }
          >
            {(close) => (
              <div className="space-y-4">
                <label className="block space-y-2">
                  <span className="text-sm font-medium">审批意见</span>
                  <textarea
                    className="min-h-24 w-full rounded-lg border bg-white p-3 text-sm outline-none focus:ring-1 focus:ring-primary"
                    onChange={(event) => setApproveComment(event.target.value)}
                    placeholder="填写该预约的审批意见"
                    value={approveComment}
                  />
                  <span className="block break-all text-xs text-slate-500">当前预约 ID：{id}</span>
                </label>
                <div className="flex justify-end">
                  <Button className="w-full sm:w-auto" disabled={busy} onClick={() => patch(`/api/reservations/${id}/approve`, "approved", { comment: approveComment }, close)} type="button">
                    确认通过
                  </Button>
                </div>
              </div>
            )}
          </AdminDialog>
          <AdminDialog
            description="填写驳回原因后拒绝该预约，拒绝后会释放预约时段。"
            title="拒绝预约"
            trigger={
              <Button className="h-10 w-full sm:h-8 sm:w-auto" disabled={busy} onClick={() => setRejectComment("")} size="sm" variant="outline">
                拒绝
              </Button>
            }
          >
            {(close) => (
              <div className="space-y-4">
                <label className="block space-y-2">
                  <span className="text-sm font-medium">驳回原因</span>
                  <textarea
                    className="min-h-24 w-full rounded-lg border bg-white p-3 text-sm outline-none focus:ring-1 focus:ring-primary"
                    onChange={(event) => setRejectComment(event.target.value)}
                    placeholder="填写该预约的驳回原因，拒绝等同取消该预约申请"
                    value={rejectComment}
                  />
                  <span className="block break-all text-xs text-slate-500">当前预约 ID：{id}</span>
                </label>
                <div className="flex justify-end">
                  <Button className="w-full sm:w-auto" disabled={busy} onClick={() => patch(`/api/reservations/${id}/reject`, "rejected", { comment: rejectComment }, close)} type="button" variant="outline">
                    确认拒绝
                  </Button>
                </div>
              </div>
            )}
          </AdminDialog>
        </>
      ) : null}
      {visibleStatus === "approved" && canCheckIn ? (
        <Button className="h-10 w-full sm:h-8 sm:w-auto" disabled={busy} onClick={() => patch(`/api/reservations/${id}/check-in`, "in_use")} size="sm" variant="outline">
          签到
        </Button>
      ) : null}
      {visibleStatus === "in_use" && canCheckOut ? (
        <>
          <Button className="h-10 w-full sm:h-8 sm:w-auto" disabled={busy} onClick={() => patch(`/api/reservations/${id}/check-out`, "completed")} size="sm">
            签退并入账
          </Button>
        </>
      ) : null}
      {canCancel && !canReview && (visibleStatus === "pending" || visibleStatus === "approved") ? (
        <AdminDialog
          description="取消后该预约时段会释放，请填写取消原因。"
          title="取消预约"
          trigger={
            <Button className="h-10 w-full sm:h-8 sm:w-auto" disabled={busy} onClick={() => setCancelReason("")} size="sm" variant="ghost">
              取消
            </Button>
          }
        >
          {(close) => (
            <div className="space-y-4">
              <label className="block space-y-2">
                <span className="text-sm font-medium">取消原因</span>
                <textarea
                  className="min-h-24 w-full rounded-lg border bg-white p-3 text-sm outline-none focus:ring-1 focus:ring-primary"
                  onChange={(event) => setCancelReason(event.target.value)}
                  placeholder="填写取消该预约的原因"
                  value={cancelReason}
                />
                <span className="block break-all text-xs text-slate-500">当前预约 ID：{id}</span>
              </label>
              <div className="flex justify-end">
                <Button className="w-full sm:w-auto" disabled={busy} onClick={() => patch(`/api/reservations/${id}/cancel`, "cancelled", { reason: cancelReason }, close)} type="button" variant="outline">
                  确认取消
                </Button>
              </div>
            </div>
          )}
        </AdminDialog>
      ) : null}
      {message ? <span className="text-xs text-slate-500">{message}</span> : null}
    </div>
  );
}
