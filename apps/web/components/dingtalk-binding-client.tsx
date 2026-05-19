"use client";

import { useEffect, useMemo, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { QRCodeSVG } from "qrcode.react";
import { CheckCircle2, ExternalLink, Link2Off, Loader2, QrCode } from "lucide-react";
import { browserDelete, browserPost, DingTalkBinding } from "@/lib/api";
import { confirmTwice } from "@/lib/confirm";
import { Button } from "@/components/ui/button";

export function DingTalkBindingClient({ binding }: { binding: DingTalkBinding }) {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);
  const authCode = searchParams.get("authCode") || searchParams.get("code") || "";
  const state = searchParams.get("state") || "";
  const bindError = searchParams.get("error") || "";
  const authUrl = useMemo(() => binding.authUrl ?? "", [binding.authUrl]);

  useEffect(() => {
    if (!authCode) {
      if (bindError) {
        setMessage(`钉钉授权失败：${bindError}`);
      }
      return;
    }
    let cancelled = false;
    async function bind() {
      setPending(true);
      setMessage("正在完成钉钉绑定。");
      try {
        await browserPost<DingTalkBinding>("/api/me/dingtalk-binding", { authCode, state });
        if (!cancelled) {
          setMessage("钉钉绑定已完成。");
          router.replace("/settings/dingtalk");
          router.refresh();
        }
      } catch (error) {
        if (!cancelled) {
          setMessage(error instanceof Error ? error.message : "绑定失败");
        }
      } finally {
        if (!cancelled) {
          setPending(false);
        }
      }
    }
    bind();
    return () => {
      cancelled = true;
    };
  }, [authCode, bindError, router, state]);

  async function unbind() {
    if (!confirmTwice("确定解除当前账号的钉钉绑定吗？", "请再次确认。解除后该账号不会继续接收个人钉钉通知。")) {
      return;
    }
    setPending(true);
    setMessage("");
    try {
      await browserDelete<DingTalkBinding>("/api/me/dingtalk-binding");
      setMessage("钉钉绑定已解除。");
      router.refresh();
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "解绑失败");
    } finally {
      setPending(false);
    }
  }

  return (
    <div className="space-y-5">
      <div className="rounded-lg border bg-white p-4">
        <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
          <div className="min-w-0">
            <div className="flex items-center gap-2">
              {binding.bound ? <CheckCircle2 className="h-5 w-5 text-emerald-600" /> : <QrCode className="h-5 w-5 text-primary" />}
              <h2 className="font-bold text-slate-900">{binding.bound ? "已绑定钉钉" : "扫码绑定钉钉"}</h2>
            </div>
            <div className="mt-3 grid gap-2 text-sm text-slate-600 sm:grid-cols-2">
              <Summary label="绑定状态" value={binding.bound ? "已绑定" : "未绑定"} />
              <Summary label="钉钉姓名" value={binding.name || "未绑定"} />
              <Summary label="企业 UserId" value={binding.userId || "未绑定"} />
              <Summary label="UnionId" value={binding.unionId || "未绑定"} />
            </div>
          </div>
          {binding.bound ? (
            <Button className="w-full md:w-auto" disabled={pending} onClick={unbind} type="button" variant="outline">
              {pending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Link2Off className="h-4 w-4" />}
              解除绑定
            </Button>
          ) : null}
        </div>
      </div>

      {!binding.bound ? (
        <div className="rounded-lg border bg-white p-4">
          {binding.authUrl ? (
            <div className="grid gap-5 md:grid-cols-[260px_1fr] md:items-center">
              <div className="flex h-[260px] w-full items-center justify-center rounded-lg border bg-slate-50 md:w-[260px]">
                {authUrl ? <QRCodeSVG className="h-[220px] w-[220px]" includeMargin size={220} value={authUrl} /> : <QrCode className="h-24 w-24 text-slate-400" />}
              </div>
              <div className="space-y-4">
                <div>
                  <h3 className="text-base font-bold text-slate-900">使用钉钉扫码授权</h3>
                  <p className="mt-2 text-sm leading-6 text-slate-600">扫码后会回到当前系统并自动完成绑定，绑定成功后个人通知会同步推送到钉钉企业应用。</p>
                </div>
                <Button asChild className="w-full sm:w-auto" variant="outline">
                  <a href={binding.authUrl} rel="noreferrer" target="_blank">
                    <ExternalLink className="h-4 w-4" />
                    打开钉钉授权页
                  </a>
                </Button>
              </div>
            </div>
          ) : (
            <div className="rounded-md bg-amber-50 p-4 text-sm leading-6 text-amber-900">钉钉应用尚未启用或缺少扫码绑定回调地址，请联系管理员配置后再绑定。</div>
          )}
        </div>
      ) : null}

      {message ? <p className="rounded-md bg-slate-100 p-3 text-sm text-slate-700">{message}</p> : null}
      {pending && !message ? <p className="rounded-md bg-slate-100 p-3 text-sm text-slate-700">正在处理，请稍候。</p> : null}
    </div>
  );
}

function Summary({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0 rounded-md bg-slate-50 px-3 py-2">
      <p className="text-xs text-slate-500">{label}</p>
      <p className="mt-1 break-words font-medium text-slate-800">{value}</p>
    </div>
  );
}
