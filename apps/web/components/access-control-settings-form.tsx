"use client";

import { FormEvent, startTransition, useState } from "react";
import { useRouter } from "next/navigation";
import { Pencil, Save, ShieldCheck } from "lucide-react";
import { AccessControlSettings, AccessControlSettingsPayload, browserPatch } from "@/lib/api";
import { AdminDialog } from "@/components/admin-dialog";
import { Button } from "@/components/ui/button";
import { PasswordInput } from "@/components/ui/password-input";

export function AccessControlSettingsForm({ settings }: { settings: AccessControlSettings }) {
  const router = useRouter();
  const [currentSettings, setCurrentSettings] = useState(settings);
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>, close?: () => void) {
    event.preventDefault();
    setPending(true);
    setMessage("");

    const form = new FormData(event.currentTarget);
    const payload: AccessControlSettingsPayload = {
      enabled: form.get("enabled") === "on",
      vendor: String(form.get("vendor") ?? "hikvision") as AccessControlSettingsPayload["vendor"],
      endpoint: String(form.get("endpoint") ?? ""),
      clientId: String(form.get("clientId") ?? ""),
      clientSecret: String(form.get("clientSecret") ?? ""),
      accessGroup: String(form.get("accessGroup") ?? ""),
      autoGrantOnApproval: form.get("autoGrantOnApproval") === "on",
      autoRevokeOnCompletion: form.get("autoRevokeOnCompletion") === "on",
    };

    try {
      const updated = await browserPatch<AccessControlSettings>("/api/access-control-settings", payload);
      setCurrentSettings(updated);
      setMessage(`已保存，更新时间 ${formatDateTime(updated.updatedAt)}`);
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
      <div className="rounded-lg border bg-white p-4">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div className="min-w-0 space-y-4">
            <div>
              <div className="flex items-center gap-2">
                <ShieldCheck className="h-5 w-5 text-primary" aria-hidden="true" />
                <h2 className="font-bold text-slate-900">门禁权限对接</h2>
              </div>
              <p className="mt-2 break-words text-sm leading-6 text-slate-600">
                预留大华、海康威视门禁平台接入参数；每台仪器在仪器管理里单独启用并绑定授权组或点位后，审批通过才会自动下发授权。
              </p>
            </div>
            <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
              <Summary label="状态" value={currentSettings.enabled ? "启用" : "停用"} />
              <Summary label="厂商" value={vendorLabel(currentSettings.vendor)} />
              <Summary label="对接地址" value={currentSettings.endpoint || "未配置"} />
              <Summary label="默认授权组" value={currentSettings.accessGroup || "未配置"} />
              <Summary label="自动下发" value={currentSettings.autoGrantOnApproval ? "开启" : "关闭"} />
              <Summary label="自动回收" value={currentSettings.autoRevokeOnCompletion ? "开启" : "关闭"} />
              <Summary label="密钥" value={currentSettings.clientSecretConfigured ? "已配置" : "未配置"} />
              <Summary label="最近更新" value={formatUpdatedAt(currentSettings.updatedAt, currentSettings.updatedBy)} />
            </div>
          </div>
          <AdminDialog
            description="修改门禁接入参数后，已在仪器管理中启用门禁联动的仪器会在审批通过与预约完成时向事件流发出授权或回收事件。"
            maxWidth="max-w-4xl"
            title="修改门禁权限对接"
            trigger={
              <Button className="w-full lg:w-auto" variant="outline">
                <Pencil className="h-4 w-4" aria-hidden="true" />
                修改
              </Button>
            }
          >
            {(close) => (
              <form className="space-y-6" onSubmit={(event) => submit(event, close)}>
                <div className="grid gap-4 md:grid-cols-2">
                  <label className="block space-y-2 md:col-span-2">
                    <span className="flex h-10 items-center gap-3 rounded-md border bg-white px-3 text-sm">
                      <input className="h-4 w-4" defaultChecked={currentSettings.enabled} name="enabled" type="checkbox" />
                      启用门禁权限对接
                    </span>
                    <FieldHint value={`当前：${currentSettings.enabled ? "启用" : "停用"}`} />
                  </label>
                  <label className="block space-y-2">
                    <span className="text-sm font-medium">厂商</span>
                    <select className="h-10 w-full rounded-md border bg-white px-3 text-sm" defaultValue={currentSettings.vendor} name="vendor">
                      <option value="hikvision">海康威视</option>
                      <option value="dahua">大华</option>
                      <option value="custom">自定义</option>
                    </select>
                    <FieldHint value={`当前：${vendorLabel(currentSettings.vendor)}`} />
                  </label>
                  <Field defaultValue={currentSettings.endpoint} label="对接地址" name="endpoint" placeholder="https://door-api.example.com" />
                  <Field defaultValue={currentSettings.clientId} label="客户端 ID" name="clientId" placeholder="access-control-app" />
                  <Field
                    hint={currentSettings.clientSecretConfigured ? "当前：已配置，留空保持不变" : "当前：未配置"}
                    label={currentSettings.clientSecretConfigured ? "客户端密钥（留空保持不变）" : "客户端密钥"}
                    name="clientSecret"
                    placeholder="client secret"
                    type="password"
                  />
                  <Field defaultValue={currentSettings.accessGroup} label="默认门禁授权组（可选）" name="accessGroup" placeholder="A 楼实验室门禁组" />
                  <label className="block space-y-2 md:col-span-2">
                    <span className="flex h-10 items-center gap-3 rounded-md border bg-white px-3 text-sm">
                      <input className="h-4 w-4" defaultChecked={currentSettings.autoGrantOnApproval} name="autoGrantOnApproval" type="checkbox" />
                      审批通过后自动下发门禁权限
                    </span>
                    <FieldHint value={`当前：${currentSettings.autoGrantOnApproval ? "开启" : "关闭"}`} />
                  </label>
                  <label className="block space-y-2 md:col-span-2">
                    <span className="flex h-10 items-center gap-3 rounded-md border bg-white px-3 text-sm">
                      <input className="h-4 w-4" defaultChecked={currentSettings.autoRevokeOnCompletion} name="autoRevokeOnCompletion" type="checkbox" />
                      预约完成或取消后自动回收门禁权限
                    </span>
                    <FieldHint value={`当前：${currentSettings.autoRevokeOnCompletion ? "开启" : "关闭"}`} />
                  </label>
                </div>

                <div className="flex justify-end">
                  <Button disabled={pending} type="submit">
                    <Save className="h-4 w-4" aria-hidden="true" />
                    {pending ? "保存中..." : "保存门禁权限"}
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

function Summary({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0 rounded-md bg-slate-50 px-3 py-2">
      <p className="text-xs text-slate-500">{label}</p>
      <p className="mt-1 break-words font-medium text-slate-800">{value}</p>
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

function vendorLabel(vendor: AccessControlSettings["vendor"]) {
  const labels: Record<AccessControlSettings["vendor"], string> = {
    hikvision: "海康威视",
    dahua: "大华",
    custom: "自定义",
  };
  return labels[vendor];
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
