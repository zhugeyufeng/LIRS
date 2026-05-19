import { NextRequest } from "next/server";

import { clearBusinessDataCache } from "@/lib/business-data-cache";

const apiBaseUrl = process.env.API_BASE_URL ?? "http://localhost:8090";
const proxyTimeoutMs = Number(process.env.API_PROXY_TIMEOUT_MS ?? 60000);
const authTokenKey = "lirs.authToken";

export const dynamic = "force-dynamic";

export async function GET(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  return proxy(request, context);
}

export async function POST(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  return proxy(request, context);
}

export async function PATCH(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  return proxy(request, context);
}

export async function PUT(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  return proxy(request, context);
}

export async function DELETE(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  return proxy(request, context);
}

async function proxy(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  const { path } = await context.params;
  const joinedPath = path.join("/");
  const upstreamUrl = new URL(`/api/${path.join("/")}${request.nextUrl.search}`, apiBaseUrl);
  const headers = forwardRequestHeaders(request);

  const method = request.method;
  const body = ["GET", "HEAD"].includes(method) ? undefined : Buffer.from(await request.arrayBuffer());
  let response: Response;
  try {
    response = await fetchWithTimeout(upstreamUrl, {
      method,
      headers,
      body,
      cache: "no-store",
    }, proxyTimeoutMs);
  } catch (error) {
    const message = error instanceof Error && error.name === "AbortError" ? "后端接口超时，请稍后重试或联系管理员查看导入日志" : "后端接口不可用";
    return Response.json({ error: message }, { status: 504 });
  }

  const responseHeaders = new Headers(response.headers);
  responseHeaders.delete("content-encoding");
  responseHeaders.delete("content-length");
  if (shouldInvalidateBusinessData(method, response)) {
    clearBusinessDataCache();
  }
  if (method === "POST" && (joinedPath === "login" || joinedPath === "dingtalk/quick-login" || joinedPath === "dingtalk/login-bind-existing") && response.ok) {
    const text = await response.text();
    const auth = parseJson<{ token?: string }>(text);
    if (auth?.token) {
      responseHeaders.append("set-cookie", authCookie(auth.token, request));
    }
    return new Response(text, {
      status: response.status,
      headers: responseHeaders,
    });
  }
  if (method === "POST" && joinedPath === "dingtalk/web-login" && response.ok) {
    const text = await response.text();
    const result = parseJson<{ auth?: { token?: string } }>(text);
    if (result?.auth?.token) {
      responseHeaders.append("set-cookie", authCookie(result.auth.token, request));
    }
    return new Response(text, {
      status: response.status,
      headers: responseHeaders,
    });
  }
  if (method === "POST" && (joinedPath === "logout" || joinedPath === "logout-all")) {
    responseHeaders.append("set-cookie", clearAuthCookie(request));
  }
  return new Response(response.body, {
    status: response.status,
    headers: responseHeaders,
  });
}

function shouldInvalidateBusinessData(method: string, response: Response) {
  return !["GET", "HEAD"].includes(method) && response.ok;
}

function parseJson<T>(text: string): T | null {
  try {
    return JSON.parse(text) as T;
  } catch {
    return null;
  }
}

function forwardRequestHeaders(request: NextRequest) {
  const allowedHeaders = new Set(["accept", "accept-language", "authorization", "content-type", "cookie", "user-agent", "x-request-id"]);
  const headers = new Headers();
  for (const [key, value] of request.headers.entries()) {
    if (allowedHeaders.has(key.toLowerCase())) {
      headers.set(key, value);
    }
  }
  if (!headers.has("authorization")) {
    const token = request.cookies.get(authTokenKey)?.value;
    if (token) {
      headers.set("authorization", `Bearer ${token}`);
    }
  }
  return headers;
}

async function fetchWithTimeout(input: URL, init: RequestInit, timeoutMs: number) {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), timeoutMs);
  try {
    return await fetch(input, { ...init, signal: controller.signal });
  } finally {
    clearTimeout(timeout);
  }
}

function authCookie(token: string, request: NextRequest) {
  return `${authTokenKey}=${token}; Path=/; SameSite=Lax; HttpOnly${secureCookieSuffix(request)}; Max-Age=604800`;
}

function clearAuthCookie(request: NextRequest) {
  return `${authTokenKey}=; Path=/; SameSite=Lax; HttpOnly${secureCookieSuffix(request)}; Max-Age=0`;
}

function secureCookieSuffix(request: NextRequest) {
  const forwardedProto = request.headers.get("x-forwarded-proto");
  return request.nextUrl.protocol === "https:" || forwardedProto === "https" ? "; Secure" : "";
}
