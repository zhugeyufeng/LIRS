"use client";

import { FormEvent, startTransition, useEffect, useState } from "react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { Bot, Pencil, Save } from "lucide-react";
import { AIAssistantSettings, AIAssistantSettingsPayload, aiAssistantProviderDefaults, browserPatch, Tenant, User } from "@/lib/api";
import { confirmTwice } from "@/lib/confirm";
import { AdminDialog } from "@/components/admin-dialog";
import { Button } from "@/components/ui/button";
import { PasswordInput } from "@/components/ui/password-input";

export function AIAssistantSettingsForm({
  currentUser,
  selectedTenant,
  settings,
  tenants,
}: {
  currentUser: User;
  selectedTenant: Tenant;
  settings: AIAssistantSettings;
  tenants: Tenant[];
}) {
  const router = useRouter();
  const isSuperAdmin = currentUser.role === "super_admin";
  const [currentSettings, setCurrentSettings] = useState(settings);
  const [selectedProvider, setSelectedProvider] = useState(settings.provider || "openai_compatible");
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);

  useEffect(() => {
    setCurrentSettings(settings);
    setSelectedProvider(settings.provider || "openai_compatible");
    setMessage("");
  }, [settings, selectedTenant.id]);

  async function submit(event: FormEvent<HTMLFormElement>, close?: () => void) {
    event.preventDefault();
    if (!confirmTwice(`确定修改“${selectedTenant.name}”的 AI 模型设置吗？`, "请再次确认。保存后该机构 AI 助手会立即按新模型 API 运行。")) {
      return;
    }
    const form = new FormData(event.currentTarget);
    const payload: AIAssistantSettingsPayload = {
      enabled: form.get("enabled") === "on",
      provider: String(form.get("provider") ?? "openai_compatible"),
      baseUrl: String(form.get("baseUrl") ?? ""),
      apiKey: String(form.get("apiKey") ?? ""),
      model: String(form.get("model") ?? ""),
      systemPrompt: String(form.get("systemPrompt") ?? ""),
      temperature: Number(form.get("temperature") ?? 0.2),
      maxTokens: Number(form.get("maxTokens") ?? 1200),
    };
    setPending(true);
    setMessage("");
    try {
      const updated = await browserPatch<AIAssistantSettings>(tenantScopedPath("/api/ai-assistant-settings", selectedTenant.id), payload);
      setCurrentSettings(updated);
      setMessage(`${selectedTenant.name} 的 AI 模型设置已保存，更新时间 ${formatUpdatedAt(updated.updatedAt, updated.updatedBy)}。`);
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
      {isSuperAdmin ? <TenantSelector selectedTenant={selectedTenant} tenants={tenants} /> : <TenantNotice selectedTenant={selectedTenant} />}
      <div className="rounded-lg border bg-white p-4">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div className="min-w-0 space-y-4">
            <div>
              <div className="flex items-center gap-2">
                <Bot className="h-5 w-5 text-primary" aria-hidden="true" />
                <h2 className="font-bold text-slate-900">{selectedTenant.name} 的 AI 助手模型</h2>
              </div>
              <p className="mt-2 break-words text-sm leading-6 text-slate-600">AI 助手必须启用并配置 API Key、API 地址和模型后才会回答用户提问；API Key 只保存不回显。</p>
            </div>
            <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
              <Summary label="机构" value={selectedTenant.name} />
              <Summary label="状态" value={currentSettings.enabled ? "启用" : "停用"} />
              <Summary label="供应商" value={providerLabel(currentSettings.provider)} />
              <Summary label="API 地址" value={currentSettings.baseUrl || "未配置"} />
              <Summary label="模型" value={currentSettings.model || "未配置"} />
              <Summary label="API Key" value={currentSettings.apiKeyConfigured ? "已配置" : "未配置"} />
              <Summary label="温度" value={String(currentSettings.temperature)} />
              <Summary label="最大输出" value={`${currentSettings.maxTokens} token`} />
              <Summary label="最近更新" value={formatUpdatedAt(currentSettings.updatedAt, currentSettings.updatedBy)} />
            </div>
          </div>
          <AdminDialog
            description="支持 OpenAI 兼容接口和 DeepSeek；切换供应商时会带出默认 API 地址和模型。"
            maxWidth="max-w-4xl"
            title="修改 AI 模型设置"
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
                      启用 AI 助手真实模型调用
                    </span>
                    <FieldHint value={`当前：${currentSettings.enabled ? "启用" : "停用"}`} />
                  </label>
                  <label className="block space-y-2">
                    <span className="text-sm font-medium">供应商</span>
                    <select className="h-10 w-full rounded-md border bg-white px-3 text-sm" name="provider" onChange={(event) => setSelectedProvider(event.currentTarget.value)} value={selectedProvider}>
                      <option value="openai_compatible">OpenAI 兼容接口</option>
                      <option value="deepseek">DeepSeek</option>
                    </select>
                    <FieldHint value="当前版本使用 /chat/completions 兼容协议。" />
                  </label>
                  <Field defaultValue={defaultBaseUrl(currentSettings, selectedProvider)} key={`${selectedProvider}:baseUrl`} label="API 地址" name="baseUrl" placeholder={apiAddressPlaceholder(selectedProvider)} type="url" />
                  <Field defaultValue={defaultModel(currentSettings, selectedProvider)} key={`${selectedProvider}:model`} label="模型名称" name="model" placeholder={modelPlaceholder(selectedProvider)} />
                  <Field
                    hint={currentSettings.apiKeyConfigured ? "当前：已配置，留空保持不变" : "当前：未配置"}
                    label={currentSettings.apiKeyConfigured ? "API Key（留空保持不变）" : "API Key"}
                    name="apiKey"
                    placeholder="sk-..."
                    type="password"
                  />
                  <Field defaultValue={currentSettings.temperature} label="温度" max={2} min={0} name="temperature" placeholder="0.2" step="0.1" type="number" />
                  <Field defaultValue={currentSettings.maxTokens} label="最大输出 token" max={8000} min={1} name="maxTokens" placeholder="1200" step="1" type="number" />
                  <label className="block space-y-2 md:col-span-2">
                    <span className="text-sm font-medium">系统提示词</span>
                    <textarea className="min-h-32 w-full rounded-md border bg-white px-3 py-2 text-sm" defaultValue={currentSettings.systemPrompt} name="systemPrompt" />
                    <FieldHint value="模型会同时收到当前机构的预约、资源、培训、样本、空间和物联网统计上下文。" />
                  </label>
                </div>
                <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                  {message ? <p className="rounded-md bg-slate-100 px-3 py-2 text-sm text-slate-700">{message}</p> : <span />}
                  <Button className="w-full sm:w-auto" disabled={pending} type="submit">
                    <Save className="h-4 w-4" aria-hidden="true" />
                    {pending ? "保存中" : "保存 AI 模型设置"}
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

function TenantSelector({ selectedTenant, tenants }: { selectedTenant: Tenant; tenants: Tenant[] }) {
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
      <label className="block min-w-0 space-y-2">
        <span className="text-sm font-medium text-slate-900">选择机构</span>
        <select className="h-10 w-full rounded-md border bg-white px-3 text-sm" onChange={(event) => changeTenant(event.currentTarget.value)} value={selectedTenant.id}>
          {tenants.map((tenant) => (
            <option key={tenant.id} value={tenant.id}>
              {tenant.name}
            </option>
          ))}
        </select>
      </label>
    </div>
  );
}

function TenantNotice({ selectedTenant }: { selectedTenant: Tenant }) {
  return (
    <div className="rounded-lg border bg-slate-50/60 p-4">
      <p className="text-sm font-medium text-slate-900">当前机构：{selectedTenant.name}</p>
    </div>
  );
}

function tenantScopedPath(path: string, tenantId: string) {
  const query = new URLSearchParams();
  query.set("tenantId", tenantId);
  return `${path}?${query.toString()}`;
}

function providerLabel(value: string) {
  if (value === "deepseek") {
    return "DeepSeek";
  }
  return value === "openai_compatible" ? "OpenAI 兼容接口" : value || "未配置";
}

function defaultBaseUrl(settings: AIAssistantSettings, provider: string) {
  if ((settings.provider || "openai_compatible") === provider && settings.baseUrl) {
    return settings.baseUrl;
  }
  return aiAssistantProviderDefaults(provider).baseUrl;
}

function defaultModel(settings: AIAssistantSettings, provider: string) {
  if ((settings.provider || "openai_compatible") === provider && settings.model) {
    return settings.model;
  }
  return aiAssistantProviderDefaults(provider).model;
}

function apiAddressPlaceholder(provider: string) {
  return aiAssistantProviderDefaults(provider).baseUrl;
}

function modelPlaceholder(provider: string) {
  return aiAssistantProviderDefaults(provider).model;
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
  max,
  min,
  name,
  placeholder,
  step,
  type = "text",
}: {
  defaultValue?: string | number;
  hint?: string;
  label: string;
  max?: number;
  min?: number;
  name: string;
  placeholder?: string;
  step?: string;
  type?: string;
}) {
  return (
    <label className="block space-y-2">
      <span className="text-sm font-medium">{label}</span>
      {type === "password" ? (
        <PasswordInput className="h-10 w-full rounded-md border bg-white px-3 text-sm" name={name} placeholder={placeholder} />
      ) : (
        <input className="h-10 w-full rounded-md border bg-white px-3 text-sm" defaultValue={defaultValue} max={max} min={min} name={name} placeholder={placeholder} step={step} type={type} />
      )}
      {hint ? <FieldHint value={hint} /> : null}
    </label>
  );
}

function FieldHint({ value }: { value: string }) {
  return <span className="block break-words text-xs leading-5 text-slate-500">{value}</span>;
}

function formatUpdatedAt(value: string, by?: string) {
  if (!value) {
    return "尚未保存";
  }
  const formatted = new Date(value).toLocaleString("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    timeZone: "Asia/Shanghai",
  });
  return by ? `${formatted} / ${by}` : formatted;
}
