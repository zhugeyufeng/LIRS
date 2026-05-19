"use client";

import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";
import { Link2, Loader2 } from "lucide-react";
import { browserDingTalkLoginBindExisting } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { PasswordInput } from "@/components/ui/password-input";

export function DingTalkLoginBindForm({
  bindingToken,
  dingTalkName,
  nextPath,
  tenantName,
}: {
  bindingToken: string;
  dingTalkName: string;
  nextPath: string;
  tenantName: string;
}) {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setMessage("");
    try {
      const auth = await browserDingTalkLoginBindExisting({
        bindingToken,
        email,
        password,
        device: "dingtalk-web-bind",
      });
      setMessage(`已绑定并登录：${auth.user.name}`);
      router.replace(nextPath);
      router.refresh();
    } catch (error) {
      const rawMessage = error instanceof Error ? error.message : "";
      if (rawMessage.includes("invalid email") || rawMessage.includes("invalid dingtalk binding")) {
        setMessage("邮箱、密码或绑定凭证不正确。");
      } else {
        setMessage(rawMessage || "绑定失败");
      }
    } finally {
      setPending(false);
    }
  }

  return (
    <form className="space-y-4" onSubmit={submit}>
      <div className="rounded-md bg-slate-50 p-3 text-sm leading-6 text-slate-700">
        <p>钉钉身份：{dingTalkName || "待绑定钉钉用户"}</p>
        <p>绑定机构：{tenantName || "当前机构"}</p>
      </div>
      <label className="block space-y-2">
        <span className="text-sm font-medium">现有账户邮箱</span>
        <input className="h-10 w-full rounded-md border bg-white px-3 text-sm" name="email" onChange={(event) => setEmail(event.currentTarget.value)} required type="email" value={email} />
      </label>
      <label className="block space-y-2">
        <span className="text-sm font-medium">现有账户密码</span>
        <PasswordInput name="password" onChange={(event) => setPassword(event.currentTarget.value)} required value={password} />
      </label>
      <Button className="w-full" disabled={pending || !bindingToken} type="submit">
        {pending ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" /> : <Link2 className="h-4 w-4" aria-hidden="true" />}
        {pending ? "绑定中" : "绑定并登录"}
      </Button>
      {message ? <p className="rounded-md bg-slate-100 p-3 text-sm text-slate-700">{message}</p> : null}
    </form>
  );
}
