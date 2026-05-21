"use client";

import { FormEvent, useState } from "react";
import { Bot, ChevronDown, MessageSquareText, Send, X } from "lucide-react";
import { AssistantQuery, browserPost, type User } from "@/lib/api";
import { Button } from "@/components/ui/button";

type ChatMessage = {
  id: string;
  role: "user" | "assistant";
  content: string;
};

export function FloatingAssistant({ currentUser }: { currentUser: User }) {
  const [open, setOpen] = useState(false);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const formElement = event.currentTarget;
    const form = new FormData(formElement);
    const question = String(form.get("question") ?? "").trim();
    if (!question || pending) {
      return;
    }
    setMessages((current) => [...current, { id: `user-${Date.now()}`, role: "user", content: question }]);
    setPending(true);
    setError("");
    formElement.reset();
    try {
      const result = await browserPost<AssistantQuery>("/api/ai-assistant", { question });
      setMessages((current) => [...current, { id: result.id, role: "assistant", content: result.answer }]);
    } catch (submitError) {
      setError(submitError instanceof Error ? submitError.message : "AI 助手暂时不可用");
    } finally {
      setPending(false);
    }
  }

  if (!open) {
    return (
      <div className="fixed bottom-4 right-4 z-[80] sm:bottom-6 sm:right-6">
        <Button className="h-12 rounded-full px-4 shadow-lg shadow-slate-900/20" onClick={() => setOpen(true)} type="button">
          <Bot className="h-5 w-5" aria-hidden="true" />
          <span className="hidden sm:inline">AI 助手</span>
        </Button>
      </div>
    );
  }

  return (
    <section className="fixed bottom-4 right-4 z-[80] flex max-h-[calc(100dvh-2rem)] w-[calc(100vw-2rem)] max-w-md flex-col overflow-hidden rounded-lg border bg-white shadow-2xl shadow-slate-900/20 sm:bottom-6 sm:right-6" aria-label="AI 助手">
      <div className="flex items-start justify-between gap-3 border-b bg-slate-950 px-4 py-3 text-white">
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <Bot className="h-5 w-5 shrink-0" aria-hidden="true" />
            <h2 className="font-bold">AI 助手</h2>
          </div>
          <p className="mt-1 break-words text-xs text-slate-300">{currentUser.tenantName} / {currentUser.name}</p>
        </div>
        <div className="flex shrink-0 items-center gap-1">
          <Button className="h-8 w-8 text-white hover:bg-white/10" onClick={() => setOpen(false)} size="icon" type="button" variant="ghost">
            <ChevronDown className="h-4 w-4" aria-hidden="true" />
            <span className="sr-only">收起 AI 助手</span>
          </Button>
          <Button className="h-8 w-8 text-white hover:bg-white/10" onClick={() => setMessages([])} size="icon" type="button" variant="ghost">
            <X className="h-4 w-4" aria-hidden="true" />
            <span className="sr-only">清空当前对话</span>
          </Button>
        </div>
      </div>

      <div className="min-h-0 flex-1 space-y-3 overflow-y-auto bg-slate-50 p-3">
        {messages.length === 0 ? (
          <div className="rounded-md border border-dashed bg-white p-4 text-sm leading-6 text-slate-600">
            <MessageSquareText className="mb-2 h-5 w-5 text-primary" aria-hidden="true" />
            可直接询问预约、资源、培训、样本、空间和物联网设备相关问题。
          </div>
        ) : null}
        {messages.map((message) => (
          <div className={`flex ${message.role === "user" ? "justify-end" : "justify-start"}`} key={message.id}>
            <div className={`max-w-[88%] rounded-lg px-3 py-2 text-sm leading-6 ${message.role === "user" ? "bg-primary text-white" : "border bg-white text-slate-700"}`}>{message.content}</div>
          </div>
        ))}
        {pending ? <p className="rounded-md bg-white px-3 py-2 text-sm text-slate-500">正在生成回答...</p> : null}
        {error ? <p className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">{error}</p> : null}
      </div>

      <form className="border-t bg-white p-3" onSubmit={submit}>
        <div className="flex gap-2">
          <input className="h-10 min-w-0 flex-1 rounded-md border bg-white px-3 text-sm" name="question" placeholder="输入问题..." />
          <Button disabled={pending} size="icon" type="submit">
            <Send className="h-4 w-4" aria-hidden="true" />
            <span className="sr-only">发送问题</span>
          </Button>
        </div>
      </form>
    </section>
  );
}
