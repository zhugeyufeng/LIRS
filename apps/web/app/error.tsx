"use client";

import { Button } from "@/components/ui/button";

export default function ErrorPage({ reset }: { error: Error; reset: () => void }) {
  return (
    <div className="flex min-h-screen items-center justify-center bg-slate-100 p-6">
      <div className="max-w-md rounded-lg border bg-white p-6 shadow-sm">
        <h1 className="text-lg font-bold">页面加载失败</h1>
        <p className="mt-2 text-sm text-slate-600">请求处理失败，请稍后重试或返回上一页。</p>
        <Button className="mt-4" onClick={reset}>
          重试
        </Button>
      </div>
    </div>
  );
}
