"use client";

import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";
import { Mail, MessageCircle, Pencil, Save } from "lucide-react";
import {
  browserPatch,
  NotificationChannelSettings,
  SMTPSettings,
  SMTPSettingsPayload,
  WeChatSettings,
  WeChatSettingsPayload,
} from "@/lib/api";
import { AdminDialog } from "@/components/admin-dialog";
import { Button } from "@/components/ui/button";
import { PasswordInput } from "@/components/ui/password-input";

export function NotificationChannelSettingsForm({ settings }: { settings: NotificationChannelSettings }) {
  const router = useRouter();
  const [currentSettings, setCurrentSettings] = useState(settings);
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState("");

  async function saveSMTP(event: FormEvent<HTMLFormElement>, close?: () => void) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const payload: SMTPSettingsPayload = {
      enabled: form.get("enabled") === "on",
      host: String(form.get("host") ?? ""),
      port: Number(form.get("port") ?? 587),
      username: String(form.get("username") ?? ""),
      password: String(form.get("password") ?? ""),
      fromEmail: String(form.get("fromEmail") ?? ""),
      fromName: String(form.get("fromName") ?? ""),
    };
    await save("smtp", "/api/notification-channel-settings/smtp", payload, close);
  }

  async function saveWeChat(event: FormEvent<HTMLFormElement>, close?: () => void) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const payload: WeChatSettingsPayload = {
      enabled: form.get("enabled") === "on",
      accountType: String(form.get("accountType") ?? "service_account"),
      appId: String(form.get("appId") ?? ""),
      appSecret: String(form.get("appSecret") ?? ""),
      serviceAccountName: String(form.get("serviceAccountName") ?? ""),
      templateId: String(form.get("templateId") ?? ""),
      token: String(form.get("token") ?? ""),
      encodingAesKey: String(form.get("encodingAesKey") ?? ""),
    };
    await save("wechat", "/api/notification-channel-settings/wechat", payload, close);
  }

  async function save(key: string, path: string, payload: unknown, close?: () => void) {
    setPending(key);
    setMessage("");
    try {
      const updated = await browserPatch<SMTPSettings | WeChatSettings>(path, payload);
      setCurrentSettings((current) =>
        key === "smtp"
          ? { ...current, smtp: updated as SMTPSettings }
          : { ...current, wechat: updated as WeChatSettings },
      );
      setMessage("通知通道设置已保存。");
      close?.();
      router.refresh();
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "保存失败");
    } finally {
      setPending("");
    }
  }

  return (
    <div className="space-y-5">
      <div className="rounded-lg border bg-white p-4">
        <div className="flex flex-col justify-between gap-4 sm:flex-row sm:items-start">
          <div className="min-w-0">
            <div className="flex items-center gap-2">
              <Mail className="h-5 w-5 shrink-0 text-primary" />
              <h2 className="font-bold text-slate-900">SMTP 邮箱验证码</h2>
            </div>
            <div className="mt-3 grid gap-2 text-sm text-slate-600 sm:grid-cols-2">
              <Summary label="状态" value={currentSettings.smtp.enabled ? "启用" : "停用"} />
              <Summary label="服务器" value={currentSettings.smtp.host || "未配置"} />
              <Summary label="端口" value={String(currentSettings.smtp.port || 587)} />
              <Summary label="发件邮箱" value={currentSettings.smtp.fromEmail || "未配置"} />
              <Summary label="密码" value={currentSettings.smtp.passwordConfigured ? "已配置" : "未配置"} />
            </div>
          </div>
          <AdminDialog
            description="SMTP 用于注册验证码发送，密码留空时保持原配置不变。"
            title="修改 SMTP 邮箱验证码"
            trigger={
              <Button className="w-full sm:w-auto" variant="outline">
                <Pencil className="h-4 w-4" />
                修改
              </Button>
            }
          >
            {(close) => (
              <form className="space-y-4" onSubmit={(event) => saveSMTP(event, close)}>
                <SMTPFields settings={currentSettings.smtp} />
                <div className="flex justify-end">
                  <Button disabled={pending === "smtp"} type="submit">
                    <Save className="h-4 w-4" />
                    {pending === "smtp" ? "保存中" : "保存 SMTP"}
                  </Button>
                </div>
              </form>
            )}
          </AdminDialog>
        </div>
      </div>

      <div className="rounded-lg border bg-white p-4">
        <div className="flex flex-col justify-between gap-4 sm:flex-row sm:items-start">
          <div className="min-w-0">
            <div className="flex items-center gap-2">
              <MessageCircle className="h-5 w-5 shrink-0 text-primary" />
              <h2 className="font-bold text-slate-900">微信公众号 / 服务号预留接口</h2>
            </div>
            <div className="mt-3 grid gap-2 text-sm text-slate-600 sm:grid-cols-2">
              <Summary label="状态" value={currentSettings.wechat.enabled ? "启用" : "停用"} />
              <Summary label="账号类型" value={currentSettings.wechat.accountType === "official_account" ? "公众号" : "服务号"} />
              <Summary label="账号名称" value={currentSettings.wechat.serviceAccountName || "未配置"} />
              <Summary label="AppID" value={currentSettings.wechat.appId || "未配置"} />
              <Summary label="AppSecret" value={currentSettings.wechat.appSecretConfigured ? "已配置" : "未配置"} />
            </div>
          </div>
          <AdminDialog
            description="这里预留微信公众号和服务号接口参数，实际发送通道可后续接入。"
            title="修改微信通知通道"
            trigger={
              <Button className="w-full sm:w-auto" variant="outline">
                <Pencil className="h-4 w-4" />
                修改
              </Button>
            }
          >
            {(close) => (
              <form className="space-y-4" onSubmit={(event) => saveWeChat(event, close)}>
                <WeChatFields settings={currentSettings.wechat} />
                <div className="flex justify-end">
                  <Button disabled={pending === "wechat"} type="submit">
                    <Save className="h-4 w-4" />
                    {pending === "wechat" ? "保存中" : "保存微信接口"}
                  </Button>
                </div>
              </form>
            )}
          </AdminDialog>
        </div>
      </div>

      {message ? <p className="rounded-md bg-slate-100 p-3 text-sm text-slate-700">{message}</p> : null}
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

function SMTPFields({ settings }: { settings: SMTPSettings }) {
  return (
    <div className="grid gap-4 md:grid-cols-2">
      <label className="block space-y-2 md:col-span-2">
        <span className="flex h-10 items-center gap-3 rounded-md border bg-white px-3 text-sm">
          <input className="h-4 w-4" defaultChecked={settings.enabled} name="enabled" type="checkbox" />
          启用 SMTP 邮箱验证码
        </span>
        <FieldHint value={`当前：${settings.enabled ? "启用" : "停用"}`} />
      </label>
      <Field defaultValue={settings.host} label="SMTP Host" name="host" placeholder="smtp.example.com" />
      <Field defaultValue={String(settings.port || 587)} label="端口" name="port" placeholder="587" type="number" />
      <Field defaultValue={settings.username} label="用户名" name="username" placeholder="smtp user" />
      <Field hint={`当前：${settings.passwordConfigured ? "已配置，留空保持不变" : "未配置"}`} label={settings.passwordConfigured ? "密码（留空保持不变）" : "密码"} name="password" placeholder="smtp password" type="password" />
      <Field defaultValue={settings.fromEmail} label="发件邮箱" name="fromEmail" placeholder="noreply@example.com" type="email" />
      <Field defaultValue={settings.fromName} label="发件名称" name="fromName" placeholder="实验室运营系统" />
    </div>
  );
}

function WeChatFields({ settings }: { settings: WeChatSettings }) {
  return (
    <div className="grid gap-4 md:grid-cols-2">
      <label className="block space-y-2 md:col-span-2">
        <span className="flex h-10 items-center gap-3 rounded-md border bg-white px-3 text-sm">
          <input className="h-4 w-4" defaultChecked={settings.enabled} name="enabled" type="checkbox" />
          启用微信通知通道预留配置
        </span>
        <FieldHint value={`当前：${settings.enabled ? "启用" : "停用"}`} />
      </label>
      <label className="block space-y-2">
        <span className="text-sm font-medium">账号类型</span>
        <select className="h-10 w-full rounded-md border bg-white px-3 text-sm" defaultValue={settings.accountType || "service_account"} name="accountType">
          <option value="service_account">服务号</option>
          <option value="official_account">公众号</option>
        </select>
        <FieldHint value={`当前：${settings.accountType === "official_account" ? "公众号" : "服务号"}`} />
      </label>
      <Field defaultValue={settings.serviceAccountName} label="账号名称" name="serviceAccountName" placeholder="单位服务号名称" />
      <Field defaultValue={settings.appId} label="AppID" name="appId" placeholder="wx..." />
      <Field hint={`当前：${settings.appSecretConfigured ? "已配置，留空保持不变" : "未配置"}`} label={settings.appSecretConfigured ? "AppSecret（留空保持不变）" : "AppSecret"} name="appSecret" placeholder="AppSecret" type="password" />
      <Field defaultValue={settings.templateId} label="模板 ID" name="templateId" placeholder="模板消息 ID" />
      <Field defaultValue={settings.token} label="Token" name="token" placeholder="服务器配置 Token" />
      <Field defaultValue={settings.encodingAesKey} label="EncodingAESKey" name="encodingAesKey" placeholder="消息加解密密钥" />
    </div>
  );
}

function Field({
  defaultValue = "",
  hint,
  label,
  name,
  placeholder,
  type = "text",
}: {
  defaultValue?: string;
  hint?: string;
  label: string;
  name: string;
  placeholder: string;
  type?: string;
}) {
  return (
    <label className="block space-y-2">
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
  const text = value.trim();
  return text === "" ? "未设置" : text;
}
