"use client";

import { FormEvent, startTransition, useState } from "react";
import { useRouter } from "next/navigation";
import { CalendarPlus } from "lucide-react";
import { browserPost, Instrument, Reservation, ReservationPayload } from "@/lib/api";
import { formatServiceWindow } from "@/lib/instrument-rules";
import { Button } from "@/components/ui/button";

export function ReservationForm({ instrument }: { instrument: Instrument }) {
  const router = useRouter();
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);
  const intervalHours = Math.max(instrument.bookingIntervalHours, 1);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setMessage("");
    const formElement = event.currentTarget;
    const form = new FormData(formElement);
    const startTime = toIsoDateTime(form.get("startTime"));
    const endTime = toIsoDateTime(form.get("endTime"));
    if (!startTime || !endTime) {
      setPending(false);
      setMessage("请选择有效的预约开始和结束时间。");
      return;
    }
    const payload: ReservationPayload = {
      instrumentId: instrument.id,
      purpose: String(form.get("purpose") ?? ""),
      startTime,
      endTime,
    };
    payload.idempotencyKey = `${payload.instrumentId}:${payload.purpose}:${payload.startTime}:${payload.endTime}`.toLowerCase();
    try {
      const reservation = await browserPost<Reservation>("/api/reservations", payload);
      setMessage(`预约已提交，当前状态：${reservation.status}，预计费用 ${(reservation.fee ?? 0).toFixed(2)} 元。`);
      formElement.reset();
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
    <form className="space-y-4" onSubmit={submit}>
      <div className="flex items-center gap-2 text-sm font-semibold">
        <CalendarPlus className="h-4 w-4 text-primary" />
        新建预约
      </div>
      <input name="instrumentId" type="hidden" value={instrument.id} />
      <div className="rounded-lg border bg-slate-50 p-3 text-sm">
        <p className="font-medium text-slate-800">{instrument.name}</p>
        <p className="mt-1 text-xs text-slate-500">
          {instrument.department} / {instrument.location} / ¥{instrument.hourlyRate}/小时
        </p>
      </div>
      <input className="h-10 w-full rounded-md border bg-white px-3 text-sm" name="purpose" placeholder="实验用途" required />
      <div className="grid gap-3 sm:grid-cols-2">
        <div className="space-y-1">
          <p className="text-[10px] font-bold uppercase text-slate-400">开始时间</p>
          <input className="h-10 w-full rounded-md border bg-white px-3 text-sm" name="startTime" required step={intervalHours * 3600} type="datetime-local" />
        </div>
        <div className="space-y-1">
          <p className="text-[10px] font-bold uppercase text-slate-400">结束时间</p>
          <input className="h-10 w-full rounded-md border bg-white px-3 text-sm" name="endTime" required step={intervalHours * 3600} type="datetime-local" />
        </div>
      </div>
      <p className="text-xs leading-5 text-slate-500">
        开放 {formatServiceWindow(instrument)}，可预约未来 {instrument.bookingWindowDays} 天，按 {intervalHours} 小时时段提交。
      </p>
      <Button className="w-full sm:w-auto" disabled={pending} type="submit">
        {pending ? "提交中..." : "提交预约"}
      </Button>
      {message ? <p className="rounded-md bg-slate-100 p-3 text-sm">{message}</p> : null}
    </form>
  );
}

function toIsoDateTime(value: FormDataEntryValue | null) {
  const date = new Date(String(value ?? ""));
  return Number.isNaN(date.getTime()) ? "" : date.toISOString();
}
