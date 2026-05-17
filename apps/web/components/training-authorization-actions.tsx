"use client";

import { FormEvent, startTransition, useState } from "react";
import { useRouter } from "next/navigation";
import { Edit3, Save } from "lucide-react";
import { AdminDialog } from "@/components/admin-dialog";
import { Button } from "@/components/ui/button";
import { browserPatch, type Instrument, type TrainingAuthorization, type TrainingAuthorizationPayload, type TrainingCourse, type User } from "@/lib/api";

export function TrainingAuthorizationEditDialog({
  authorization,
  courses,
  instruments,
  users,
}: {
  authorization: TrainingAuthorization;
  courses: TrainingCourse[];
  instruments: Instrument[];
  users: User[];
}) {
  const router = useRouter();
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setMessage("");
    const form = new FormData(event.currentTarget);
    const selectedUserId = String(form.get("userId") ?? "").trim();
    const selectedUser = users.find((user) => user.id === selectedUserId);
    const payload: TrainingAuthorizationPayload = {
      userId: selectedUserId || undefined,
      userName: selectedUser?.name ?? String(form.get("userName") ?? authorization.userName).trim(),
      courseId: String(form.get("courseId") ?? "").trim() || undefined,
      instrumentId: String(form.get("instrumentId") ?? "").trim() || undefined,
      status: String(form.get("status") ?? authorization.status),
      expiresAt: toIsoDateTime(form.get("expiresAt")),
      notes: String(form.get("notes") ?? "").trim(),
    };
    try {
      await browserPatch<TrainingAuthorization>(`/api/training/authorizations/${authorization.id}`, payload);
      setMessage("授权记录已更新");
      startTransition(() => router.refresh());
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "保存失败");
    } finally {
      setPending(false);
    }
  }

  return (
    <AdminDialog
      description={`正在修改：${authorization.userName} / ${authorization.courseTitle || "未关联课程"}`}
      title="修改授权记录"
      trigger={
        <Button className="w-full sm:w-auto" size="sm" variant="outline">
          <Edit3 className="h-4 w-4" aria-hidden="true" />
          修改
        </Button>
      }
    >
      <form className="space-y-4" onSubmit={submit}>
        <div className="grid gap-4 md:grid-cols-2">
          <label className="grid gap-1 text-sm">
            <span className="font-medium text-slate-700">授权用户</span>
            <select className="h-10 rounded-md border bg-white px-3 text-sm" defaultValue={authorization.userId ?? ""} name="userId">
              <option value="">仅记录姓名</option>
              {users.map((user) => (
                <option key={user.id} value={user.id}>
                  {user.name} / {user.email}
                </option>
              ))}
            </select>
          </label>
          <label className="grid gap-1 text-sm">
            <span className="font-medium text-slate-700">姓名</span>
            <input className="h-10 rounded-md border bg-white px-3 text-sm" defaultValue={authorization.userName} name="userName" placeholder="未选择用户时填写姓名" />
          </label>
        </div>
        <div className="grid gap-4 md:grid-cols-2">
          <label className="grid gap-1 text-sm">
            <span className="font-medium text-slate-700">培训课程</span>
            <select className="h-10 rounded-md border bg-white px-3 text-sm" defaultValue={authorization.courseId ?? ""} name="courseId">
              <option value="">不关联课程</option>
              {courses.map((course) => (
                <option key={course.id} value={course.id}>
                  {course.title}
                </option>
              ))}
            </select>
          </label>
          <label className="grid gap-1 text-sm">
            <span className="font-medium text-slate-700">授权仪器</span>
            <select className="h-10 rounded-md border bg-white px-3 text-sm" defaultValue={authorization.instrumentId ?? ""} name="instrumentId">
              <option value="">不关联仪器</option>
              {instruments.map((instrument) => (
                <option key={instrument.id} value={instrument.id}>
                  {instrument.name}
                </option>
              ))}
            </select>
          </label>
        </div>
        <div className="grid gap-4 md:grid-cols-2">
          <label className="grid gap-1 text-sm">
            <span className="font-medium text-slate-700">授权状态</span>
            <select className="h-10 rounded-md border bg-white px-3 text-sm" defaultValue={authorization.status} name="status">
              <option value="pending">待审核</option>
              <option value="active">已授权</option>
              <option value="expired">已过期</option>
              <option value="revoked">已撤销</option>
            </select>
          </label>
          <label className="grid gap-1 text-sm">
            <span className="font-medium text-slate-700">到期时间</span>
            <input className="h-10 rounded-md border bg-white px-3 text-sm" defaultValue={toLocalDateTime(authorization.expiresAt)} name="expiresAt" required type="datetime-local" />
          </label>
        </div>
        <label className="grid gap-1 text-sm">
          <span className="font-medium text-slate-700">备注</span>
          <textarea className="min-h-24 rounded-md border bg-white px-3 py-2 text-sm leading-6" defaultValue={authorization.notes} name="notes" placeholder="填写授权范围、限制条件或驳回原因" />
        </label>
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <p className="text-xs text-muted-foreground">{message || "审批通过后可用于后续预约准入判断和门禁权限同步。"}</p>
          <Button disabled={pending} type="submit">
            <Save className="h-4 w-4" aria-hidden="true" />
            {pending ? "保存中..." : "保存"}
          </Button>
        </div>
      </form>
    </AdminDialog>
  );
}

function toIsoDateTime(value: FormDataEntryValue | null) {
  const date = new Date(String(value ?? ""));
  return Number.isNaN(date.getTime()) ? "" : date.toISOString();
}

function toLocalDateTime(value: string) {
  const date = new Date(value);
  const local = new Date(date.getTime() - date.getTimezoneOffset() * 60000);
  return local.toISOString().slice(0, 16);
}
