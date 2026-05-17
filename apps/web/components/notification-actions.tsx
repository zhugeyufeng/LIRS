"use client";

import { FormEvent, startTransition, useState } from "react";
import { useRouter } from "next/navigation";
import { CheckCheck, Megaphone, Send } from "lucide-react";
import { browserPatch, browserPost, Notification } from "@/lib/api";
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

export function MarkAllNotificationsRead({ disabled }: { disabled: boolean }) {
  const router = useRouter();
  const [pending, setPending] = useState(false);
  const [message, setMessage] = useState("");

  async function markAll() {
    setPending(true);
    setMessage("");
    try {
      await browserPatch<{ count: number }>("/api/notifications/read-all");
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

export function AnnouncementForm({ departments }: { departments: string[] }) {
  const router = useRouter();
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);
  const [scope, setScope] = useState("global");

  async function submit(event: FormEvent<HTMLFormElement>, close?: () => void) {
    event.preventDefault();
    setPending(true);
    setMessage("");
    const formElement = event.currentTarget;
    const form = new FormData(formElement);
    try {
      await browserPost<Notification>("/api/notifications", {
        title: String(form.get("title") ?? ""),
        body: String(form.get("body") ?? ""),
        level: String(form.get("level") ?? "info"),
        targetScope: scope,
        target: String(form.get("target") ?? ""),
      });
      setMessage("公告已发布");
      formElement.reset();
      close?.();
      startTransition(() => {
        router.refresh();
      });
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "发布失败");
    } finally {
      setPending(false);
    }
  }

  return (
    <div className="space-y-2">
      <AdminDialog
        description="公告会写入通知系统；消息中心只负责展示用户可见通知。"
        maxWidth="max-w-3xl"
        title="发布公告"
        trigger={
          <Button className="w-full" disabled={pending}>
            <Megaphone className="h-4 w-4" aria-hidden="true" />
            发布公告
          </Button>
        }
      >
        {(close) => (
          <form className="space-y-4" onSubmit={(event) => submit(event, close)}>
            <input className="h-10 w-full rounded-md border bg-white px-3 text-sm" name="title" placeholder="公告标题" required />
            <textarea className="min-h-24 w-full rounded-md border bg-white px-3 py-2 text-sm" name="body" placeholder="公告内容" required />
            <div className="grid gap-3 sm:grid-cols-2">
              <select className="h-10 rounded-md border bg-white px-3 text-sm" name="level" defaultValue="info">
                <option value="info">普通</option>
                <option value="warning">提醒</option>
                <option value="success">成功</option>
              </select>
              <select className="h-10 rounded-md border bg-white px-3 text-sm" name="targetScope" value={scope} onChange={(event) => setScope(event.target.value)}>
                <option value="global">全局</option>
                <option value="department">部门</option>
                <option value="personal">个人</option>
              </select>
            </div>
            {scope !== "global" ? (
              scope === "personal" ? (
                <input className="h-10 w-full rounded-md border bg-white px-3 text-sm" name="target" placeholder="用户邮箱或用户 ID" required />
              ) : (
                <select className="h-10 w-full rounded-md border bg-white px-3 text-sm" name="target" required>
                  <option value="">选择部门</option>
                  {departments.map((item) => (
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
                {pending ? "发布中..." : "发布公告"}
              </Button>
            </div>
          </form>
        )}
      </AdminDialog>
      {message ? <p className="text-xs text-slate-500">{message}</p> : null}
    </div>
  );
}
