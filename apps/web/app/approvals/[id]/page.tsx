import Link from "next/link";
import { notFound, redirect } from "next/navigation";
import { AdminShell } from "@/components/admin-shell";
import { ReservationActions } from "@/components/reservation-actions";
import { ReservationDetailDialog } from "@/components/reservation-detail-dialog";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";
import { formatDateTimeRange, formatDurationHours } from "@/lib/datetime";
import { isTenantAdminRole } from "@/lib/permissions";

export default async function ApprovalDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const [currentUser, reservations] = await Promise.all([api.me(), api.reservations()]);
  if (currentUser.role !== "group_leader" && !isTenantAdminRole(currentUser.role)) {
    redirect("/dashboard");
  }
  const reservation = reservations.find((item) => item.id === id);
  if (!reservation) {
    notFound();
  }
  const isAdmin = isTenantAdminRole(currentUser.role);

  return (
    <AdminShell active="approvals" allowedSections={isAdmin ? undefined : ["approvals"]} currentUser={currentUser} title="审批详情" description="查看申请信息、流程状态和处理动作。">
      <Card>
        <CardHeader className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <CardTitle>申请单 #{reservation.id.slice(0, 8).toUpperCase()}</CardTitle>
          <Link className="inline-flex h-9 items-center justify-center rounded-md border px-3 text-sm font-medium text-slate-700 hover:bg-slate-50" href="/approvals">
            返回待处理申请
          </Link>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
            <Info label="申请人" value={reservation.userName} />
            <Info label="仪器" value={reservation.instrumentName} />
            <Info label="团队" value={reservation.groupName || "部门直属"} />
            <Info label="状态" value={statusLabel(reservation.status)} />
            <Info label="预约时间段" value={formatDateTimeRange(reservation.startTime, reservation.endTime)} />
            <Info label="预约时长" value={formatDurationHours(reservation.startTime, reservation.endTime)} />
          </div>
          <div className="rounded-lg border bg-slate-50/40 p-4 text-sm leading-7 text-slate-700">{reservation.purpose}</div>
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <ReservationDetailDialog
              actions={<ReservationActions canCancel={false} canCheckIn={false} canCheckOut={false} id={reservation.id} status={reservation.status} />}
              description="查看完整预约信息，并在弹窗内完成审批。"
              reservation={reservation}
              triggerLabel="弹窗详情"
            />
            <div className="flex flex-col gap-2 sm:flex-row">
              <ReservationActions canCancel={false} canCheckIn={false} canCheckOut={false} id={reservation.id} status={reservation.status} />
            </div>
          </div>
        </CardContent>
      </Card>
    </AdminShell>
  );
}

function Info({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border bg-white p-4">
      <p className="text-xs text-slate-500">{label}</p>
      <p className="mt-1 break-words font-bold text-slate-900">{value}</p>
    </div>
  );
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
