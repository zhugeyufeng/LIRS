"use client";

import { FormEvent, startTransition, useState } from "react";
import { useRouter } from "next/navigation";
import { CheckCheck, Megaphone, Pencil, Send, Trash2 } from "lucide-react";
import { browserDelete, browserPatch, browserPost, Notification } from "@/lib/api";
import { confirmTwice } from "@/lib/confirm";
import { AdminDialog } from "@/components/admin-dialog";
import { Button } from "@/components/ui/button";

export function MarkNotificationRead({ id, read }: { id: string; read: boolean }) {
  const router = useRouter();
  const [done, setDone] = useState(read);
  const [message, setMessage] = useState("");

  async function mark() {
    setMessage("");
    try {
      const item = await browserPatch<Notification>(`/api/notifications/${id}/read`);
      setDone(item.read);
      startTransition(() => {
        router.refresh();
      });
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "标记失败");
    }
  }

  return (
    <div className="flex shrink-0 flex-wrap items-center gap-2 sm:justify-end">
      <Button className="h-10 w-full sm:h-8 sm:w-auto" disabled={done} onClick={mark} size="sm" variant="outline">
        {done ? "已读" : "标为已读"}
      </Button>
      {message ? <span className="text-xs text-red-600">{message}</span> : null}
    </div>
  );
}

export function MarkAllNotificationsRead({ disabled, source }: { disabled: boolean; source?: "system" | "announcement" }) {
  const router = useRouter();
  const [pending, setPending] = useState(false);
  const [message, setMessage] = useState("");

  async function markAll() {
    setPending(true);
    setMessage("");
    try {
      await browserPatch<{ count: number }>(notificationPath("/api/notifications/read-all", undefined, source));
      startTransition(() => {
        router.refresh();
      });
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "批量标记失败");
    } finally {
      setPending(false);
    }
  }

  return (
    <div className="flex flex-col items-stretch gap-2 sm:items-end">
      <Button className="h-10 w-full sm:h-8 sm:w-auto" disabled={disabled || pending} onClick={markAll} size="sm" variant="outline">
        <CheckCheck className="h-4 w-4" aria-hidden="true" />
        {pending ? "处理中..." : "全选标为已读"}
      </Button>
      {message ? <span className="text-xs text-red-600">{message}</span> : null}
    </div>
  );
}

