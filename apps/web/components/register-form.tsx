"use client";

import Link from "next/link";
import { FormEvent, useEffect, useState } from "react";
import { MailCheck, ShieldCheck } from "lucide-react";
import {
  browserPost,
  EmailVerificationCodePayload,
  EmailVerificationCodeResponse,
  OrganizationUnit,
  RegisterPayload,
  Tenant,
  User,
} from "@/lib/api";
import { Button } from "@/components/ui/button";
import { PasswordInput } from "@/components/ui/password-input";

export function RegisterForm({ departments, tenants }: { departments: string[]; tenants: Tenant[] }) {
  const [message, setMessage] = useState<string>("");
  const [pending, setPending] = useState(false);
  const [codePending, setCodePending] = useState(false);
  const [institutionKey, setInstitutionKey] = useState(tenants[0]?.code || tenants[0]?.id || "");
  const [departmentOptions, setDepartmentOptions] = useState(departments);
  const [departmentsPending, setDepartmentsPending] = useState(false);
  const selectedTenant = findTenant(tenants, institutionKey);
  const selectedTenantId = selectedTenant?.id ?? "";

  useEffect(() => {
    if (!selectedTenantId) {
      setDepartmentOptions([]);
      return;
    }
    const controller = new AbortController();
    setDepartmentsPending(true);
    fetch(`/api/organization-units?kind=department&tenantId=${encodeURIComponent(selectedTenantId)}`, {
      signal: controller.signal,
    })
      .then((response) => (response.ok ? response.json() : []))
      .then((items: OrganizationUnit[]) => {
        setDepartmentOptions(items.map((item) => item.name));
      })
      .catch((error) => {
        if ((error as Error).name !== "AbortError") {
          setDepartmentOptions([]);
        }
      })
      .finally(() => {
        if (!controller.signal.aborted) {
          setDepartmentsPending(false);
        }
      });
    return () => controller.abort();
  }, [selectedTenantId]);

  async function sendCode(event: FormEvent<HTMLButtonElement>) {
    event.preventDefault();
    const form = event.currentTarget.form;
    if (!form) {
      return;
    }
    const formData = new FormData(form);
    const tenant = findTenant(tenants, institutionKey);
    if (!tenant) {
      setMessage("请填写有效的机构 ID 或机构编码。");
      return;
    }
    const payload: EmailVerificationCodePayload = {
      tenantId: tenant.id,
      tenantCode: tenant.code,
      email: String(formData.get("email") ?? ""),
    };
    setCodePending(true);
    setMessage("");
    try {
      const result = await browserPost<EmailVerificationCodeResponse>("/api/email-verification-codes", payload);
      setMessage(result.message);
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "验证码发送失败");
    } finally {
      setCodePending(false);
    }
  }

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setMessage("");
    const formElement = event.currentTarget;
    const form = new FormData(formElement);
    const tenant = findTenant(tenants, institutionKey);
    if (!tenant) {
      setMessage("请填写有效的机构 ID 或机构编码。");
      setPending(false);
      return;
    }
    const payload: RegisterPayload = {
      tenantId: tenant.id,
      tenantCode: tenant.code,
      accountType: "user",
      name: String(form.get("name") ?? ""),
      phone: String(form.get("phone") ?? ""),
      email: String(form.get("email") ?? ""),
      password: String(form.get("password") ?? ""),
      department: String(form.get("department") ?? ""),
      verificationCode: String(form.get("verificationCode") ?? ""),
    };
    try {
      const user = await browserPost<User>("/api/register", payload);
      setMessage(`${user.name} 已提交注册申请。请使用注册邮箱登录，再进入个人设置的钉钉绑定页面扫码授权。`);
      formElement.reset();
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "注册失败");
    } finally {
      setPending(false);
    }
  }

  return (
    <form className="space-y-4" onSubmit={submit}>
      <label className="block space-y-2">
        <span className="text-sm font-medium">机构 ID / 机构编码</span>
        <input
          className="h-10 w-full rounded-md border bg-white px-3 text-sm outline-none focus:ring-2 focus:ring-primary"
          list="tenant-register-options"
          name="institutionKey"
          onChange={(event) => setInstitutionKey(event.currentTarget.value)}
          placeholder="输入租户管理中的机构 ID 或编码"
          required
          value={institutionKey}
        />
        <datalist id="tenant-register-options">
          {tenants.map((tenant) => (
            <option key={tenant.id} value={tenant.code}>
              {tenant.name} / {tenant.id}
            </option>
          ))}
        </datalist>
        <span className="text-xs text-slate-500">
          {selectedTenant ? `当前机构：${selectedTenant.name}，机构 ID：${selectedTenant.id}` : "请使用租户管理中显示的机构 ID 或机构编码。"}
        </span>
      </label>
      <div className="grid gap-4 md:grid-cols-2">
        <Field label="姓名" name="name" placeholder="真实姓名" />
        <Field label="手机号" name="phone" placeholder="13800000000" />
      </div>
      <div className="grid gap-3 sm:grid-cols-[1fr_auto] sm:items-end">
        <Field label="邮箱" name="email" placeholder="name@university.edu.cn" type="email" />
        <Button className="w-full sm:w-auto" disabled={codePending} onClick={sendCode} type="button" variant="outline">
          <MailCheck className="h-4 w-4" />
          {codePending ? "发送中" : "发送验证码"}
        </Button>
      </div>
      <Field label="邮箱验证码" name="verificationCode" placeholder="6 位验证码" />
      <Field label="密码" name="password" placeholder="至少 8 位" type="password" />
      <label className="block space-y-2">
        <span className="text-sm font-medium">所属部门/实验室</span>
        <select className="h-10 w-full rounded-md border bg-white px-3 text-sm" name="department" required>
          <option value="">{departmentsPending ? "正在加载部门" : "请选择部门"}</option>
          {departmentOptions.map((department) => (
            <option key={department} value={department}>
              {department}
            </option>
          ))}
        </select>
        {departmentOptions.length === 0 && !departmentsPending ? <span className="text-xs text-slate-500">该单位尚未配置部门，请联系管理员先维护组织架构管理。</span> : null}
      </label>
      <div className="flex gap-2 rounded-lg border border-blue-100 bg-blue-50 p-3 text-sm text-blue-800">
        <ShieldCheck className="mt-0.5 h-4 w-4 shrink-0" />
        <p>前台仅支持普通用户注册。机构管理员必须由系统超级管理员在管理后台人员管理中设置。</p>
      </div>
      <Button className="w-full" disabled={pending} type="submit">
        {pending ? "提交中..." : "提交注册"}
      </Button>
      {message ? (
        <div className="space-y-2 rounded-md bg-slate-100 p-3 text-sm text-slate-700">
          <p>{message}</p>
          {message.includes("已提交注册申请") ? (
            <Link className="font-medium text-primary underline-offset-4 hover:underline" href="/login">
              前往登录后绑定钉钉
            </Link>
          ) : null}
        </div>
      ) : null}
    </form>
  );
}

function findTenant(tenants: Tenant[], key: string) {
  const normalized = key.trim().toLowerCase();
  if (!normalized) {
    return undefined;
  }
  return tenants.find((tenant) => tenant.id.toLowerCase() === normalized || tenant.code.toLowerCase() === normalized);
}

function Field({
  label,
  name,
  placeholder,
  type = "text",
}: {
  label: string;
  name: string;
  placeholder: string;
  type?: string;
}) {
  return (
    <label className="block space-y-2">
      <span className="text-sm font-medium">{label}</span>
      {type === "password" ? (
        <PasswordInput minLength={8} name={name} placeholder={placeholder} required />
      ) : (
        <input className="h-10 w-full rounded-md border bg-white px-3 text-sm outline-none focus:ring-2 focus:ring-primary" name={name} placeholder={placeholder} required type={type} />
      )}
    </label>
  );
}
