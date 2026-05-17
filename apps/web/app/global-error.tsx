"use client";

import { Button } from "@/components/ui/button";

export default function GlobalError({ reset }: { error: Error; reset: () => void }) {
  return (
    <html lang="zh-CN">
      <body>
        <div className="flex min-h-screen items-center justify-center bg-slate-100 p-6">
          <div className="max-w-md rounded-lg border bg-white p-6 shadow-sm">
            <h1 className="text-lg font-bold">系统加载失败</h1>
            <p className="mt-2 text-sm text-slate-600">应用启动或页面渲染失败，请稍后重试。</p>
            <Button className="mt-4" onClick={reset}>
              重试
            </Button>
          </div>
        </div>
      </body>
    </html>
  );
}
