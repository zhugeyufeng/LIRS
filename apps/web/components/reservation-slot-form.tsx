"use client";

import { FormEvent, startTransition, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { CalendarPlus } from "lucide-react";
import { browserPost, Instrument, Reservation, ReservationBatchPayload, Slot } from "@/lib/api";
import { formatServiceWindow } from "@/lib/instrument-rules";
import { Button } from "@/components/ui/button";

type SelectableSlot = Slot & {
  disabled: boolean;
};

type SlotRange = {
  startTime: string;
  endTime: string;
};

const statusLabels: Record<string, string> = {
  available: "可约",
  occupied: "已占用",
  maintenance: "维护",
  disabled: "停用",
  unavailable: "不可约",
};

export function ReservationSlotForm({
  instrument,
  slots,
  bookableAfter,
}: {
  instrument: Instrument;
  slots: Slot[];
  bookableAfter: string;
}) {
  const router = useRouter();
  const [selectedSlotKeys, setSelectedSlotKeys] = useState<string[]>([]);
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);

  const groupedSlots = useMemo(() => {
    const bookableAfterDate = new Date(bookableAfter);
    return slots.reduce<Record<string, SelectableSlot[]>>((groups, slot) => {
      const start = new Date(slot.startTime);
      const dateLabel = start.toLocaleDateString("zh-CN", {
        weekday: "short",
        month: "2-digit",
        day: "2-digit",
        timeZone: "Asia/Shanghai",
      });
      groups[dateLabel] ??= [];
      groups[dateLabel].push({
        ...slot,
        disabled: slot.status !== "available" || start < bookableAfterDate,
      });
      return groups;
    }, {});
  }, [bookableAfter, slots]);

  const selectableSlots = useMemo(() => Object.values(groupedSlots).flat(), [groupedSlots]);
  const orderedSlots = useMemo(() => [...selectableSlots].sort(compareSlots), [selectableSlots]);
  const selectedSlots = useMemo(
    () => orderedSlots.filter((slot) => selectedSlotKeys.includes(slotKey(slot))),
    [orderedSlots, selectedSlotKeys],
  );
  const selectedSummary = selectedSlots.length > 0 ? slotSelectionSummary(selectedSlots) : null;
  const intervalHours = Math.max(instrument.bookingIntervalHours, 1);

  function handleSlotClick(slot: SelectableSlot) {
    if (slot.disabled || pending) {
      return;
    }
    setMessage("");
    const key = slotKey(slot);

    if (selectedSlotKeys.includes(key)) {
      setSelectedSlotKeys((current) => current.filter((item) => item !== key));
      return;
    }

    const nextSlots = [...selectedSlots, slot].sort(compareSlots);
    const nextDurationHours = selectedDurationHours(nextSlots);
    if (instrument.maxBookingHours > 0 && nextDurationHours > instrument.maxBookingHours) {
      setMessage(`最长预约 ${instrument.maxBookingHours} 小时。`);
      return;
    }
    setSelectedSlotKeys(nextSlots.map(slotKey));
  }

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setMessage("");
    if (!selectedSummary) {
      setMessage("请选择预约时段。");
      return;
    }
    setPending(true);
    const formElement = event.currentTarget;
    const form = new FormData(formElement);
    const payload: ReservationBatchPayload = {
      instrumentId: instrument.id,
      purpose: String(form.get("purpose") ?? ""),
      periods: selectedSummary.ranges.map((range) => ({
        startTime: new Date(range.startTime).toISOString(),
        endTime: new Date(range.endTime).toISOString(),
      })),
    };
    payload.idempotencyKey = `${payload.instrumentId}:${payload.purpose}:${payload.periods.map((period) => `${period.startTime}-${period.endTime}`).join("|")}`.toLowerCase();
    try {
      const reservations = await browserPost<Reservation[]>("/api/reservations/batch", payload);
      const fee = reservations.reduce((sum, reservation) => sum + (reservation.fee ?? 0), 0);
      setMessage(`预约已提交 ${reservations.length} 段，预计费用 ${fee.toFixed(2)} 元。`);
      formElement.reset();
      setSelectedSlotKeys([]);
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
    <form className="space-y-5" onSubmit={submit}>
      <div className="rounded-lg border bg-slate-50 p-4">
        <div className="flex items-start gap-3">
          <CalendarPlus className="mt-0.5 h-5 w-5 text-primary" aria-hidden="true" />
          <div className="min-w-0">
            <p className="font-bold text-slate-900">{instrument.name}</p>
            <p className="mt-1 text-sm text-slate-500">
              开放 {formatServiceWindow(instrument)}，每 {intervalHours} 小时一个预约周期，可跨天选择不连续周期，累计最长 {instrument.maxBookingHours} 小时。
            </p>
          </div>
        </div>
      </div>

      <div className="space-y-4">
        {Object.entries(groupedSlots).map(([dateLabel, daySlots]) => (
          <section className="rounded-lg border bg-white p-3" key={dateLabel}>
            <div className="mb-3 flex items-center justify-between gap-3">
              <h2 className="text-sm font-bold text-slate-800">{dateLabel}</h2>
              <span className="text-xs text-slate-500">{daySlots.filter((slot) => !slot.disabled).length} 个可约周期</span>
            </div>
            <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-6">
              {daySlots.map((slot) => {
                const key = slotKey(slot);
                const selected = selectedSlotKeys.includes(key);
                return (
                  <button
                    className={[
                      "min-h-16 rounded-md border px-2 py-2 text-left text-xs transition",
                      selected ? "border-primary bg-primary text-white shadow-sm" : slotClassName(slot),
                    ].join(" ")}
                    disabled={slot.disabled || pending}
                    key={key}
                    onClick={() => handleSlotClick(slot)}
                    type="button"
                    aria-pressed={selected}
                  >
                    <span className="block font-bold">{slotTimeRange(slot)}</span>
                    <span className={selected ? "mt-1 block text-white/80" : "mt-1 block opacity-75"}>{slotLabel(slot)}</span>
                  </button>
                );
              })}
            </div>
          </section>
        ))}
      </div>

      <div className="grid gap-3 rounded-lg border bg-slate-50 p-4 lg:grid-cols-[minmax(0,1fr)_minmax(220px,320px)_auto] lg:items-center">
        <div className="text-sm text-slate-600">
          {selectedSummary ? (
            <span>
              已选 {selectedSummary.ranges.length} 段 / {selectedSlots.length} 个周期，共 {formatHours(selectedSummary.durationHours)}：
              {selectedSummary.ranges.map((range) => formatSelectedRange(range.startTime, range.endTime)).join("；")}
            </span>
          ) : (
            <span>未选择时段</span>
          )}
        </div>
        <input className="h-11 w-full rounded-md border bg-white px-3 text-sm" name="purpose" placeholder="实验用途" required />
        <Button className="w-full lg:w-auto" disabled={pending || !selectedSummary} type="submit">
          {pending ? "提交中..." : "提交预约"}
        </Button>
      </div>

      {message ? <p className="rounded-md bg-slate-100 p-3 text-sm text-slate-700">{message}</p> : null}
    </form>
  );
}

