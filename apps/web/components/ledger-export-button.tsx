"use client";

import { Download } from "lucide-react";
import { useState } from "react";
import { Button } from "@/components/ui/button";

const browserBaseUrl = process.env.NEXT_PUBLIC_API_BASE_URL || "";

export function LedgerExportButton() {
  const [message, setMessage] = useState("");

  async function download() {
    setMessage("");
    try {
      const response = await fetch(`${browserBaseUrl}/api/ledger/export.csv`);
      if (!response.ok) {
        setMessage(`导出失败：${response.status}`);
        return;
      }
      const blob = await response.blob();
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = "lirs-ledger.csv";
      link.click();
      URL.revokeObjectURL(url);
    } catch {
      setMessage("导出失败，请稍后重试");
    }
  }

  return (
    <div className="space-y-1">
      <Button className="h-10 w-full sm:h-8 sm:w-auto" onClick={download} size="sm" type="button" variant="outline">
        <Download className="h-4 w-4" />
        导出流水
      </Button>
      {message ? <p className="text-xs text-destructive">{message}</p> : null}
    </div>
  );
}
