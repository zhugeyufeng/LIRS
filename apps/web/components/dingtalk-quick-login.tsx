"use client";

import { useEffect, useMemo, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { CheckCircle2, Loader2 } from "lucide-react";
import { browserDingTalkQuickLogin, Tenant } from "@/lib/api";

type DingTalkRuntime = {
  runtime?: {
    permission?: {
      requestAuthCode?: (options: {
        corpId?: string;
        onSuccess?: (result: { code?: string }) => void;
        onFail?: (error: unknown) => void;
      }) => void;
    };
  };
  ready?: (callback: () => void) => void;
  error?: (callback: (error: unknown) => void) => void;
};

type DingTalkWindow = Window & {
  dd?: DingTalkRuntime;
  DingTalkPC?: DingTalkRuntime;
};

export function DingTalkQuickLogin({ tenants }: { tenants: Tenant[] }) {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);
  const tenantCode = searchParams.get("tenantCode") ?? "";
  const tenantId = searchParams.get("tenantId") ?? "";
  const nextPath = safeNextPath(searchParams.get("next")) ?? "/dashboard";
  const tenant = useMemo(() => resolveTenant(tenants, tenantId, tenantCode), [tenantCode, tenantId, tenants]);
  const corpId = searchParams.get("corpId") ?? "";

  const isDingTalk = isDingTalkClient();

  useEffect(() => {
    if (!isDingTalkClient()) {
      return;
    }
    let cancelled = false;
    async function run() {
      setPending(true);
      setMessage("正在通过钉钉免登录进入 LIRS。");
      try {
        const authCode = await requestDingTalkAuthCode(corpId);
        if (cancelled) {
          return;
        }
        const auth = await browserDingTalkQuickLogin({
          authCode,
          corpId,
          tenantId: tenant?.id ?? tenantId,
          tenantCode: tenant?.code ?? tenantCode,
          device: "dingtalk",
        });
        if (cancelled) {
          return;
        }
        setMessage(`已通过钉钉登录：${auth.user.name}`);
        router.replace(nextPath);
        router.refresh();
      } catch (error) {
        if (!cancelled) {
          setMessage(error instanceof Error ? error.message : "钉钉免登录失败");
        }
      } finally {
        if (!cancelled) {
          setPending(false);
        }
      }
    }
    run();
    return () => {
      cancelled = true;
    };
  }, [corpId, isDingTalk, nextPath, router, tenant, tenantCode, tenantId]);

  if (!isDingTalk && !message) {
    return null;
  }

  return (
    <div className="mb-4 rounded-lg border bg-white p-4 text-sm text-slate-700">
      <div className="flex items-start gap-3">
        {pending ? <Loader2 className="mt-0.5 h-4 w-4 shrink-0 animate-spin text-primary" /> : <CheckCircle2 className="mt-0.5 h-4 w-4 shrink-0 text-emerald-600" />}
        <div className="min-w-0">
          <p className="font-bold text-slate-900">钉钉客户端免登录</p>
          <p className="mt-1 break-words leading-6">{message || "检测到钉钉客户端，正在准备免登录。"}</p>
        </div>
      </div>
    </div>
  );
}

function requestDingTalkAuthCode(corpId: string) {
  return new Promise<string>((resolve, reject) => {
    const runtime = dingTalkRuntime();
    const requestAuthCode = runtime?.runtime?.permission?.requestAuthCode;
    if (!requestAuthCode) {
      reject(new Error("当前钉钉客户端不支持免登录授权。"));
      return;
    }
    const request = () => {
      requestAuthCode({
        corpId: corpId || undefined,
        onSuccess: (result) => {
          const code = result.code ?? "";
          if (code) {
            resolve(code);
          } else {
            reject(new Error("钉钉未返回免登录授权码。"));
          }
        },
        onFail: (error) => reject(new Error(`钉钉免登录授权失败：${formatDingTalkError(error)}`)),
      });
    };
    if (runtime.ready) {
      runtime.ready(request);
      runtime.error?.((error) => reject(new Error(`钉钉 JSAPI 初始化失败：${formatDingTalkError(error)}`)));
      return;
    }
    request();
  });
}

function dingTalkRuntime() {
  if (typeof window === "undefined") {
    return undefined;
  }
  const currentWindow = window as DingTalkWindow;
  return currentWindow.dd ?? currentWindow.DingTalkPC;
}

function isDingTalkClient() {
  if (typeof window === "undefined") {
    return false;
  }
  return /DingTalk/i.test(window.navigator.userAgent) || Boolean(dingTalkRuntime());
}

function resolveTenant(tenants: Tenant[], tenantId: string, tenantCode: string) {
  const normalizedCode = tenantCode.trim().toLowerCase();
  return tenants.find((tenant) => (tenantId && tenant.id === tenantId) || (normalizedCode && tenant.code.toLowerCase() === normalizedCode));
}

function safeNextPath(next: string | null) {
  if (!next || !next.startsWith("/") || next.startsWith("//")) {
    return null;
  }
  return next;
}

function formatDingTalkError(error: unknown) {
  if (error instanceof Error) {
    return error.message;
  }
  if (typeof error === "string") {
    return error;
  }
  try {
    return JSON.stringify(error);
  } catch {
    return "未知错误";
  }
}
