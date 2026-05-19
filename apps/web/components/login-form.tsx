"use client";

import { FormEvent, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { BrowserRequestError, browserLogin, Tenant } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { PasswordInput } from "@/components/ui/password-input";

export function LoginForm({ tenants }: { tenants: Tenant[] }) {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [tenantChoiceOpen, setTenantChoiceOpen] = useState(false);
  const [selectedTenantId, setSelectedTenantId] = useState("");

  async function login(tenantId = "") {
    setPending(true);
    setMessage("");
    try {
      const auth = await browserLogin({
        tenantId,
        email,
        password,
        device: "web",
      });
      setMessage(`已登录：${auth.user.name}`);
      router.push(safeNextPath(searchParams.get("next")) ?? "/dashboard");
      router.refresh();
    } catch (error) {
      const rawMessage = error instanceof Error ? error.message : "";
      if ((error instanceof BrowserRequestError && error.status === 401) || rawMessage.includes("invalid email")) {
        setMessage("邮箱或密码不正确。");
      } else if (rawMessage.includes("tenant is required") || rawMessage.includes("multi-tenant")) {
        setSelectedTenantId(tenants[0]?.id ?? "");
        setTenantChoiceOpen(true);
        setMessage("该账号隶属多个机构，请选择本次登录机构。");
      } else {
        setMessage(rawMessage || "登录失败");
      }
    } finally {
      setPending(false);
    }
  }

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await login();
  }

  return (
    <>
      <form className="space-y-4" onSubmit={submit}>
      <label className="block space-y-2">
        <span className="text-sm font-medium">邮箱</span>
        <input className="h-10 w-full rounded-md border bg-white px-3 text-sm" name="email" onChange={(event) => setEmail(event.currentTarget.value)} required type="email" value={email} />
      </label>
      <label className="block space-y-2">
        <span className="text-sm font-medium">密码</span>
        <PasswordInput name="password" onChange={(event) => setPassword(event.currentTarget.value)} required value={password} />
      </label>
      <Button className="w-full" disabled={pending} type="submit">
        {pending ? "登录中..." : "登录"}
      </Button>
      {message ? <p className="rounded-md bg-slate-100 p-3 text-sm text-slate-700">{message}</p> : null}
      </form>

      {tenantChoiceOpen ? (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/50 p-4">
          <div className="w-full max-w-sm rounded-lg bg-white p-5 shadow-xl">
            <h2 className="text-lg font-bold text-slate-900">选择登录机构</h2>
            <p className="mt-2 text-sm leading-6 text-slate-600">检测到该账号隶属于多个机构，请选择本次要进入的机构。</p>
            <select className="mt-4 h-10 w-full rounded-md border bg-white px-3 text-sm" onChange={(event) => setSelectedTenantId(event.currentTarget.value)} value={selectedTenantId}>
              {tenants.map((tenant) => (
                <option key={tenant.id} value={tenant.id}>
                  {tenant.name}
                </option>
              ))}
            </select>
            <div className="mt-5 flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
              <Button disabled={pending} onClick={() => setTenantChoiceOpen(false)} type="button" variant="outline">
                取消
              </Button>
              <Button disabled={pending || !selectedTenantId} onClick={() => login(selectedTenantId)} type="button">
                {pending ? "登录中..." : "进入机构"}
              </Button>
            </div>
          </div>
        </div>
      ) : null}
    </>
  );
}

function safeNextPath(next: string | null) {
  if (!next || !next.startsWith("/") || next.startsWith("//")) {
    return null;
  }
  return next;
}
