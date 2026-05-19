"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { ExternalLink, Loader2, QrCode } from "lucide-react";
import { browserDingTalkWebLogin, browserDingTalkWebLoginIntent, Tenant } from "@/lib/api";
import { Button } from "@/components/ui/button";

export function DingTalkWebLogin({ tenants }: { tenants: Tenant[] }) {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);
  const [authUrl, setAuthUrl] = useState("");
  const tenantCode = searchParams.get("tenantCode") ?? "";
  const tenantId = searchParams.get("tenantId") ?? "";
  const authCode = searchParams.get("authCode") ?? searchParams.get("code") ?? "";
  const state = searchParams.get("state") ?? "";
  const bindToken = searchParams.get("bindToken") ?? "";
  const nextPath = safeNextPath(searchParams.get("next")) ?? "/dashboard";
  const initialTenant = useMemo(() => resolveTenant(tenants, tenantId, tenantCode), [tenantCode, tenantId, tenants]);
  const [selectedTenantId, setSelectedTenantId] = useState(initialTenant?.id ?? (tenants.length === 1 ? tenants[0]?.id ?? "" : ""));
  const tenant = useMemo(() => tenants.find((item) => item.id === selectedTenantId) ?? initialTenant, [initialTenant, selectedTenantId, tenants]);

  useEffect(() => {
    if (initialTenant?.id) {
      setSelectedTenantId(initialTenant.id);
      return;
    }
    if (tenants.length === 1) {
      setSelectedTenantId(tenants[0]?.id ?? "");
    }
  }, [initialTenant, tenants]);

  useEffect(() => {
    if (!authCode || !state || bindToken) {
      return;
    }
    let cancelled = false;
    async function finishLogin() {
      setPending(true);
      setMessage("正在完成钉钉扫码登录。");
      try {
        const result = await browserDingTalkWebLogin({
          authCode,
          state,
          tenantId: tenant?.id ?? tenantId,
          tenantCode: tenant?.code ?? tenantCode,
          device: "dingtalk-web",
        });
        if (cancelled) {
          return;
        }
        if (result.bound) {
          setMessage(`已通过钉钉登录：${result.auth?.user.name ?? result.dingTalkName ?? "当前用户"}`);
          router.replace(nextPath);
          router.refresh();
          return;
        }
        const query = new URLSearchParams();
        query.set("bindToken", result.bindingToken ?? "");
        query.set("tenantId", result.tenantId ?? tenant?.id ?? tenantId);
        query.set("tenantCode", result.tenantCode ?? tenant?.code ?? tenantCode);
        if (result.dingTalkName) {
          query.set("dingTalkName", result.dingTalkName);
        }
        query.set("next", nextPath);
        router.replace(`/login/dingtalk-bind?${query.toString()}`);
      } catch (error) {
        if (!cancelled) {
          setMessage(error instanceof Error ? error.message : "钉钉扫码登录失败");
        }
      } finally {
        if (!cancelled) {
          setPending(false);
        }
      }
    }
    finishLogin();
    return () => {
      cancelled = true;
    };
  }, [authCode, bindToken, nextPath, router, state, tenant, tenantCode, tenantId]);

  async function startLogin() {
    if (tenants.length > 1 && !tenant) {
      setMessage("请先选择机构，再使用钉钉扫码登录。");
      return;
    }
    setPending(true);
    setMessage("");
    try {
      const redirectUri = `${window.location.origin}/login`;
      const intent = await browserDingTalkWebLoginIntent({
        tenantId: tenant?.id ?? tenantId,
        tenantCode: tenant?.code ?? tenantCode,
        redirectUri,
        next: nextPath,
      });
      setAuthUrl(intent.authUrl);
      window.location.href = intent.authUrl;
    } catch (error) {
      const rawMessage = error instanceof Error ? error.message : "";
      if (rawMessage.includes("tenant is required")) {
        setMessage("请先选择机构，再使用钉钉扫码登录。");
      } else {
        setMessage(rawMessage || "无法创建钉钉扫码登录");
      }
    } finally {
      setPending(false);
    }
  }

  return (
    <div className="space-y-3">
      {tenants.length > 1 ? (
        <label className="block space-y-2">
          <span className="text-sm font-medium text-slate-900">钉钉登录机构</span>
          <select className="h-10 w-full rounded-md border bg-white px-3 text-sm" onChange={(event) => setSelectedTenantId(event.currentTarget.value)} value={selectedTenantId}>
            <option value="">请选择机构</option>
            {tenants.map((item) => (
              <option key={item.id} value={item.id}>
                {item.name}
              </option>
            ))}
          </select>
        </label>
      ) : null}
      <Button className="w-full" disabled={pending || Boolean(authCode) || (tenants.length > 1 && !tenant)} onClick={startLogin} type="button" variant="outline">
        {pending ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" /> : <QrCode className="h-4 w-4" aria-hidden="true" />}
        {pending ? "正在打开钉钉" : "钉钉扫码登录"}
      </Button>
      {authUrl ? (
        <Button asChild className="w-full" variant="ghost">
          <Link href={authUrl}>
            <ExternalLink className="h-4 w-4" aria-hidden="true" />
            打开钉钉授权页
          </Link>
        </Button>
      ) : null}
      {message ? <p className="rounded-md bg-slate-100 p-3 text-sm text-slate-700">{message}</p> : null}
    </div>
  );
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
