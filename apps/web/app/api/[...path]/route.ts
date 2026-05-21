import { NextRequest } from "next/server";

import { authTokenKey } from "@/lib/auth-cookie";
import { clearBusinessDataCache } from "@/lib/business-data-cache";

const apiBaseUrl = process.env.API_BASE_URL ?? "http://localhost:8090";
const proxyTimeoutMs = Number(process.env.API_PROXY_TIMEOUT_MS ?? 60000);
const maxProxyBodyBytes = Number(process.env.MAX_PROXY_BODY_BYTES ?? 10 * 1024 * 1024);

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
  const bodyResult = await readProxyBody(request, method);
  if (bodyResult instanceof Response) {
    return bodyResult;
  }
  const body = bodyResult;
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
  const allowedHeaders = new Set(["accept", "accept-language", "content-type", "cookie", "user-agent", "x-request-id"]);
  const headers = new Headers();
  for (const [key, value] of request.headers.entries()) {
    if (allowedHeaders.has(key.toLowerCase())) {
      headers.set(key, value);
    }
  }
  const token = request.cookies.get(authTokenKey)?.value;
  if (token) {
    headers.set("authorization", `Bearer ${token}`);
  }
  return headers;
}

async function readProxyBody(request: NextRequest, method: string) {
  if (["GET", "HEAD"].includes(method)) {
    return undefined;
  }
  if (contentLengthExceeds(request.headers.get("content-length"), maxProxyBodyBytes)) {
    return Response.json({ error: "请求体超过限制，请上传 10MB 以内的文件" }, { status: 413 });
  }
  try {
    return await readLimitedRequestBody(request, maxProxyBodyBytes);
  } catch (error) {
    if (error instanceof ProxyBodyTooLargeError) {
      return Response.json({ error: "请求体超过限制，请上传 10MB 以内的文件" }, { status: 413 });
    }
    throw error;
  }
}

async function readLimitedRequestBody(request: NextRequest, maxBytes: number) {
  if (!request.body) {
    return undefined;
  }
  const reader = request.body.getReader();
  const chunks: Uint8Array[] = [];
  let received = 0;
  try {
    for (;;) {
      const { done, value } = await reader.read();
      if (done) {
        break;
      }
      received += value.byteLength;
      if (received > maxBytes) {
        throw new ProxyBodyTooLargeError();
      }
      chunks.push(value);
    }
  } finally {
    reader.releaseLock();
  }
  if (received === 0) {
    return undefined;
  }
  return Buffer.concat(chunks.map((chunk) => Buffer.from(chunk)), received);
}

class ProxyBodyTooLargeError extends Error {}

function contentLengthExceeds(value: string | null, maxBytes: number) {
  if (!value) {
    return false;
  }
  const contentLength = Number(value);
  return Number.isFinite(contentLength) && contentLength > maxBytes;
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
