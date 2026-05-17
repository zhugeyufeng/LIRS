"use client";

import { FormEvent, startTransition, useState } from "react";
import { useRouter } from "next/navigation";
import { CreditCard, Pencil, Save } from "lucide-react";
import { browserPatch, browserPost, FinancialAccount, FinancialAccountPayload, User } from "@/lib/api";
import { AdminDialog } from "@/components/admin-dialog";
import { Button } from "@/components/ui/button";

export function FinancialAccountForm({ account, users = [] }: { account?: FinancialAccount; users?: User[] }) {
  const router = useRouter();
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>, close?: () => void) {
    event.preventDefault();
    setPending(true);
    setMessage("");
    const formElement = event.currentTarget;
    const form = new FormData(formElement);
    const payload: FinancialAccountPayload = {
      userId: String(form.get("userId") ?? account?.userId ?? ""),
      creditLimit: Number(form.get("creditLimit") ?? 0),
      initialBalance: Number(form.get("initialBalance") ?? 0),
    };
    try {
      const updated = account?.id
        ? await browserPatch<FinancialAccount>(`/api/financial-accounts/${account.id}`, payload)
        : await browserPost<FinancialAccount>("/api/financial-accounts", payload);
      setMessage(`账户已保存：${updated.userName}`);
      if (!account) {
        formElement.reset();
      }
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
    <div className="space-y-2">
      <AdminDialog
        description={account ? "修改个人财务账户授信额度。" : "为启用账号创建个人财务账户。"}
        title={account ? `修改账户：${account.userName}` : "创建个人账户"}
        trigger={
          <Button className="w-full sm:w-auto" disabled={pending || (!account && users.length === 0)} size={account ? "sm" : "default"} variant={account ? "outline" : "default"}>
            {account ? <Pencil className="h-4 w-4" aria-hidden="true" /> : <CreditCard className="h-4 w-4" aria-hidden="true" />}
            {account ? "修改授信" : "创建账户"}
          </Button>
        }
      >
        {(close) => (
          <form className="space-y-4" onSubmit={(event) => submit(event, close)}>
            <label className="block space-y-2">
              <span className="text-sm font-medium">账户人员</span>
              <select className="h-10 w-full rounded-md border bg-white px-3 text-sm" defaultValue={account?.userId ?? ""} disabled={Boolean(account)} name="userId" required>
                <option value="">选择人员</option>
                {mergeUsers(users, account).map((user) => (
                  <option key={user.id} value={user.id}>
                    {user.name} / {user.department}
                  </option>
                ))}
              </select>
              {account ? <FieldHint value={`当前：${account.userName} / ${account.department}`} /> : null}
            </label>
            {account ? <input name="userId" type="hidden" value={account.userId} /> : null}
            <Field defaultValue={account?.creditLimit} label="授信额度" min={0} name="creditLimit" placeholder="填写授信额度" required step="0.01" type="number" />
            {!account ? (
              <Field label="初始余额" name="initialBalance" placeholder="填写初始余额，可为 0" step="0.01" type="number" />
            ) : null}
            <div className="flex justify-end">
              <Button className="w-full sm:w-auto" disabled={pending} type="submit" variant={account ? "outline" : "default"}>
                <Save className="h-4 w-4" aria-hidden="true" />
                {pending ? "保存中..." : account ? "保存账户" : "创建账户"}
              </Button>
            </div>
          </form>
        )}
      </AdminDialog>
      {message ? <p className="text-xs text-slate-500">{message}</p> : null}
    </div>
  );
}

function Field({
  defaultValue,
  label,
  min,
  name,
  placeholder,
  required = false,
  step,
  type = "text",
}: {
  defaultValue?: string | number;
  label: string;
  min?: number;
  name: string;
  placeholder: string;
  required?: boolean;
  step?: string;
  type?: string;
}) {
  return (
    <label className="block space-y-2">
      <span className="text-sm font-medium">{label}</span>
      <input className="h-10 w-full rounded-md border bg-white px-3 text-sm" defaultValue={defaultValue ?? ""} min={min} name={name} placeholder={placeholder} required={required} step={step} type={type} />
      {defaultValue !== undefined ? <FieldHint value={`当前：${formatCurrent(defaultValue)}`} /> : null}
    </label>
  );
}

function FieldHint({ value }: { value: string }) {
  return <span className="block break-words text-xs text-slate-500">{value}</span>;
}

function formatCurrent(value: string | number) {
  const text = String(value).trim();
  return text === "" ? "未设置" : text;
}

function mergeUsers(users: User[], account?: FinancialAccount) {
  if (!account?.userId || users.some((user) => user.id === account.userId)) {
    return users;
  }
  return [
    {
      id: account.userId,
      tenantId: "",
      tenantName: "",
      tenantCode: "",
      name: account.userName,
      email: "",
      phone: "",
      department: account.department,
      groupName: account.groupName,
      role: "",
      status: "active",
      emailVerified: true,
      financeEnabled: true,
      authEpoch: 0,
    },
    ...users,
  ];
}
