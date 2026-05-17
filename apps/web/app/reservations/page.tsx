import Link from "next/link";
import { CalendarCheck2, ClipboardCheck, Search } from "lucide-react";
import { AppShell } from "@/components/app-shell";
import { ReservationActions } from "@/components/reservation-actions";
import { ReservationDetailDialog } from "@/components/reservation-detail-dialog";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api, type Reservation } from "@/lib/api";

type SearchParams = {
  q?: string;
  status?: string;
};

const statusLabels: Record<string, string> = {
  pending: "待审批",
  approved: "已通过",
  rejected: "已驳回",
  in_use: "使用中",
  completed: "已完成",
  cancelled: "已取消",
};

export default async function ReservationsPage({ searchParams }: { searchParams?: Promise<SearchParams> }) {
  const params = (await searchParams) ?? {};
  const [currentUser, reservations] = await Promise.all([api.me(), api.reservations()]);
  const query = (params.q ?? "").trim().toLowerCase();
  const status = params.status ?? "active";
  const visibleReservations = reservations
    .filter((item) => statusMatches(item, status))
    .filter((item) => {
      if (!query) {
        return true;
      }
      return [item.id, item.instrumentName, item.userName, item.groupName, item.purpose, item.status].some((value) => value.toLowerCase().includes(query));
    })
    .sort((a, b) => new Date(b.startTime).getTime() - new Date(a.startTime).getTime());

  return (
    <AppShell currentUser={currentUser}>
      <div className="mb-6 flex flex-col justify-between gap-4 lg:flex-row lg:items-end">
        <div className="min-w-0">
          <p className="text-xs font-bold uppercase tracking-widest text-primary">预约与使用中心</p>
          <h1 className="mt-2 text-2xl font-bold text-slate-900 sm:text-3xl">预约记录</h1>
          <p className="mt-2 max-w-3xl text-sm leading-6 text-muted-foreground">
            当前身份：{currentUser.name}。这里展示你有权限查看的预约，支持查看详情、签到、签退和取消。
          </p>
        </div>
        <Button asChild className="w-full sm:w-auto">
          <Link href="/instruments">
            <CalendarCheck2 className="h-4 w-4" aria-hidden="true" />
            去预约仪器
          </Link>
        </Button>
      </div>

      <div className="mb-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <Metric label="进行中" value={reservations.filter((item) => ["pending", "approved", "in_use"].includes(item.status)).length} />
        <Metric label="待审批" value={reservations.filter((item) => item.status === "pending").length} />
        <Metric label="使用中" value={reservations.filter((item) => item.status === "in_use").length} />
        <Metric label="已完成" value={reservations.filter((item) => item.status === "completed").length} />
      </div>

      <Card>
        <CardHeader className="gap-4">
          <div className="flex flex-col gap-3 xl:flex-row xl:items-center xl:justify-between">
            <CardTitle className="flex items-center gap-2">
              <ClipboardCheck className="h-5 w-5 text-primary" />
              预约列表
            </CardTitle>
            <form action="/reservations" className="grid w-full gap-2 sm:grid-cols-[minmax(0,1fr)_180px_auto] xl:max-w-3xl">
              <div className="relative min-w-0">
                <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" aria-hidden="true" />
                <input className="h-10 w-full rounded-md border bg-white pl-10 pr-3 text-sm" defaultValue={params.q ?? ""} name="q" placeholder="搜索仪器、用途、团队" />
              </div>
              <select className="h-10 min-w-0 rounded-md border bg-white px-3 text-sm" defaultValue={status} name="status">
                <option value="active">进行中</option>
                <option value="pending">待审批</option>
                <option value="approved">已通过</option>
                <option value="in_use">使用中</option>
                <option value="completed">已完成</option>
                <option value="cancelled">已取消/驳回</option>
                <option value="all">全部</option>
              </select>
              <button className="inline-flex h-10 items-center justify-center rounded-md bg-primary px-4 text-sm font-bold text-white" type="submit">
                筛选
              </button>
            </form>
          </div>
          <div className="flex flex-wrap gap-2">
            {[
              ["active", "进行中"],
              ["completed", "已完成"],
              ["cancelled", "已取消/驳回"],
              ["all", "全部"],
            ].map(([value, label]) => (
              <Link
                className={`inline-flex h-8 items-center rounded-md px-3 text-xs font-bold ${status === value ? "bg-primary text-white" : "bg-slate-100 text-slate-600 hover:bg-slate-200"}`}
                href={`/reservations?status=${value}`}
                key={value}
              >
                {label}
              </Link>
            ))}
          </div>
        </CardHeader>
        <CardContent>
          <div className="space-y-3">
            {visibleReservations.map((item) => (
              <ReservationRow item={item} key={item.id} />
            ))}
          </div>
          {visibleReservations.length === 0 ? <p className="rounded-lg border p-4 text-sm text-slate-500">当前筛选下暂无预约。</p> : null}
        </CardContent>
      </Card>
    </AppShell>
  );
}

function ReservationRow({ item }: { item: Reservation }) {
  const actions = ["pending", "approved", "in_use"].includes(item.status) ? <ReservationActions canReview={false} id={item.id} status={item.status} /> : undefined;
  return (
    <article className="rounded-lg border bg-white p-4">
      <div className="flex flex-col justify-between gap-3 md:flex-row md:items-start">
        <div className="min-w-0">
          <p className="text-xs font-bold uppercase tracking-widest text-slate-400">RES-{item.id.slice(0, 8).toUpperCase()}</p>
          <h2 className="mt-2 break-words text-base font-bold text-slate-900">{item.instrumentName}</h2>
          <p className="mt-1 break-words text-sm text-slate-500">
            {formatDate(item.startTime)} {formatTime(item.startTime)}-{formatTime(item.endTime)} / {item.groupName || "部门直属"}
          </p>
        </div>
        <span className={`w-fit shrink-0 rounded-full px-2 py-1 text-xs font-bold ${statusClass(item.status)}`}>{statusLabels[item.status] ?? item.status}</span>
      </div>
      <p className="mt-3 break-words rounded-md bg-slate-50 p-3 text-sm leading-6 text-slate-600">{item.purpose}</p>
      <div className="mt-4 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <p className="text-sm font-bold text-primary">预计/入账费用 ¥{(item.fee ?? 0).toFixed(2)}</p>
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
          <ReservationDetailDialog
            actions={actions}
            reservation={item}
          />
        </div>
      </div>
    </article>
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

function statusMatches(item: Reservation, status: string) {
  if (status === "all") {
    return true;
  }
  if (status === "active") {
    return ["pending", "approved", "in_use"].includes(item.status);
  }
  if (status === "cancelled") {
    return item.status === "cancelled" || item.status === "rejected";
  }
  return item.status === status;
}

function statusClass(status: string) {
  const classes: Record<string, string> = {
    pending: "bg-amber-50 text-amber-700",
    approved: "bg-blue-50 text-blue-700",
    rejected: "bg-rose-50 text-rose-700",
    in_use: "bg-amber-50 text-amber-700",
    completed: "bg-emerald-50 text-emerald-700",
    cancelled: "bg-slate-100 text-slate-600",
  };
  return classes[status] ?? "bg-slate-100 text-slate-600";
}

function formatDate(value: string) {
  return new Date(value).toLocaleDateString("zh-CN", { timeZone: "Asia/Shanghai" });
}

function formatTime(value: string) {
  return new Date(value).toLocaleTimeString("zh-CN", { hour: "2-digit", minute: "2-digit", timeZone: "Asia/Shanghai" });
}
