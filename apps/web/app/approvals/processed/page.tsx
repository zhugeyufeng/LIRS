import { Eye, FileText, Search } from "lucide-react";
import { redirect } from "next/navigation";
import { ApprovalSectionTabs } from "@/components/approval-section-tabs";
import { AdminDialog } from "@/components/admin-dialog";
import { AdminShell } from "@/components/admin-shell";
import { Button } from "@/components/ui/button";
import { api } from "@/lib/api";
import { formatDateTime, formatDateTimeRange, formatDurationHours } from "@/lib/datetime";
import { isTenantAdminRole } from "@/lib/permissions";

export default async function ProcessedApprovalsPage({
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
  const visibleReservations = reservations
    .filter((item) => item.status !== "pending")
    .filter((item) => {
      if (query === "") {
        return true;
      }
      return [item.instrumentName, item.userName, item.purpose, item.groupName, item.status].some((value) => value.toLowerCase().includes(query));
    })
    .sort((a, b) => new Date(b.startTime).getTime() - new Date(a.startTime).getTime());

  return (
    <AdminShell active="approvals" allowedSections={isTenantAdminRole(currentUser.role) ? undefined : ["approvals"]} currentUser={currentUser} title="已处理申请" description="查看已处理的预约申请记录，点击条目打开详情弹窗。">
      <ApprovalSectionTabs active="processed" />
      <div className="mb-6 flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
        <div className="min-w-0">
          <p className="text-xs font-bold uppercase tracking-widest text-primary">审批中心</p>
          <h1 className="mt-2 text-2xl font-bold tracking-tight sm:text-3xl">已处理申请</h1>
          <p className="mt-2 max-w-3xl text-sm leading-6 text-muted-foreground">
            按列表查看已处理的申请记录，适合回看审批结果、履约状态和历史用途说明。
          </p>
        </div>
        <form action="/approvals/processed" className="relative w-full lg:max-w-md">
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
        <div className="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
          <h2 className="text-lg font-bold">处理记录 ({visibleReservations.length})</h2>
          <p className="text-sm text-slate-500">仅展示已处理的历史申请。</p>
        </div>

        {visibleReservations.length > 0 ? (
          <>
            <div className="space-y-3 xl:hidden">
              {visibleReservations.map((item) => (
                <ProcessedReservationCard item={item} key={item.id} />
              ))}
            </div>
            <div className="hidden overflow-x-auto rounded-lg border bg-white xl:block">
              <table className="w-full text-left text-sm">
                <thead className="bg-slate-50 text-slate-500">
                  <tr>
                    <th className="px-4 py-3">申请单</th>
                    <th className="px-4 py-3">仪器</th>
                    <th className="px-4 py-3">申请人</th>
                    <th className="px-4 py-3">时间</th>
                    <th className="px-4 py-3">状态</th>
                    <th className="px-4 py-3 text-right">操作</th>
                  </tr>
                </thead>
                <tbody className="divide-y">
                  {visibleReservations.map((item) => (
                    <tr className="hover:bg-slate-50" key={item.id}>
                      <td className="px-4 py-3">
                        <p className="font-medium text-slate-900">#{item.id.slice(0, 8).toUpperCase()}</p>
                        <p className="mt-1 text-xs text-slate-500">{formatDateTime(item.startTime)}</p>
                      </td>
                      <td className="px-4 py-3 font-medium text-slate-900">{item.instrumentName}</td>
                      <td className="px-4 py-3">
                        <p className="font-medium text-slate-900">{item.userName}</p>
                        <p className="mt-1 text-xs text-slate-500">{item.groupName}</p>
                      </td>
                      <td className="px-4 py-3 text-xs text-slate-500">
                        <div>{formatDateTimeRange(item.startTime, item.endTime)}</div>
                        <div className="mt-1">共 {formatDurationHours(item.startTime, item.endTime)}</div>
                      </td>
                      <td className="px-4 py-3">
                        <StatusBadge status={item.status} />
                      </td>
                      <td className="px-4 py-3 text-right">
                        <ReservationDetailDialog item={item} />
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </>
        ) : (
          <div className="rounded-lg border bg-white p-8 text-center text-sm text-slate-500">
            <FileText className="mx-auto h-5 w-5 text-slate-400" aria-hidden="true" />
            <p className="mt-3">暂无已处理申请。</p>
          </div>
        )}
      </section>
    </AdminShell>
  );
}

function ProcessedReservationCard({ item }: { item: { id: string; instrumentName: string; userName: string; groupName: string; purpose: string; startTime: string; endTime: string; status: string; fee: number } }) {
  return (
    <div className="rounded-xl border bg-white p-4 shadow-sm">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="text-xs font-bold uppercase tracking-widest text-slate-400">申请单 #{item.id.slice(0, 8).toUpperCase()}</p>
          <h3 className="mt-2 break-words text-base font-bold text-slate-900">{item.instrumentName}</h3>
          <p className="mt-1 break-words text-sm text-slate-500">{item.userName} / {item.groupName}</p>
        </div>
        <StatusBadge status={item.status} />
      </div>
      <div className="mt-4 grid gap-3 text-xs text-slate-500 sm:grid-cols-2">
        <Info label="预约时间段" value={formatDateTimeRange(item.startTime, item.endTime)} />
        <Info label="预约时长" value={formatDurationHours(item.startTime, item.endTime)} />
        <Info label="状态" value={statusLabel(item.status)} />
        <Info label="费用" value={`¥${(item.fee ?? 0).toFixed(2)}`} />
      </div>
      <div className="mt-4 rounded-lg border bg-slate-50/40 p-3 text-sm leading-6 text-slate-600">
        <p className="text-xs font-bold uppercase tracking-widest text-slate-400">用途说明</p>
        <p className="mt-2 break-words">{item.purpose}</p>
      </div>
      <div className="mt-4 flex justify-end">
        <ReservationDetailDialog item={item} />
      </div>
    </div>
  );
}

function ReservationDetailDialog({
  item,
}: {
  item: { id: string; instrumentName: string; userName: string; groupName: string; purpose: string; startTime: string; endTime: string; status: string; fee: number };
}) {
  return (
    <AdminDialog
      description="查看该预约申请的基础信息、用途说明和处理状态。"
      title={`预约详情：${item.instrumentName}`}
      trigger={
        <Button className="h-9 w-full sm:w-auto" size="sm" variant="outline">
          <Eye className="h-4 w-4" aria-hidden="true" />
          详情
        </Button>
      }
    >
      <div className="space-y-6">
        <div className="flex flex-col gap-3 border-b pb-4 sm:flex-row sm:items-start sm:justify-between">
          <div className="min-w-0">
            <p className="text-xs font-bold uppercase tracking-widest text-slate-400">申请单 #{item.id.slice(0, 8).toUpperCase()}</p>
            <h3 className="mt-2 break-words text-lg font-bold text-slate-900">{item.instrumentName}</h3>
            <p className="mt-1 text-sm text-slate-500">{item.userName} / {item.groupName}</p>
          </div>
          <div className="shrink-0">
            <StatusBadge status={item.status} />
          </div>
        </div>

        <section>
          <h4 className="mb-3 text-xs font-bold uppercase tracking-widest text-slate-400">基础信息</h4>
          <div className="grid gap-4 md:grid-cols-2">
            <Info label="预约仪器" value={item.instrumentName} />
            <Info label="申请人" value={item.userName} />
            <Info label="团队" value={item.groupName || "部门直属"} />
            <Info label="费用" value={`¥${(item.fee ?? 0).toFixed(2)}`} />
          </div>
        </section>

        <section>
          <h4 className="mb-3 text-xs font-bold uppercase tracking-widest text-slate-400">时间信息</h4>
          <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
            <Info label="预约时间段" value={formatDateTimeRange(item.startTime, item.endTime)} />
            <Info label="开始时间" value={formatDateTime(item.startTime)} />
            <Info label="结束时间" value={formatDateTime(item.endTime)} />
            <Info label="预约时长" value={formatDurationHours(item.startTime, item.endTime)} />
          </div>
        </section>

        <section>
          <h4 className="mb-3 text-xs font-bold uppercase tracking-widest text-slate-400">用途说明</h4>
          <div className="rounded-lg border bg-slate-50/40 p-4 text-sm leading-7 text-slate-700">{item.purpose}</div>
        </section>
      </div>
    </AdminDialog>
  );
}

function Info({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border bg-slate-50/30 p-4">
      <p className="text-[10px] font-bold uppercase tracking-widest text-slate-400">{label}</p>
      <p className="mt-2 break-words text-sm font-bold text-slate-900">{value}</p>
    </div>
  );
}

function StatusBadge({ status }: { status: string }) {
  return <span className={`inline-flex rounded-full px-2 py-1 text-[10px] font-bold ${statusClass(status)}`}>{statusLabel(status)}</span>;
}

function statusLabel(status: string) {
  const labels: Record<string, string> = {
    pending: "待处理",
    approved: "已通过",
    rejected: "已驳回",
    in_use: "使用中",
    completed: "已完成",
    cancelled: "已取消",
  };
  return labels[status] ?? status;
}

function statusClass(status: string) {
  const classes: Record<string, string> = {
    approved: "bg-blue-50 text-blue-700",
    rejected: "bg-rose-50 text-rose-700",
    in_use: "bg-amber-50 text-amber-700",
    completed: "bg-emerald-50 text-emerald-700",
    cancelled: "bg-slate-100 text-slate-600",
  };
  return classes[status] ?? "bg-slate-100 text-slate-600";
}
