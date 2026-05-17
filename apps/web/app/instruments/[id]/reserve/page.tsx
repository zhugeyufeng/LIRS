import Link from "next/link";
import { ArrowLeft, BadgeDollarSign, ClipboardList, MapPin, type LucideIcon } from "lucide-react";
import { AppShell } from "@/components/app-shell";
import { ReservationSlotForm } from "@/components/reservation-slot-form";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";
import { formatServiceWindow } from "@/lib/instrument-rules";

export default async function InstrumentReservePage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const instrument = await api.instrument(id);
  const slotDays = Math.max(1, Math.min(instrument.bookingWindowDays, 30));
  const slots = await api.slots(id, slotDays);
  const bookableAfter = new Date(Date.now() + Math.max(instrument.minAdvanceHours, 0) * 60 * 60 * 1000).toISOString();

  return (
    <AppShell>
      <Link className="mb-5 inline-flex items-center gap-2 text-sm text-slate-600 hover:text-primary" href={`/instruments/${instrument.id}`} prefetch={false}>
        <ArrowLeft className="h-4 w-4" />
        返回仪器详情
      </Link>
      <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_340px]">
        <section className="min-w-0">
          <div className="mb-5">
            <h1 className="text-2xl font-bold tracking-tight sm:text-3xl">预约 {instrument.name}</h1>
            <p className="mt-2 text-sm text-muted-foreground">按管理员设置的预约周期提交当前仪器预约。</p>
          </div>
          <ReservationSlotForm instrument={instrument} slots={slots} bookableAfter={bookableAfter} />
        </section>

        <aside className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>仪器信息</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3 text-sm">
              <Info icon={MapPin} label="位置" value={`${instrument.department} / ${instrument.location}`} />
              <Info icon={BadgeDollarSign} label="费率" value={`¥${instrument.hourlyRate}/小时`} />
              <Info icon={ClipboardList} label="归属团队" value={instrument.groupName || `${instrument.department}（部门直属）`} />
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle>预约规则</CardTitle>
            </CardHeader>
            <CardContent className="space-y-2 text-sm leading-6 text-slate-700">
              <p>{instrument.bookingRule}</p>
              <div className="grid gap-2 text-xs text-slate-600">
                <span>预约周期：{instrument.bookingIntervalHours} 小时</span>
                <span>最长预约：{instrument.maxBookingHours} 小时</span>
                <span>提前预约：{instrument.minAdvanceHours} 小时</span>
                <span>可预约：未来 {instrument.bookingWindowDays} 天</span>
                <span>开放时间：{formatServiceWindow(instrument)}</span>
              </div>
            </CardContent>
          </Card>
        </aside>
      </div>
    </AppShell>
  );
}

function Info({ icon: Icon, label, value }: { icon: LucideIcon; label: string; value: string }) {
  return (
    <div className="rounded-lg border bg-slate-50 p-3">
      <div className="mb-1 flex items-center gap-2 text-xs font-bold uppercase text-slate-400">
        <Icon className="h-4 w-4" aria-hidden="true" />
        {label}
      </div>
      <p className="font-bold text-slate-800">{value}</p>
    </div>
  );
}
