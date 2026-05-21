import Link from "next/link";
import { AlertTriangle, FileText, Search } from "lucide-react";
import { redirect } from "next/navigation";
import { ApprovalSectionTabs } from "@/components/approval-section-tabs";
import { AdminShell } from "@/components/admin-shell";
import { ReservationDetailDialog } from "@/components/reservation-detail-dialog";
import { ReservationActions } from "@/components/reservation-actions";
import { api } from "@/lib/api";
import { formatDateTimeRange, formatDurationHours } from "@/lib/datetime";
import { isTenantAdminRole, roleLabel } from "@/lib/permissions";
import { reservationStatusLabel } from "@/lib/status-labels";

export default async function ApprovalsPage({
  searchParams,
}: {
  searchParams?: Promise<{ q?: string }>;
}) {
  const params = (await searchParams) ?? {};
  const [currentUser, reservations] = await Promise.all([api.me(), api.reservations()]);
  if (currentUser.role !== "group_leader" && !isTenantAdminRole(currentUser.role)) {
    redirect("/dashboard");
  }
  const query = (params.q ?? "").trim().toLowerCase();
  const visibleReservations = reservations.filter((item) => item.status === "pending").filter((item) => {
    if (query === "") {
      return true;
    }
    return [item.instrumentName, item.userName, item.purpose, item.groupName, item.status].some((value) => value.toLowerCase().includes(query));
  });
  const isAdmin = isTenantAdminRole(currentUser.role);

  return (
    <AdminShell active="approvals" allowedSections={isAdmin ? undefined : ["approvals"]} currentUser={currentUser} title="审批中心" description="待处理申请以列表展示，点击详情弹窗查看完整信息并完成审批。">
      <ApprovalSectionTabs active="pending" />

      <div className="mb-6 flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
        <div className="min-w-0">
          <p className="text-xs font-bold uppercase tracking-widest text-primary">审批中心</p>
          <h1 className="mt-2 text-2xl font-bold tracking-tight sm:text-3xl">待处理申请</h1>
          <p className="mt-2 max-w-3xl text-sm leading-6 text-muted-foreground">按列表查看待审批预约，弹窗内可直接查看详情、用途说明和审批操作。</p>
        </div>
        <form action="/approvals" className="relative w-full lg:max-w-md">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" aria-hidden="true" />
          <input
            className="h-10 w-full rounded-lg border bg-white pl-10 pr-4 text-sm outline-none focus:ring-2 focus:ring-primary/20"
            defaultValue={params.q ?? ""}
            name="q"
            placeholder="搜索申请人、仪器、团队..."
            type="search"
          />
        </form>
      </div>

      <section className="space-y-4">
        <div className="flex items-center justify-between gap-3">
          <h2 className="text-lg font-bold">待处理申请 ({visibleReservations.length})</h2>
          <p className="text-sm text-slate-500">只展示当前待审批的预约单。</p>
        </div>

        {visibleReservations.length > 0 ? (
          <div className="space-y-3">
            {visibleReservations.map((item) => (
              <article className="rounded-xl border bg-white p-4 shadow-sm" key={item.id}>
                <div className="flex flex-col justify-between gap-3 sm:flex-row sm:items-start">
                  <div className="min-w-0">
                    <p className="text-xs font-bold uppercase tracking-widest text-slate-400">申请单 #{item.id.slice(0, 8).toUpperCase()}</p>
                    <h3 className="mt-2 break-words text-base font-bold text-slate-900">{item.instrumentName}</h3>
                    <p className="mt-1 break-words text-sm text-slate-500">
                      {item.userName} / {item.groupName}
                    </p>
                  </div>
                  <StatusBadge status={item.status} />
                </div>
                <div className="mt-4 grid gap-3 text-xs text-slate-500 sm:grid-cols-2 xl:grid-cols-4">
                  <Info label="预约时间段" value={formatDateTimeRange(item.startTime, item.endTime)} />
                  <Info label="预约时长" value={formatDurationHours(item.startTime, item.endTime)} />
                  <Info label="费用" value={`¥${(item.fee ?? 0).toFixed(2)}`} />
                  <Info label="状态" value={statusLabel(item.status)} />
                </div>
                <p className="mt-4 break-words rounded-lg bg-slate-50/60 p-3 text-sm leading-6 text-slate-600">{item.purpose}</p>
                <div className="mt-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                  <p className="text-xs text-slate-500">预约编号：RES-{item.id.slice(0, 8).toUpperCase()}</p>
                  <div className="flex flex-col gap-2 sm:flex-row">
                    <Link className="inline-flex h-9 items-center justify-center rounded-md border px-3 text-sm font-medium text-slate-700 hover:bg-slate-50" href={`/approvals/${item.id}`}>
                      详情页
                    </Link>
                    <ReservationDetailDialog
                      actions={<ReservationActions canCancel={false} canCheckIn={false} canCheckOut={false} id={item.id} status={item.status} />}
                      description="查看预约详情、用途说明与审批操作。"
                      reservation={item}
                      triggerLabel="弹窗"
                    />
                  </div>
                </div>
              </article>
            ))}
          </div>
        ) : (
          <div className="rounded-lg border bg-white p-8 text-center text-sm text-slate-500">
            <FileText className="mx-auto h-5 w-5 text-slate-400" aria-hidden="true" />
            <p className="mt-3">暂无待处理申请。</p>
          </div>
        )}
      </section>

      <div className="mt-6 flex items-center gap-2 text-xs text-slate-500">
        <AlertTriangle className="h-3.5 w-3.5" aria-hidden="true" />
        超过 24 小时仍未审批的预约会被系统自动取消并通知申请人。
      </div>
      <div className="mt-2 text-xs text-slate-500">当前角色：{roleLabel(currentUser.role)}</div>
    </AdminShell>
  );
}

function Info({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border bg-slate-50/30 p-4">
      <p className="text-[10px] font-bold uppercase text-slate-400">{label}</p>
      <p className="mt-2 break-words text-sm font-bold text-slate-900">{value}</p>
    </div>
  );
}

function StatusBadge({ status }: { status: string }) {
  return <span className={`inline-flex w-fit rounded-full px-2 py-1 text-[10px] font-bold ${statusClass(status)}`}>{reservationStatusLabel(status)}</span>;
}

function statusLabel(status: string) {
  return reservationStatusLabel(status);
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
