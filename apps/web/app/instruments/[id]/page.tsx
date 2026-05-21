import Link from "next/link";
import { ArrowLeft, BadgeDollarSign, CalendarClock, CalendarPlus, CheckCircle2, ClipboardList, MapPin, Settings2, Wrench, type LucideIcon } from "lucide-react";
import { AppShell } from "@/components/app-shell";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";
import { formatServiceWindow } from "@/lib/instrument-rules";
import { instrumentStatusLabel } from "@/lib/status-labels";

export default async function InstrumentDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const instrument = await api.instrument(id);
  const previewDays = Math.max(1, Math.min(instrument.bookingWindowDays, 14));

  return (
    <AppShell>
      <Link className="mb-5 inline-flex items-center gap-2 text-sm text-slate-600 hover:text-primary" href="/instruments" prefetch={false}>
        <ArrowLeft className="h-4 w-4" />
        返回仪器目录
      </Link>
      <div className="grid gap-8 xl:grid-cols-[minmax(0,1fr)_380px]">
        <div className="space-y-6">
          <section className="overflow-hidden rounded-xl border bg-white shadow-sm">
            <div className="flex aspect-[16/7] items-center justify-center bg-slate-100">
              <Settings2 className="h-24 w-24 text-primary/30" aria-hidden="true" />
            </div>
            <div className="p-6">
              <div className="flex flex-col justify-between gap-4 md:flex-row md:items-start">
                <div>
                  <h1 className="text-2xl font-bold">{instrument.name}</h1>
                  <p className="mt-2 text-sm text-muted-foreground">
                    {instrument.brand} {instrument.model} / 资产编号 {instrument.assetCode || "未登记"}
                  </p>
                </div>
                <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
                  <span className="w-fit rounded bg-primary px-3 py-1 text-xs font-bold text-white">
                    {instrumentStatusLabel(instrument.status)}
                  </span>
                  <Button asChild variant="outline">
                    <Link href={`/instruments/${instrument.id}/calendar`} prefetch={false}>
                      <CalendarClock className="h-4 w-4" aria-hidden="true" />
                      查看日历
                    </Link>
                  </Button>
                  <Button asChild>
                    <Link href={`/instruments/${instrument.id}/reserve`} prefetch={false}>
                      <CalendarPlus className="h-4 w-4" aria-hidden="true" />
                      预约仪器
                    </Link>
                  </Button>
                </div>
              </div>
              <p className="mt-5 leading-7 text-slate-700">{instrument.description}</p>
              <div className="mt-6 grid gap-4 md:grid-cols-3">
                <Info icon={MapPin} label="位置" value={`${instrument.department} / ${instrument.location}`} />
                <Info icon={BadgeDollarSign} label="费率" value={`¥${instrument.hourlyRate}/小时`} />
                <Info icon={ClipboardList} label="归属团队" value={instrument.groupName || `${instrument.department}（部门直属）`} />
              </div>
            </div>
          </section>

          <Card>
            <CardHeader>
              <CardTitle>技术参数与预约规则</CardTitle>
            </CardHeader>
            <CardContent className="grid gap-4 md:grid-cols-2">
              <div className="rounded-lg border bg-slate-50 p-4">
                <p className="mb-2 text-xs font-bold uppercase text-slate-400">技术参数</p>
                <p className="text-sm leading-7">{instrument.technicalSpecs || "暂无参数"}</p>
              </div>
              <div className="rounded-lg border bg-slate-50 p-4">
                <p className="mb-2 text-xs font-bold uppercase text-slate-400">预约规则</p>
                <p className="text-sm leading-7">{instrument.bookingRule}</p>
                <div className="mt-3 grid gap-2 text-xs text-slate-600 sm:grid-cols-2">
                  <span>最长 {instrument.maxBookingHours} 小时</span>
                  <span>提前 {instrument.minAdvanceHours} 小时</span>
                  <span>取消截止 {instrument.cancelCutoffHours} 小时</span>
                  <span>签到窗口 {instrument.checkinWindowMinutes} 分钟</span>
                  <span>可预约未来 {instrument.bookingWindowDays} 天</span>
                  <span>预约时段 {instrument.bookingIntervalHours} 小时</span>
                  <span>开放时间 {formatServiceWindow(instrument)}</span>
                </div>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <CalendarClock className="h-5 w-5 text-primary" />
                未来时段
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <p className="text-sm leading-6 text-slate-600">
                当前仪器开放 {formatServiceWindow(instrument)}，每 {instrument.bookingIntervalHours} 小时一个预约周期；完整可预约、维护和已锁定时段在日历页按天加载。
              </p>
              <div className="grid gap-3 sm:grid-cols-3">
                <Metric label="预览天数" value={`${previewDays} 天`} />
                <Metric label="预约周期" value={`${instrument.bookingIntervalHours} 小时`} />
                <Metric label="提前预约" value={`${instrument.minAdvanceHours} 小时`} />
              </div>
              <div className="flex flex-col gap-3 sm:flex-row">
                <Button asChild className="w-full sm:w-auto" variant="outline">
                  <Link href={`/instruments/${instrument.id}/calendar`} prefetch={false}>
                    <CalendarClock className="h-4 w-4" aria-hidden="true" />
                    查看时段日历
                  </Link>
                </Button>
                <Button asChild className="w-full sm:w-auto">
                  <Link href={`/instruments/${instrument.id}/reserve`} prefetch={false}>
                    <CalendarPlus className="h-4 w-4" aria-hidden="true" />
                    选择时段预约
                  </Link>
                </Button>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Wrench className="h-5 w-5 text-primary" />
                维护记录
              </CardTitle>
            </CardHeader>
            <CardContent className="text-sm leading-7 text-slate-700">
              {instrument.maintenanceSummary || "暂无维护记录"}
            </CardContent>
          </Card>
        </div>

        <aside className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>预约入口</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <p className="text-sm leading-6 text-slate-600">
                每 {instrument.bookingIntervalHours} 小时一个预约周期，开放 {formatServiceWindow(instrument)}，进入预约页后选择可用时间块提交。
              </p>
              <Button asChild className="w-full" variant="outline">
                <Link href={`/instruments/${instrument.id}/calendar`} prefetch={false}>
                  <CalendarClock className="h-4 w-4" aria-hidden="true" />
                  查看日历
                </Link>
              </Button>
              <Button asChild className="w-full">
                <Link href={`/instruments/${instrument.id}/reserve`} prefetch={false}>
                  <CalendarPlus className="h-4 w-4" aria-hidden="true" />
                  去预约
                </Link>
              </Button>
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle>使用统计</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3 text-sm">
              <div className="flex items-center justify-between rounded-lg border p-3">
                <span>已完成使用</span>
                <span className="font-bold">{instrument.usageCount} 次</span>
              </div>
              <div className="flex items-center gap-2 rounded-lg border border-emerald-100 bg-emerald-50 p-3 text-emerald-700">
                <CheckCircle2 className="h-4 w-4" />
                审批中、已通过、使用中时段都会被锁定。
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
    <div className="rounded-lg border bg-slate-50 p-4">
      <div className="mb-2 flex items-center gap-2 text-xs font-bold uppercase text-slate-400">
        <Icon className="h-4 w-4" aria-hidden="true" />
        {label}
      </div>
      <p className="text-sm font-bold">{value}</p>
    </div>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border bg-slate-50 p-4">
      <p className="text-xs text-slate-500">{label}</p>
      <p className="mt-2 font-bold text-slate-900">{value}</p>
    </div>
  );
}