function slotKey(slot: Slot) {
  return `${slot.startTime}-${slot.endTime}`;
}

function compareSlots(a: Slot, b: Slot) {
  return new Date(a.startTime).getTime() - new Date(b.startTime).getTime();
}

function selectedDurationHours(slots: Slot[]) {
  return slots.reduce((sum, slot) => sum + (new Date(slot.endTime).getTime() - new Date(slot.startTime).getTime()) / 3600000, 0);
}

function slotSelectionSummary(slots: Slot[]) {
  const ordered = [...slots].sort(compareSlots);
  const ranges: SlotRange[] = [];
  for (const slot of ordered) {
    const previous = ranges[ranges.length - 1];
    if (previous && new Date(previous.endTime).getTime() === new Date(slot.startTime).getTime()) {
      previous.endTime = slot.endTime;
    } else {
      ranges.push({ startTime: slot.startTime, endTime: slot.endTime });
    }
  }
  return {
    ranges,
    durationHours: selectedDurationHours(ordered),
  };
}

function formatHours(value: number) {
  return Number.isInteger(value) ? `${value} 小时` : `${value.toFixed(1)} 小时`;
}

function formatSelectedRange(startTime: string, endTime: string) {
  const startDate = new Date(startTime).toLocaleDateString("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    timeZone: "Asia/Shanghai",
  });
  const endDate = new Date(endTime).toLocaleDateString("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    timeZone: "Asia/Shanghai",
  });
  if (startDate === endDate) {
    return `${startDate} ${formatSlotTimeRange(startTime, endTime)}`;
  }
  const formatter = new Intl.DateTimeFormat("zh-CN", {
    hour: "2-digit",
    minute: "2-digit",
    timeZone: "Asia/Shanghai",
  });
  return `${startDate} ${formatter.format(new Date(startTime))} - ${endDate} ${formatter.format(new Date(endTime))}`;
}

function slotTimeRange(slot: Slot) {
  return formatSlotTimeRange(slot.startTime, slot.endTime);
}

function formatSlotTimeRange(startTime: string, endTime: string) {
  const formatter = new Intl.DateTimeFormat("zh-CN", {
    hour: "2-digit",
    minute: "2-digit",
    timeZone: "Asia/Shanghai",
  });
  return `${formatter.format(new Date(startTime))}-${formatter.format(new Date(endTime))}`;
}

function slotClassName(slot: SelectableSlot) {
  if (slot.disabled) {
    return "cursor-not-allowed border-slate-200 bg-slate-100 text-slate-400";
  }
  return "border-emerald-200 bg-emerald-50 text-emerald-800 hover:border-primary hover:bg-primary/10";
}

function slotLabel(slot: SelectableSlot) {
  if (slot.status === "available" && slot.disabled) {
    return statusLabels.unavailable;
  }
  return statusLabels[slot.status] ?? slot.status;
}
