"use client";

import { startTransition, useState } from "react";
import { useRouter } from "next/navigation";
import { browserPost, MaterialPurchaseMonthlyConfirmation } from "@/lib/api";
import { confirmTwice } from "@/lib/confirm";
import { Button } from "@/components/ui/button";

export function MaterialPurchaseMonthConfirmButton({ month }: { month?: string }) {
  const router = useRouter();
  const [value, setValue] = useState(month ?? new Date().toISOString().slice(0, 7));
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);

  async function confirmMonth() {
    if (!confirmTwice(`确定确认 ${value} 的申购汇总吗？`, "请再次确认。确认后该月所有申购不能退回、取消或修改重新提交。")) {
      return;
    }
    setPending(true);
    setMessage("");
    try {
      await browserPost<MaterialPurchaseMonthlyConfirmation>("/api/material-purchases/monthly-confirmations", { month: value });
      setMessage(`${value} 申购汇总已确认。`);
      startTransition(() => router.refresh());
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "确认失败");
    } finally {
      setPending(false);
    }
  }

  return (
    <div className="space-y-2">
      <label className="block space-y-2">
        <span className="text-sm font-medium">汇总月份</span>
        <input className="h-10 w-full rounded-md border bg-white px-3 text-sm" onChange={(event) => setValue(event.currentTarget.value)} type="month" value={value} />
      </label>
      <Button className="w-full" disabled={pending || value === ""} onClick={confirmMonth} type="button" variant="outline">
        {pending ? "确认中..." : "确认当月申购汇总"}
      </Button>
      {message ? <p className="text-xs text-slate-500">{message}</p> : null}
    </div>
  );
}