export function AnnouncementForm({
  departments,
  groups = [],
  initial,
  mode = "create",
  selectedTenantId,
}: {
  departments: string[];
  groups?: string[];
  initial?: Notification;
  mode?: "create" | "edit";
  selectedTenantId?: string;
}) {
  const router = useRouter();
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);
  const [scope, setScope] = useState(initial?.targetScope ?? "global");
  const isEdit = mode === "edit" && Boolean(initial);
  const targetOptions = scope === "group" ? groups : departments;

  async function submit(event: FormEvent<HTMLFormElement>, close?: () => void) {
    event.preventDefault();
    if (isEdit && !confirmTwice(`确定修改公告“${initial?.title ?? ""}”吗？`, "请再次确认。修改后用户看到的公告内容会立即更新。")) {
      return;
    }
    setPending(true);
    setMessage("");
    const formElement = event.currentTarget;
    const form = new FormData(formElement);
    const path = notificationPath(isEdit ? `/api/notifications/${initial?.id}` : "/api/notifications", selectedTenantId);
    const payload = {
      title: String(form.get("title") ?? ""),
      body: String(form.get("body") ?? ""),
      level: String(form.get("level") ?? "info"),
      targetScope: scope,
      target: String(form.get("target") ?? ""),
      userId: scope === "personal" ? String(form.get("target") ?? "") : "",
      groupName: scope === "group" ? String(form.get("target") ?? "") : "",
      department: scope === "department" ? String(form.get("target") ?? "") : "",
    };
    try {
      if (isEdit) {
        await browserPatch<Notification>(path, payload);
      } else {
        await browserPost<Notification>(path, payload);
      }
      setMessage(isEdit ? "公告已保存" : "公告已发布");
      formElement.reset();
      close?.();
      startTransition(() => {
        router.refresh();
      });
    } catch (error) {
      setMessage(error instanceof Error ? error.message : (isEdit ? "保存失败" : "发布失败"));
    } finally {
      setPending(false);
    }
  }

  return (
    <div className="space-y-2">
      <AdminDialog
        description="公告会写入通知系统；消息中心只负责展示用户可见通知。"
        maxWidth="max-w-3xl"
        title={isEdit ? "修改公告" : "发布公告"}
        trigger={
          <Button className={isEdit ? "w-full sm:w-auto" : "w-full"} disabled={pending} size={isEdit ? "sm" : "default"} variant={isEdit ? "outline" : "default"}>
            {isEdit ? <Pencil className="h-4 w-4" aria-hidden="true" /> : <Megaphone className="h-4 w-4" aria-hidden="true" />}
            {isEdit ? "修改" : "发布公告"}
          </Button>
        }
      >
        {(close) => (
          <form className="space-y-4" onSubmit={(event) => submit(event, close)}>
            <input className="h-10 w-full rounded-md border bg-white px-3 text-sm" defaultValue={initial?.title ?? ""} name="title" placeholder="公告标题" required />
            <textarea className="min-h-24 w-full rounded-md border bg-white px-3 py-2 text-sm" defaultValue={initial?.body ?? ""} name="body" placeholder="公告内容" required />
            <div className="grid gap-3 sm:grid-cols-2">
              <select className="h-10 rounded-md border bg-white px-3 text-sm" name="level" defaultValue={initial?.level ?? "info"}>
                <option value="info">普通</option>
                <option value="warning">提醒</option>
                <option value="success">成功</option>
              </select>
              <select className="h-10 rounded-md border bg-white px-3 text-sm" name="targetScope" value={scope} onChange={(event) => setScope(event.target.value)}>
                <option value="global">全局</option>
                <option value="department">部门</option>
                <option value="group">团队</option>
                <option value="personal">个人</option>
              </select>
            </div>
            {scope !== "global" ? (
              scope === "personal" ? (
                <input className="h-10 w-full rounded-md border bg-white px-3 text-sm" defaultValue={initial?.userId ?? ""} name="target" placeholder="用户邮箱或用户 ID" required />
              ) : (
                <select className="h-10 w-full rounded-md border bg-white px-3 text-sm" defaultValue={scope === "group" ? initial?.groupName ?? "" : initial?.department ?? ""} name="target" required>
                  <option value="">{scope === "group" ? "选择团队" : "选择部门"}</option>
                  {targetOptions.map((item) => (
                    <option key={item} value={item}>
                      {item}
                    </option>
                  ))}
                </select>
              )
            ) : null}
            <div className="flex justify-end">
              <Button className="w-full sm:w-auto" disabled={pending} type="submit">
                <Send className="h-4 w-4" aria-hidden="true" />
                {pending ? (isEdit ? "保存中..." : "发布中...") : isEdit ? "保存公告" : "发布公告"}
              </Button>
            </div>
          </form>
        )}
      </AdminDialog>
      {message ? <p className="text-xs text-slate-500">{message}</p> : null}
    </div>
  );
}

export function DeleteNotificationButton({ id, selectedTenantId, title }: { id: string; selectedTenantId?: string; title: string }) {
  const router = useRouter();
  const [pending, setPending] = useState(false);
  const [message, setMessage] = useState("");

  async function remove() {
    if (!confirmTwice(`确定删除“${title}”吗？`, "请再次确认删除该公告。删除后普通用户不会再看到这条公告。")) {
      return;
    }
    setPending(true);
    setMessage("");
    try {
      await browserDelete<Notification>(notificationPath(`/api/notifications/${id}`, selectedTenantId));
      startTransition(() => {
        router.refresh();
      });
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "删除失败");
    } finally {
      setPending(false);
    }
  }

  return (
    <div className="flex flex-col gap-1">
      <Button className="w-full sm:w-auto" disabled={pending} onClick={remove} size="sm" type="button" variant="outline">
        <Trash2 className="h-4 w-4" aria-hidden="true" />
        {pending ? "删除中..." : "删除"}
      </Button>
      {message ? <span className="text-xs text-red-600">{message}</span> : null}
    </div>
  );
}

function notificationPath(path: string, tenantId?: string, source?: string) {
  const normalizedTenantId = tenantId?.trim() ?? "";
  const normalizedSource = source?.trim() ?? "";
  if (!normalizedTenantId && !normalizedSource) {
    return path;
  }
  const query = new URLSearchParams();
  if (normalizedTenantId) {
    query.set("tenantId", normalizedTenantId);
  }
  if (normalizedSource) {
    query.set("source", normalizedSource);
  }
  return `${path}?${query.toString()}`;
}
