"use client";

import { FormEvent, startTransition, useOptimistic, useState } from "react";
import { useRouter } from "next/navigation";
import { PlusCircle, Save } from "lucide-react";
import { browserPatch, browserPost, Instrument, MaintenanceOrder, MaintenancePayload } from "@/lib/api";
import { maintenanceStatusLabel } from "@/lib/status-labels";
import { AdminDialog } from "@/components/admin-dialog";
import { Button } from "@/components/ui/button";

export function MaintenanceForm({ instruments }: { instruments: Instrument[] }) {
  const router = useRouter();
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>, close?: () => void) {
    event.preventDefault();
    setPending(true);
    setMessage("");
    const formElement = event.currentTarget;
    const form = new FormData(formElement);
    const startTime = toIsoDateTime(form.get("startTime"));
    const endTime = toIsoDateTime(form.get("endTime"));
    if (!startTime || !endTime) {
      setPending(false);
      setMessage("请选择有效的维护开始和结束时间。");
      return;
    }
    const payload: MaintenancePayload = {
      instrumentId: String(form.get("instrumentId") ?? ""),
      type: String(form.get("type") ?? "routine"),
      priority: String(form.get("priority") ?? "normal"),
      handler: String(form.get("handler") ?? ""),
      description: String(form.get("description") ?? ""),
      startTime,
      endTime,
    };
    try {
      const order = await browserPost<MaintenanceOrder>("/api/maintenance", payload);
      setMessage(`维护工单已创建：${order.instrumentName}`);
      formElement.reset();
      close?.();
      startTransition(() => {
        router.refresh();
      });
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "创建失败");
    } finally {
      setPending(false);
    }
  }

  return (
    <div className="space-y-2">
      <AdminDialog
        description="维护窗口会联动仪器可预约状态，并取消受影响的未开始预约。"
        maxWidth="max-w-3xl"
        title="创建维护窗口"
        trigger={
          <Button className="w-full" disabled={pending || instruments.length === 0}>
            <PlusCircle className="h-4 w-4" aria-hidden="true" />
            创建维护工单
          </Button>
        }
      >
        {(close) => (
          <form className="space-y-4" onSubmit={(event) => submit(event, close)}>
            <select className="h-10 w-full rounded-md border bg-white px-3 text-sm" name="instrumentId" required>
              <option value="">选择仪器</option>
              {instruments.map((item) => (
                <option key={item.id} value={item.id}>
                  {item.name} / {item.location}
                </option>
              ))}
            </select>
            <div className="grid gap-3 md:grid-cols-2">
              <select className="h-10 rounded-md border bg-white px-3 text-sm" name="type" defaultValue="routine">
                <option value="routine">例行维护</option>
                <option value="fault">故障维护</option>
                <option value="emergency">紧急维护</option>
              </select>
              <select className="h-10 rounded-md border bg-white px-3 text-sm" name="priority" defaultValue="normal">
                <option value="normal">普通</option>
                <option value="high">高</option>
                <option value="urgent">紧急</option>
              </select>
            </div>
            <input className="h-10 w-full rounded-md border bg-white px-3 text-sm" name="handler" placeholder="处理人，留空则先登记为已上报" />
            <input className="h-10 w-full rounded-md border bg-white px-3 text-sm" name="description" placeholder="维护说明" required />
            <div className="grid gap-3 md:grid-cols-2">
              <input className="h-10 rounded-md border bg-white px-3 text-sm" name="startTime" required step={3600} type="datetime-local" />
              <input className="h-10 rounded-md border bg-white px-3 text-sm" name="endTime" required step={3600} type="datetime-local" />
            </div>
            <div className="flex justify-end">
              <Button className="w-full sm:w-auto" disabled={pending} type="submit">
                <Save className="h-4 w-4" aria-hidden="true" />
                {pending ? "创建中..." : "创建维护工单"}
              </Button>
            </div>
          </form>
        )}
      </AdminDialog>
      {message ? <p className="text-xs text-slate-500">{message}</p> : null}
    </div>
  );
}

function toIsoDateTime(value: FormDataEntryValue | null) {
  const date = new Date(String(value ?? ""));
  return Number.isNaN(date.getTime()) ? "" : date.toISOString();
}

export function MaintenanceCompleteButton({ id, status }: { id: string; status: string }) {
  const router = useRouter();
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);
  const [visibleStatus, setVisibleStatus] = useOptimistic(status, (_currentStatus: string, nextStatus: string) => nextStatus);

  async function patch(path: string, nextStatus: string, payload?: unknown) {
    setPending(true);
    startTransition(() => {
      setVisibleStatus(nextStatus);
    });
    setMessage(`正在更新为：${maintenanceStatusLabel(nextStatus)}`);
    try {
      const order = await browserPatch<MaintenanceOrder>(path, payload);
      setMessage(`已更新为：${maintenanceStatusLabel(order.status)}`);
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
    <div className="flex flex-wrap items-center gap-2">
      <span className="w-fit rounded bg-slate-100 px-2 py-1 text-xs font-bold">{maintenanceStatusLabel(visibleStatus)}</span>
      {(visibleStatus === "reported" || visibleStatus === "assigned") ? (
        <Button className="h-10 w-full sm:h-8 sm:w-auto" disabled={pending} onClick={() => patch(`/api/maintenance/${id}/start`, "in_progress")} size="sm" variant="outline">
          开始处理
        </Button>
      ) : null}
      {visibleStatus !== "completed" && visibleStatus !== "cancelled" ? (
        <Button className="h-10 w-full sm:h-8 sm:w-auto" disabled={pending} onClick={() => patch(`/api/maintenance/${id}/complete`, "completed", { result: "维护完成，仪器恢复可用。" })} size="sm">
          完成
        </Button>
      ) : null}
      {visibleStatus !== "completed" && visibleStatus !== "cancelled" ? (
        <Button className="h-10 w-full sm:h-8 sm:w-auto" disabled={pending} onClick={() => patch(`/api/maintenance/${id}/cancel`, "cancelled", { reason: "维护计划取消" })} size="sm" variant="ghost">
          取消
        </Button>
      ) : null}
      {message ? <span className="text-xs text-slate-500">{message}</span> : null}
    </div>
  );
}
