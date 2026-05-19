"use client";

import { Printer, QrCode } from "lucide-react";
import { QRCodeSVG } from "qrcode.react";
import { Button } from "@/components/ui/button";
import { browserMaterialQRCodePrintAdapter } from "@/lib/material-qr-print-adapter";

type MaterialQRCodeCardProps = {
  materialName: string;
  materialSpec: string;
  materialLocation: string;
  qrCode: string;
};

export function MaterialQRCodeCard({ materialName, materialSpec, materialLocation, qrCode }: MaterialQRCodeCardProps) {
  if (!qrCode) {
    return (
      <div className="rounded-lg border border-dashed bg-slate-50 p-4 text-sm text-slate-500">
        <div className="flex items-center gap-2 font-bold text-slate-700">
          <QrCode className="h-4 w-4" aria-hidden="true" />
          二维码
        </div>
        <p className="mt-2 leading-6">资源尚未生成二维码，请在管理中心保存资源后再查看。</p>
      </div>
    );
  }

  function printQRCode() {
    browserMaterialQRCodePrintAdapter.print({
      materialName,
      materialSpec,
      materialLocation,
      qrCode,
    });
  }

  return (
    <div className="space-y-4 rounded-lg border bg-white p-4" data-material-qr-print>
      <div className="flex flex-col justify-between gap-3 sm:flex-row sm:items-start">
        <div className="min-w-0">
          <div className="flex items-center gap-2 font-bold text-slate-900">
            <QrCode className="h-4 w-4 text-primary" aria-hidden="true" />
            资源二维码
          </div>
          <p className="mt-1 break-words text-xs text-slate-500">{qrCode}</p>
        </div>
        <Button className="w-full sm:w-auto print:hidden" onClick={printQRCode} size="sm" type="button" variant="outline">
          <Printer className="h-4 w-4" aria-hidden="true" />
          打印
        </Button>
      </div>

      <div className="flex justify-center">
        <div className="rounded-md border bg-white p-3">
          <QRCodeSVG includeMargin size={196} value={qrCode} />
        </div>
      </div>

      <div className="space-y-1 text-center text-sm">
        <p className="break-words font-bold text-slate-900">{materialName}</p>
        <p className="break-words text-slate-600">{materialSpec || "未登记规格"}</p>
        <p className="break-words text-xs text-slate-500">{materialLocation || "未登记库位"}</p>
      </div>
    </div>
  );
}
