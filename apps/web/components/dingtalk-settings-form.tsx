"use client";

import { FormEvent, startTransition, useEffect, useState } from "react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { Eye, EyeOff, Pencil, Save } from "lucide-react";
import { browserPatch, DingTalkSettings, DingTalkSettingsPayload, Tenant, User } from "@/lib/api";
import { AdminDialog } from "@/components/admin-dialog";
import { Button } from "@/components/ui/button";
import { PasswordInput } from "@/components/ui/password-input";

export function DingTalkSettingsForm({
  currentUser,
  origin,
  selectedTenant,
  settings,
  tenants,
}: {
  currentUser: User;
  origin: string;
  selectedTenant: Tenant;
  settings: DingTalkSettings;
  tenants: Tenant[];
}) {
  const router = useRouter();
  const isSuperAdmin = currentUser.role === "super_admin";
  const [currentSettings, setCurrentSettings] = useState(settings);
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);
  const eventCallbackUrl = tenantEventCallbackURL(origin, selectedTenant.code || selectedTenant.id);

  useEffect(() => {
    setCurrentSettings(settings);
    setMessage("");
  }, [settings, selectedTenant.id]);

  async function submit(event: FormEvent<HTMLFormElement>, close?: () => void) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const payload: DingTalkSettingsPayload = {
      schemaVersion: 2,
      enabled: form.get("enabled") === "on",
      clientId: String(form.get("clientId") ?? ""),
      clientSecret: String(form.get("clientSecret") ?? ""),
      corpId: String(form.get("corpId") ?? ""),
      robotCode: String(form.get("robotCode") ?? ""),
      oauthRedirectUri: String(form.get("oauthRedirectUri") ?? ""),
      eventCallbackUrl: String(form.get("eventCallbackUrl") ?? ""),
      eventAesKey: String(form.get("eventAesKey") ?? ""),
      eventToken: String(form.get("eventToken") ?? ""),
    };
    setPending(true);
    setMessage("");
    try {
      const updated = await browserPatch<DingTalkSettings>(tenantScopedPath("/api/notification-channel-settings/dingtalk", selectedTenant.id), payload);
      setCurrentSettings(updated);
      setMessage(`${selectedTenant.name} 的钉钉应用设置已保存，更新时间 ${formatUpdatedAt(updated.updatedAt, updated.updatedBy)}。`);
      close?.();
      startTransition(() => {
        router.refresh();
      });
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "保存失败");
    } finally {
      setPending(false);
    }
  }

  return (
    <div className="space-y-5">
      {isSuperAdmin ? (
        <TenantSelector origin={origin} selectedTenant={selectedTenant} tenants={tenants} />
      ) : (
        <div className="rounded-lg border bg-slate-50/60 p-4">
          <p className="text-sm font-medium text-slate-900">当前机构：{selectedTenant.name}</p>
          <p className="mt-1 break-words text-xs text-slate-500">事件订阅 URL 使用本机构独立地址：{eventCallbackUrl}</p>
        </div>
      )}
      <div className="rounded-lg border bg-white p-4">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div className="min-w-0 space-y-4">
            <div>
              <h2 className="font-bold text-slate-900">{selectedTenant.name} 的钉钉企业应用</h2>
              <p className="mt-2 break-words text-sm leading-6 text-slate-600">每个机构独立保存新版 Client ID、Client Secret、Corp ID、机器人编码和 HTTP 事件订阅配置。</p>
            </div>
            <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
              <Summary label="机构编码" value={selectedTenant.code || selectedTenant.id} />
              <Summary label="状态" value={currentSettings.enabled ? "启用" : "停用"} />
              <Summary label="配置版本" value={`v${currentSettings.schemaVersion || 2}`} />
              <Summary label="Client ID" value={currentSettings.clientId || "未配置"} />
              <SecretSummary configured={currentSettings.clientSecretConfigured} label="Client Secret" secret={currentSettings.clientSecret ?? ""} />
              <Summary label="Corp ID" value={currentSettings.corpId || "未配置"} />
              <Summary label="机器人编码" value={currentSettings.robotCode || "未配置"} />
              <Summary label="扫码绑定回调地址" value={currentSettings.oauthRedirectUri || "未配置"} />
              <Summary label="事件订阅 URL" value={currentSettings.eventCallbackUrl || eventCallbackUrl} />
              <SecretSummary configured={currentSettings.eventAesKeyConfigured} label="事件 AES Key" secret={currentSettings.eventAesKey ?? ""} />
              <SecretSummary configured={currentSettings.eventTokenConfigured} label="事件 Token" secret={currentSettings.eventToken ?? ""} />
              <Summary label="最近更新" value={formatUpdatedAt(currentSettings.updatedAt, currentSettings.updatedBy)} />
            </div>
          </div>
          <AdminDialog
            description={`密钥、事件 AES Key 和事件 Token 留空时保持原配置不变；保存后只影响 ${selectedTenant.name}。`}
            maxWidth="max-w-4xl"
            title="修改钉钉企业应用"
            trigger={
              <Button className="w-full lg:w-auto" variant="outline">
                <Pencil className="h-4 w-4" aria-hidden="true" />
                修改
              </Button>
            }
          >
            {(close) => (
              <form className="space-y-5" onSubmit={(event) => submit(event, close)}>
                <div className="grid gap-4 md:grid-cols-2">
                  <label className="block space-y-2 md:col-span-2">
                    <span className="flex h-10 items-center gap-3 rounded-md border bg-white px-3 text-sm">
                      <input className="h-4 w-4" defaultChecked={currentSettings.enabled} name="enabled" type="checkbox" />
                      启用钉钉工作通知
                    </span>
                    <FieldHint value={`当前：${currentSettings.enabled ? "启用" : "停用"}`} />
                  </label>
                  <Field defaultValue={currentSettings.clientId} label="Client ID" name="clientId" placeholder="钉钉企业应用 Client ID" />
                  <Field
                    defaultValue=""
                    hint={`当前：${currentSettings.clientSecretConfigured ? "已配置，留空保持不变" : "未配置"}`}
                    label={currentSettings.clientSecretConfigured ? "Client Secret（留空保持不变）" : "Client Secret"}
                    name="clientSecret"
                    placeholder="钉钉企业应用 Client Secret"
                    type="password"
                  />
                  <Field defaultValue={currentSettings.corpId} label="Corp ID" name="corpId" placeholder="ding..." />
                  <Field defaultValue={currentSettings.robotCode} label="机器人编码" name="robotCode" placeholder="钉钉企业应用机器人编码" />
                  <Field className="md:col-span-2" defaultValue={currentSettings.oauthRedirectUri} label="扫码绑定回调地址" name="oauthRedirectUri" placeholder="https://example.com/settings/dingtalk" type="url" />
                  <Field className="md:col-span-2" defaultValue={currentSettings.eventCallbackUrl || eventCallbackUrl} label="事件订阅 URL" name="eventCallbackUrl" placeholder={eventCallbackUrl} type="url" />
                  <Field
                    defaultValue=""
                    hint={`当前：${currentSettings.eventAesKeyConfigured ? "已配置，留空保持不变" : "未配置"}`}
                    label={currentSettings.eventAesKeyConfigured ? "事件 AES Key（留空保持不变）" : "事件 AES Key"}
                    name="eventAesKey"
                    placeholder="钉钉事件订阅 EncodingAESKey"
                    type="password"
                  />
                  <Field
                    defaultValue=""
                    hint={`当前：${currentSettings.eventTokenConfigured ? "已配置，留空保持不变" : "未配置"}`}
                    label={currentSettings.eventTokenConfigured ? "事件 Token（留空保持不变）" : "事件 Token"}
                    name="eventToken"
                    placeholder="钉钉事件订阅 Token"
                    type="password"
                  />
                </div>
                <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                  {message ? <p className="rounded-md bg-slate-100 px-3 py-2 text-sm text-slate-700">{message}</p> : <span />}
                  <Button className="w-full sm:w-auto" disabled={pending} type="submit">
                    <Save className="h-4 w-4" aria-hidden="true" />
                    {pending ? "保存中" : "保存钉钉设置"}
                  </Button>
                </div>
              </form>
            )}
          </AdminDialog>
        </div>
      </div>
      {message ? <p className="rounded-md bg-slate-100 px-3 py-2 text-sm text-slate-700">{message}</p> : null}
    </div>
  );
}

