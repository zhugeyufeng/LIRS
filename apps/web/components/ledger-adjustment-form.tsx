"use client";

import { FormEvent, startTransition, useState } from "react";
import { useRouter } from "next/navigation";
import { PlusCircle, Save } from "lucide-react";
import { browserPost, LedgerEntry, User } from "@/lib/api";
import { AdminDialog } from "@/components/admin-dialog";
import { Button } from "@/components/ui/button";

export function LedgerAdjustmentForm({ users }: { users: User[] }) {
  const router = useRouter();
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>, close?: () => void) {
    event.preventDefault();
    setPending(true);
    setMessage("");
    const formElement = event.currentTarget;
    const form = new FormData(formElement);
    try {
      const entry = await browserPost<LedgerEntry>("/api/ledger/adjustments", {
        originalEntryId: String(form.get("originalEntryId") ?? ""),
        userId: String(form.get("userId") ?? ""),
        amount: Number(form.get("amount") ?? 0),
        reason: String(form.get("reason") ?? ""),
      });
      setMessage(`调整流水已生成：¥${entry.amount.toFixed(2)}`);
      formElement.reset();
      close?.();
      startTransition(() => {
        router.refresh();
      });
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "提交失败");
    } finally {
      setPending(false);
    }
  }

  return (
    <div className="space-y-2">
      <AdminDialog
        description="财务纠错通过新增调整流水完成，历史流水不会被修改或删除。"
        title="生成账务调整"
        trigger={
          <Button className="w-full" disabled={pending || users.length === 0}>
            <PlusCircle className="h-4 w-4" aria-hidden="true" />
            生成调整流水
          </Button>
        }
      >
        {(close) => (
          <form className="space-y-4" onSubmit={(event) => submit(event, close)}>
            <label className="block space-y-2">
              <span className="text-sm font-medium">调整人员</span>
              <select className="h-10 w-full rounded-md border bg-white px-3 text-sm" name="userId" required>
                <option value="">选择人员</option>
                {users.map((user) => (
                  <option key={user.id} value={user.id}>
                    {user.name} / {user.department}
                  </option>
                ))}
              </select>
              <span className="block text-xs text-slate-500">调整流水会写入所选人员的个人财务账户。</span>
            </label>
            <Field label="原始流水 ID" name="originalEntryId" placeholder="可选，用于关联被纠错的流水" />
            <Field label="调整金额" name="amount" placeholder="填写调整金额，可为负数" required type="number" />
            <Field label="调整原因" name="reason" placeholder="填写本次调整原因" required />
            <div className="flex justify-end">
              <Button className="w-full sm:w-auto" disabled={pending} type="submit">
                <Save className="h-4 w-4" aria-hidden="true" />
                {pending ? "提交中..." : "生成调整流水"}
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
  label,
  name,
  placeholder,
  required = false,
  type = "text",
}: {
  label: string;
  name: string;
  placeholder: string;
  required?: boolean;
  type?: string;
}) {
  return (
    <label className="block space-y-2">
      <span className="text-sm font-medium">{label}</span>
      <input className="h-10 w-full rounded-md border bg-white px-3 text-sm" name={name} placeholder={placeholder} required={required} type={type} />
    </label>
  );
}
