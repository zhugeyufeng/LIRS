"use client";

import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";
import { browserPatch, clearAuth, PasswordChangePayload, User } from "@/lib/api";
import { confirmTwice } from "@/lib/confirm";
import { Button } from "@/components/ui/button";
import { PasswordInput } from "@/components/ui/password-input";
import { roleLabel } from "@/lib/permissions";
import { userStatusLabel } from "@/lib/status-labels";

export function ProfileSettingsForm({ user }: { user: User }) {
  return (
    <div className="space-y-4">
      <div className="grid gap-3 rounded-lg border bg-slate-50 p-4 text-sm md:grid-cols-2">
        <ReadonlyItem label="姓名" value={user.name} />
        <ReadonlyItem label="手机号" value={user.phone} />
        <ReadonlyItem label="所属部门/实验室" value={user.department} />
        <ReadonlyItem label="登录邮箱" value={user.email} />
        <ReadonlyItem label="所属单位" value={user.tenantName} />
        <ReadonlyItem label="角色" value={roleLabel(user.role)} />
        <ReadonlyItem label="账号状态" value={userStatusLabel(user.status)} />
      </div>
      <p className="rounded-lg border border-amber-100 bg-amber-50 p-3 text-xs leading-5 text-amber-800">
        姓名、手机号和所属部门/实验室由管理员在管理后台维护，个人账号不可自行修改。
      </p>
    </div>
  );
}

export function PasswordSettingsForm() {
  const router = useRouter();
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!confirmTwice("确定修改当前账号密码吗？", "请再次确认。修改后当前账号的所有登录会话会失效。")) {
      return;
    }
    setPending(true);
    setMessage("");
    const form = new FormData(event.currentTarget);
    const currentPassword = String(form.get("currentPassword") ?? "");
    const newPassword = String(form.get("newPassword") ?? "");
    const confirmPassword = String(form.get("confirmPassword") ?? "");
    if (newPassword.length < 8) {
      setMessage("新密码至少 8 位。");
      setPending(false);
      return;
    }
    if (newPassword !== confirmPassword) {
      setMessage("两次输入的新密码不一致。");
      setPending(false);
      return;
    }

    const payload: PasswordChangePayload = { currentPassword, newPassword };
    try {
      await browserPatch<{ ok: boolean }>("/api/me/password", payload);
      clearAuth();
      router.push("/login");
      router.refresh();
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "修改失败");
      setPending(false);
    }
  }

  return (
    <form className="space-y-4" onSubmit={submit}>
      <Field hint="修改对象：当前登录账号" label="当前密码" name="currentPassword" type="password" />
      <div className="grid gap-4 md:grid-cols-2">
        <Field hint="填写当前账号的新密码" label="新密码" minLength={8} name="newPassword" type="password" />
        <Field hint="再次填写当前账号的新密码" label="确认新密码" minLength={8} name="confirmPassword" type="password" />
      </div>
      <p className="rounded-lg border border-amber-100 bg-amber-50 p-3 text-xs leading-5 text-amber-800">
        修改密码后，当前账号的所有登录会话会失效，需要重新登录。
      </p>
      <Button className="w-full sm:w-auto" disabled={pending} type="submit" variant="outline">
        {pending ? "修改中..." : "修改密码"}
      </Button>
      {message ? <p className="rounded-md bg-slate-100 p-3 text-sm text-slate-700">{message}</p> : null}
    </form>
  );
}

function Field({
  defaultValue,
  hint,
  label,
  minLength,
  name,
  type = "text",
}: {
  defaultValue?: string;
  hint?: string;
  label: string;
  minLength?: number;
  name: string;
  type?: string;
}) {
  return (
    <label className="block space-y-2">
      <span className="text-sm font-medium">{label}</span>
      {type === "password" ? (
        <PasswordInput className="focus:ring-primary/20" defaultValue={defaultValue} minLength={minLength} name={name} required />
      ) : (
        <input
          className="h-10 w-full rounded-md border bg-white px-3 text-sm outline-none focus:ring-2 focus:ring-primary/20"
          defaultValue={defaultValue}
          minLength={minLength}
          name={name}
          required
          type={type}
        />
      )}
      <FieldHint value={hint ?? `当前：${formatCurrent(defaultValue ?? "")}`} />
    </label>
  );
}

function FieldHint({ value }: { value: string }) {
  return <span className="block break-words text-xs text-slate-500">{value}</span>;
}

function formatCurrent(value: string) {
  const text = value.trim();
  return text === "" ? "未设置" : text;
}

function ReadonlyItem({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-xs text-slate-500">{label}</p>
      <p className="mt-1 break-words font-medium text-slate-800">{formatCurrent(value)}</p>
    </div>
  );
}