function TenantSelector({ origin, selectedTenant, tenants }: { origin: string; selectedTenant: Tenant; tenants: Tenant[] }) {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();

  function changeTenant(value: string) {
    const next = new URLSearchParams(searchParams.toString());
    next.set("tenantId", value);
    router.push(`${pathname}?${next.toString()}`);
  }

  return (
    <div className="rounded-lg border bg-slate-50/60 p-4">
      <div className="flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
        <label className="block min-w-0 flex-1 space-y-2">
          <span className="text-sm font-medium text-slate-900">选择机构</span>
          <select className="h-10 w-full rounded-md border bg-white px-3 text-sm" onChange={(event) => changeTenant(event.currentTarget.value)} value={selectedTenant.id}>
            {tenants.map((tenant) => (
              <option key={tenant.id} value={tenant.id}>
                {tenant.name}
              </option>
            ))}
          </select>
        </label>
        <p className="break-words text-xs text-slate-500 md:max-w-lg md:text-right">当前事件订阅 URL：{tenantEventCallbackURL(origin, selectedTenant.code || selectedTenant.id)}</p>
      </div>
    </div>
  );
}

function tenantEventCallbackURL(origin: string, tenantCode: string) {
  return `${origin}/api/dingtalk/events/${encodeURIComponent(tenantCode)}`;
}

