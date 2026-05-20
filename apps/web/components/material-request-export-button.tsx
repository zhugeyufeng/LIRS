"use client";

import { Download } from "lucide-react";
import { useState } from "react";
import { Button } from "@/components/ui/button";

function currentMonth() {
  return new Date().toISOString().slice(0, 7);
}

export function MaterialRequestExportButton() {
  const [month, setMonth] = useState(currentMonth());
  const [message, setMessage] = useState("");

  async function download() {
    setMessage("");
    const exportMonth = month.trim() || currentMonth();
    try {
      const query = new URLSearchParams({ month: exportMonth });
      const response = await fetch(`/api/material-requests/monthly-export.xlsx?${query.toString()}`, { credentials: "include" });
      if (!response.ok) {
        setMessage(`导出失败：${response.status}`);
        return;
      }
      const blob = await response.blob();
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = `${exportMonth.replace("-", "")}标准物质领用记录表.xlsx`;
      link.click();
      URL.revokeObjectURL(url);
    } catch {
      setMessage("导出失败，请稍后重试");
    }
  }

  return (
    <div className="flex min-w-0 flex-col gap-2 sm:flex-row sm:items-center">
      <label className="sr-only" htmlFor="material-request-export-month">
        导出月份
      </label>
      <input
        className="h-10 min-w-0 rounded-md border bg-white px-3 text-sm sm:h-8 sm:w-36"
        id="material-request-export-month"
        onChange={(event) => setMonth(event.target.value)}
        type="month"
        value={month}
      />
      <Button className="h-10 w-full sm:h-8 sm:w-auto" onClick={download} size="sm" type="button" variant="outline">
        <Download className="h-4 w-4" aria-hidden="true" />
        按月导出领用记录
      </Button>
      {message ? <p className="text-xs text-destructive sm:ml-1">{message}</p> : null}
    </div>
  );
}
