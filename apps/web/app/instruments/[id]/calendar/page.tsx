import Link from "next/link";
import { ArrowLeft, CalendarDays, Clock3 } from "lucide-react";
import { AppShell } from "@/components/app-shell";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api, type Slot } from "@/lib/api";
import { formatServiceWindow } from "@/lib/instrument-rules";

type GroupedSlots = Record<string, Slot[]>;

export default async function InstrumentCalendarPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const instrument = await api.instrument(id);
  const days = Math.max(1, Math.min(instrument.bookingWindowDays, 30));
  const slots = await api.slots(id, days);
  const grouped = groupSlots(slots);
  const availableCount = slots.filter((slot) => slot.status === "available").length;

  return (
    <AppShell>
      <Link className="mb-5 inline-flex items-center gap-2 text-sm text-slate-600 hover:text-primary" href={`/instruments/${instrument.id}`} prefetch={false}>
        <ArrowLeft className="h-4 w-4" />
        返回仪器详情
      </Link>

      <div className="mb-6 flex flex-col justify-between gap-4 lg:flex-row lg:items-end">
        <div className="min-w-0">
          <p className="text-xs font-bold uppercase tracking-widest text-primary">仪器日历</p>
          <h1 className="mt-2 text-2xl font-bold text-slate-900 sm:text-3xl">{instrument.name}</h1>
          <p className="mt-2 max-w-3xl text-sm leading-6 text-muted-foreground">
            展示未来 {days} 天的可预约时段、维护占用和已锁定时间。开放时间 {formatServiceWindow(instrument)}，每 {instrument.bookingIntervalHours} 小时一个周期。
          </p>
        </div>
        <Button asChild className="w-full sm:w-auto">
          <Link href={`/instruments/${instrument.id}/reserve`} prefetch={false}>提交预约</Link>
        </Button>
      </div>

      <div className="mb-6 grid gap-4 sm:grid-cols-3">
        <Metric label="可预约周期" value={availableCount} />
        <Metric label="全部周期" value={slots.length} />
        <Metric label="可预约天数" value={days} />
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <CalendarDays className="h-5 w-5 text-primary" />
            时间块日历
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {Object.entries(grouped).map(([day, daySlots]) => (
            <section className="rounded-lg border bg-white p-3" key={day}>
              <div className="mb-3 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
                <h2 className="font-bold text-slate-900">{day}</h2>
                <p className="text-xs text-slate-500">
                  {daySlots.filter((slot) => slot.status === "available").length} 个可预约 / {daySlots.length} 个周期
                </p>
              </div>
              <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 md:grid-cols-4 xl:grid-cols-6">
                {daySlots.map((slot) => (
                  <SlotCard key={`${slot.startTime}-${slot.endTime}`} slot={slot} />
                ))}
              </div>
            </section>
          ))}
          {slots.length === 0 ? <p className="rounded-lg border p-4 text-sm text-slate-500">暂无可展示时间块。</p> : null}
        </CardContent>
      </Card>
    </AppShell>
  );
}

function SlotCard({ slot }: { slot: Slot }) {
  return (
    <div className={`min-h-16 rounded-md border px-2 py-2 text-xs ${slotClass(slot.status)}`}>
      <div className="flex items-center gap-1 font-bold">
        <Clock3 className="h-3.5 w-3.5" aria-hidden="true" />
        {formatTime(slot.startTime)}-{formatTime(slot.endTime)}
      </div>
      <p className="mt-2">{slotLabel(slot.status)}</p>
      {slot.reason ? <p className="mt-1 break-words opacity-75">{slot.reason}</p> : null}
    </div>
  );
}

function Metric({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-lg border bg-white p-4">
      <p className="text-sm text-slate-500">{label}</p>
      <p className="mt-2 text-2xl font-bold">{value}</p>
    </div>
  );
}

function groupSlots(slots: Slot[]): GroupedSlots {
  return slots.reduce<GroupedSlots>((groups, slot) => {
    const day = new Date(slot.startTime).toLocaleDateString("zh-CN", {
      weekday: "short",
      month: "2-digit",
      day: "2-digit",
      timeZone: "Asia/Shanghai",
    });
    groups[day] ??= [];
    groups[day].push(slot);
    return groups;
  }, {});
}

function slotClass(status: string) {
  const classes: Record<string, string> = {
    available: "border-emerald-200 bg-emerald-50 text-emerald-800",
    occupied: "border-amber-200 bg-amber-50 text-amber-800",
    maintenance: "border-red-200 bg-red-50 text-red-800",
    disabled: "border-slate-200 bg-slate-100 text-slate-500",
  };
  return classes[status] ?? "border-slate-200 bg-slate-50 text-slate-600";
}

function slotLabel(status: string) {
  const labels: Record<string, string> = {
    available: "可预约",
    occupied: "已占用",
    maintenance: "维护中",
    disabled: "已停用",
    unavailable: "不可预约",
  };
  return labels[status] ?? status;
}

function formatTime(value: string) {
  return new Date(value).toLocaleTimeString("zh-CN", { hour: "2-digit", minute: "2-digit", timeZone: "Asia/Shanghai" });
}