function tenantScopedPath(path: string, tenantId: string) {
  const query = new URLSearchParams();
  query.set("tenantId", tenantId);
  return `${path}?${query.toString()}`;
}

function Summary({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0 rounded-md bg-slate-50 px-3 py-2">
      <p className="text-xs text-slate-500">{label}</p>
      <p className="mt-1 break-words font-medium text-slate-800">{value}</p>
    </div>
  );
}

function SecretSummary({ configured, label, secret }: { configured: boolean; label: string; secret: string }) {
  const [visible, setVisible] = useState(false);
  const value = configured ? (visible && secret ? secret : "已配置") : "未配置";

  return (
    <div className="min-w-0 rounded-md bg-slate-50 px-3 py-2">
      <div className="flex items-center justify-between gap-3">
        <p className="text-xs text-slate-500">{label}</p>
        {configured && secret ? (
          <button className="inline-flex h-7 w-7 items-center justify-center rounded-md text-slate-500 hover:bg-white hover:text-slate-900" onClick={() => setVisible((current) => !current)} type="button">
            {visible ? <EyeOff className="h-4 w-4" aria-hidden="true" /> : <Eye className="h-4 w-4" aria-hidden="true" />}
            <span className="sr-only">{visible ? "隐藏密钥" : "显示密钥"}</span>
          </button>
        ) : null}
      </div>
      <p className="mt-1 break-all font-medium text-slate-800">{value}</p>
    </div>
  );
}

function Field({
  className = "",
  defaultValue = "",
  hint,
  label,
  name,
  placeholder,
  type = "text",
}: {
  className?: string;
  defaultValue?: string;
  hint?: string;
  label: string;
  name: string;
  placeholder: string;
  type?: string;
}) {
  return (
    <label className={`block space-y-2 ${className}`}>
      <span className="text-sm font-medium">{label}</span>
      {type === "password" ? (
        <PasswordInput defaultValue={defaultValue} name={name} placeholder={placeholder} />
      ) : (
        <input className="h-10 w-full rounded-md border bg-white px-3 text-sm" defaultValue={defaultValue} name={name} placeholder={placeholder} type={type} />
      )}
      <FieldHint value={hint ?? `当前：${formatCurrent(defaultValue)}`} />
    </label>
  );
}

function FieldHint({ value }: { value: string }) {
  return <span className="block break-words text-xs text-slate-500">{value}</span>;
}

function formatCurrent(value: string) {
  const trimmed = value.trim();
  return trimmed || "未配置";
}

function formatDateTime(value: string) {
  return new Date(value).toLocaleString("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    timeZone: "Asia/Shanghai",
  });
}

function formatUpdatedAt(value: string, updatedBy: string) {
  if (!value || value.startsWith("0001-01-01")) {
    return "尚未保存";
  }
  return `${formatDateTime(value)}${updatedBy ? ` · ${updatedBy}` : ""}`;
}
