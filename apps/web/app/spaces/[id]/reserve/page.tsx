import Link from "next/link";
import { notFound } from "next/navigation";
import { CalendarClock } from "lucide-react";
import { AppShell } from "@/components/app-shell";
import { SpaceReservationForm } from "@/components/extension-forms";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";

export default async function SpaceReservePage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const currentUser = await api.me();
  const [spaces, reservations] = await Promise.all([api.spaces(), api.spaceReservations()]);
  const space = spaces.find((item) => item.id === id);
  if (!space) {
    notFound();
  }
  const spaceReservations = reservations.filter((item) => item.spaceId === id);

  return (
    <AppShell currentUser={currentUser}>
      <Link className="mb-5 inline-flex items-center text-sm text-slate-600 hover:text-primary" href="/spaces">
        返回空间资源
      </Link>
      <div className="mb-6">
        <h1 className="text-2xl font-bold">{space.name}</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          {space.department || "未设置部门"} / {space.location} / 容量 {space.capacity} 人
        </p>
      </div>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_360px]">
        <div className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <CalendarClock className="h-5 w-5 text-primary" />
                预约记录
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              {spaceReservations.map((item) => (
                <div className="rounded-lg border p-4 text-sm" key={item.id}>
                  <div className="flex flex-col justify-between gap-3 sm:flex-row sm:items-start">
                    <div className="min-w-0">
                      <p className="break-words font-semibold text-slate-900">{item.purpose}</p>
                      <p className="mt-1 break-words text-xs text-slate-500">{item.requester}</p>
                    </div>
                    <span className="w-fit rounded bg-slate-100 px-2 py-1 text-xs font-bold text-slate-700">{item.status}</span>
                  </div>
                  <p className="mt-3 text-slate-600">
                    {formatDateTime(item.startTime)} - {formatDateTime(item.endTime)}
                  </p>
                </div>
              ))}
              {spaceReservations.length === 0 ? <p className="rounded-lg border border-dashed p-4 text-sm text-slate-500">暂无预约记录。</p> : null}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>空间说明</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3 text-sm leading-6 text-slate-600">
              <p>{space.description || "未填写空间说明。"}</p>
              <p>门禁点位：{space.accessControlPoint || "未设置"}</p>
              <p>状态：{space.status}</p>
            </CardContent>
          </Card>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>提交空间预约</CardTitle>
          </CardHeader>
          <CardContent>
            <SpaceReservationForm actorName={currentUser.name} space={space} />
          </CardContent>
        </Card>
      </div>
    </AppShell>
  );
}

function formatDateTime(value: string) {
  return new Date(value).toLocaleString("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    timeZone: "Asia/Shanghai",
  });
}
