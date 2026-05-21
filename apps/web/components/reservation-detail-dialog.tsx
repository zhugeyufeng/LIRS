import { Eye } from "lucide-react";
import type { ReactNode } from "react";
import { AdminDialog } from "@/components/admin-dialog";
import { Button } from "@/components/ui/button";
import type { Reservation } from "@/lib/api";
import { formatDateTime, formatDateTimeRange, formatDurationHours } from "@/lib/datetime";
import { reservationStatusLabel } from "@/lib/status-labels";

export function ReservationDetailDialog({
  actions,
  description = "查看预约单详情、用途说明和处理状态。",
  reservation,
  triggerLabel = "详情",
}: {
  actions?: ReactNode;
  description?: string;
  reservation: Reservation;
  triggerLabel?: string;
}) {
  return (
    <AdminDialog
      description={description}
      maxWidth="max-w-5xl"
      title={`预约详情：${reservation.instrumentName}`}
      trigger={
        <Button className="w-full sm:w-auto" size="sm" variant="outline">
          <Eye className="h-4 w-4" aria-hidden="true" />
          {triggerLabel}
        </Button>
      }
    >
      <div className="space-y-6">
        <header className="flex flex-col gap-3 border-b pb-4 sm:flex-row sm:items-start sm:justify-between">
          <div className="min-w-0">
            <p className="text-xs font-bold uppercase tracking-widest text-slate-400">申请单 #{reservation.id.slice(0, 8).toUpperCase()}</p>
            <h3 className="mt-2 break-words text-lg font-bold text-slate-900">{reservation.instrumentName}</h3>
            <p className="mt-1 break-words text-sm text-slate-500">
              {reservation.userName} / {reservation.groupName || "部门直属"}
            </p>
          </div>
          <StatusBadge status={reservation.status} />
        </header>

        <section>
          <h4 className="mb-3 text-xs font-bold uppercase tracking-widest text-slate-400">基础信息</h4>
          <div className="grid gap-4 md:grid-cols-2">
            <Info label="预约仪器" value={reservation.instrumentName} />
            <Info label="申请人" value={reservation.userName} />
            <Info label="团队" value={reservation.groupName || "部门直属"} />
            <Info label="费用" value={`¥${(reservation.fee ?? 0).toFixed(2)}`} />
          </div>
        </section>

        <section>
          <h4 className="mb-3 text-xs font-bold uppercase tracking-widest text-slate-400">时间信息</h4>
          <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
            <Info label="预约时间段" value={formatDateTimeRange(reservation.startTime, reservation.endTime)} />
            <Info label="开始时间" value={formatDateTime(reservation.startTime)} />
            <Info label="结束时间" value={formatDateTime(reservation.endTime)} />
            <Info label="预约时长" value={formatDurationHours(reservation.startTime, reservation.endTime)} />
          </div>
        </section>

        <section>
          <h4 className="mb-3 text-xs font-bold uppercase tracking-widest text-slate-400">用途说明</h4>
          <div className="rounded-lg border bg-slate-50/40 p-4 text-sm leading-7 text-slate-700">{reservation.purpose}</div>
        </section>

        {actions ? (
          <section>
            <h4 className="mb-3 text-xs font-bold uppercase tracking-widest text-slate-400">处理操作</h4>
            <div className="rounded-lg border bg-slate-50 p-4">
              <div className="flex flex-col gap-3 sm:flex-row sm:flex-wrap sm:items-center">{actions}</div>
            </div>
          </section>
        ) : null}
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
  return <span className={`inline-flex rounded-full px-2 py-1 text-[10px] font-bold ${statusClass(status)}`}>{reservationStatusLabel(status)}</span>;
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
