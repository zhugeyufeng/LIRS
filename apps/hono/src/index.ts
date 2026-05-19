import { serve } from "@hono/node-server";
import { Hono, type Context, type Next } from "hono";
import { cors } from "hono/cors";
import { z } from "zod";
import { sql } from "drizzle-orm";
import { bearerToken, currentUserFromToken } from "./auth.js";
import { db } from "./db.js";
import { ensureRedis } from "./redis.js";

const port = Number(process.env.PORT ?? 8090);
const goApiBaseUrl = process.env.GO_API_BASE_URL ?? "http://localhost:8081";
const allowedOrigins = (process.env.ALLOWED_ORIGINS ?? "http://localhost:3000")
  .split(",")
  .map((item) => item.trim())
  .filter(Boolean);
const defaultAllowedOrigin = allowedOrigins[0] ?? "http://localhost:3000";
const maxRequestBodyBytes = Number(process.env.MAX_REQUEST_BODY_BYTES ?? 1024 * 1024);
const proxyTimeoutMs = Number(process.env.PROXY_TIMEOUT_MS ?? 60000);

const app = new Hono();

app.onError((error, c) => {
  console.error("unhandled hono error", error);
  return c.json({ error: "internal server error" }, 500);
});

app.use(
  "*",
  cors({
    origin: (origin) => (origin && allowedOrigins.includes(origin) ? origin : defaultAllowedOrigin),
    allowHeaders: ["Origin", "Content-Type", "Authorization"],
    allowMethods: ["GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"],
    credentials: true,
  }),
);

app.use("*", requestLogger);
app.use("*", requestBodySizeLimit(maxRequestBodyBytes));

app.get("/healthz", async (c) => {
  const checks: Record<string, string> = {};
  try {
    await withTimeout(db.execute(sql`SELECT 1`), 3000, "postgres health check timeout");
    checks.postgres = "ok";
  } catch {
    checks.postgres = "error";
  }
  try {
    const redis = await ensureRedis();
    await withTimeout(redis.ping(), 3000, "redis health check timeout");
    checks.redis = "ok";
  } catch {
    checks.redis = "error";
  }
  const ok = checks.postgres === "ok" && checks.redis === "ok";
  return c.json({ status: ok ? "ok" : "degraded", service: "hono", checks }, ok ? 200 : 503);
});

app.get("/api/me", async (c) => {
  const token = bearerToken(c.req.header("Authorization"));
  if (!token) {
    return c.json({ error: "missing bearer token" }, 401);
  }
  const user = await currentUserFromToken(token);
  if (!user) {
    return c.json({ error: "invalid or expired session" }, 401);
  }
  return c.json(user);
});

app.post("/api/login", rateLimit({ limit: 10, windowMs: 60_000, keyPrefix: "login" }), validateAndProxy(z.object({
  tenantId: z.string().optional().default(""),
  tenantCode: z.string().optional().default(""),
  email: z.string().email(),
  password: z.string().min(1),
  device: z.string().optional().default("web"),
})));

app.post("/api/dingtalk/quick-login", rateLimit({ limit: 20, windowMs: 60_000, keyPrefix: "dingtalk-quick-login" }), validateAndProxy(z.object({
  tenantId: z.string().optional().default(""),
  tenantCode: z.string().optional().default(""),
  authCode: z.string().min(1),
  corpId: z.string().optional().default(""),
  device: z.string().optional().default("dingtalk"),
})));

app.post("/api/register", rateLimit({ limit: 5, windowMs: 60_000, keyPrefix: "register" }), validateAndProxy(z.object({
  tenantId: z.string().optional().default(""),
  tenantCode: z.string().optional().default(""),
  accountType: z.enum(["user"]).optional().default("user"),
  name: z.string().min(1),
  phone: z.string().min(1),
  email: z.string().email(),
  password: z.string().min(8),
  department: z.string().min(1),
  verificationCode: z.string().min(4),
})));

app.post("/api/email-verification-codes", rateLimit({ limit: 3, windowMs: 60_000, keyPrefix: "email-code" }), validateAndProxy(z.object({
  tenantId: z.string().optional().default(""),
  tenantCode: z.string().optional().default(""),
  email: z.string().email(),
})));

app.post("/api/verify-email", validateAndProxy(z.object({
  token: z.string().min(1),
})));

app.patch("/api/me/profile", validateAndProxy(z.object({
  name: z.string().optional(),
  phone: z.string().optional(),
  department: z.string().optional(),
  groupName: z.string().optional(),
})));

app.patch("/api/me/password", validateAndProxy(z.object({
  currentPassword: z.string().min(1),
  newPassword: z.string().min(8),
})));

const emptyJsonObject = z.object({});
const footerSettingsSchema = z.object({
  brandName: z.string().min(1),
  brandTagline: z.string().min(1),
  baseUrl: z.string().optional().default(""),
  description: z.string().min(1),
  copyright: z.string().min(1),
  sections: z.array(z.object({
    title: z.string().min(1),
    lines: z.array(z.string()),
  })),
});
const copyEntrySchema = z.object({
  key: z.string().min(1),
  label: z.string().optional().default(""),
  value: z.string().optional().default(""),
  scope: z.string().optional().default("custom"),
  description: z.string().optional().default(""),
});
const copySettingsSchema = z.object({
  entries: z.array(copyEntrySchema),
});
const tenantSchema = z.object({
  name: z.string().min(1),
  code: z.string().optional().default(""),
  financeEnabled: z.boolean().optional().default(true),
  status: z.enum(["active", "disabled"]).optional().default("active"),
});
const smtpSettingsSchema = z.object({
  enabled: z.boolean().optional().default(false),
  host: z.string().optional().default(""),
  port: z.coerce.number().int().positive().optional().default(587),
  username: z.string().optional().default(""),
  password: z.string().optional(),
  fromEmail: z.string().optional().default(""),
  fromName: z.string().optional().default(""),
});
const wechatSettingsSchema = z.object({
  enabled: z.boolean().optional().default(false),
  accountType: z.enum(["official_account", "service_account"]).optional().default("service_account"),
  appId: z.string().optional().default(""),
  appSecret: z.string().optional(),
  serviceAccountName: z.string().optional().default(""),
  templateId: z.string().optional().default(""),
  token: z.string().optional().default(""),
  encodingAesKey: z.string().optional().default(""),
});
const dingTalkSettingsSchema = z.object({
  schemaVersion: z.coerce.number().int().optional().default(2),
  enabled: z.boolean().optional().default(false),
  clientId: z.string().optional().default(""),
  clientSecret: z.string().optional(),
  corpId: z.string().optional().default(""),
  robotCode: z.string().optional().default(""),
  oauthRedirectUri: z.string().optional().default(""),
  eventCallbackUrl: z.string().optional().default(""),
  eventAesKey: z.string().optional(),
  eventToken: z.string().optional(),
});
const dingTalkTestSchema = z.object({
  userId: z.string().min(1),
});
const dingTalkBindingSchema = z.object({
  authCode: z.string().min(1),
  state: z.string().optional().default(""),
});
const accessControlSettingsSchema = z.object({
  enabled: z.boolean().optional().default(false),
  vendor: z.enum(["hikvision", "dahua", "custom"]).optional().default("hikvision"),
  endpoint: z.string().optional().default(""),
  clientId: z.string().optional().default(""),
  clientSecret: z.string().optional(),
  accessGroup: z.string().optional().default(""),
  autoGrantOnApproval: z.boolean().optional().default(true),
  autoRevokeOnCompletion: z.boolean().optional().default(true),
});
const instrumentSchema = z.object({
  name: z.string().min(1),
  category: z.string().min(1),
  department: z.string().min(1),
  groupName: z.string().optional().default(""),
  status: z.enum(["available", "busy", "maintenance", "disabled"]),
  location: z.string().min(1),
  hourlyRate: z.coerce.number().nonnegative(),
  brand: z.string(),
  model: z.string(),
  assetCode: z.string(),
  accessControlEnabled: z.boolean().optional().default(false),
  accessControlGroup: z.string().optional().default(""),
  accessControlPoint: z.string().optional().default(""),
  description: z.string(),
  technicalSpecs: z.string(),
  bookingRule: z.string(),
  maintenanceSummary: z.string(),
  maxBookingHours: z.coerce.number().int().positive(),
  minAdvanceHours: z.coerce.number().int().nonnegative(),
  cancelCutoffHours: z.coerce.number().int().nonnegative(),
  checkinWindowMinutes: z.coerce.number().int().nonnegative(),
  bookingWindowDays: z.coerce.number().int().positive(),
  bookingIntervalHours: z.coerce.number().int().positive().max(12),
  serviceStartHour: z.coerce.number().int().min(0).max(23).optional().default(0),
  serviceEndHour: z.coerce.number().int().min(1).max(24).optional().default(24),
});
const organizationUnitSchema = z.object({
  kind: z.enum(["department", "group"]),
  name: z.string().min(1),
  parentName: z.string().optional().default(""),
});
const reservationDecisionSchema = z.object({
  comment: z.string().optional().default(""),
});
const reservationCancelSchema = z.object({
  reason: z.string().optional().default(""),
});
const userReviewSchema = z.object({
  tenantId: z.string().optional().default(""),
  role: z.enum(["unassigned", "student", "teacher", "researcher", "group_leader", "material_admin", "finance_admin", "tenant_admin", "lab_admin", "super_admin"]),
  groupName: z.string().optional().default(""),
  department: z.string().min(1),
  email: z.string().email(),
  phone: z.string().min(1),
  status: z.enum(["pending_approval", "active", "disabled"]),
});
const userMembershipSchema = z.object({
  tenantId: z.string().min(1),
  role: z.enum(["unassigned", "student", "teacher", "researcher", "group_leader", "material_admin", "finance_admin", "tenant_admin", "lab_admin"]),
  groupName: z.string().optional().default(""),
  department: z.string().optional().default(""),
  status: z.enum(["pending_approval", "active", "disabled"]),
});
const announcementSchema = z.object({
  title: z.string().min(1),
  body: z.string().min(1),
  level: z.enum(["info", "warning", "success"]),
  targetScope: z.enum(["global", "department", "group", "personal"]),
  target: z.string().optional().default(""),
  userId: z.string().optional().default(""),
  groupName: z.string().optional().default(""),
  department: z.string().optional().default(""),
});
const ledgerAdjustmentSchema = z.object({
  originalEntryId: z.string().optional().default(""),
  userId: z.string().min(1),
  groupName: z.string().optional().default(""),
  amount: z.coerce.number().finite().refine((value) => value !== 0 && Math.abs(value) <= 1_000_000, "amount must be non-zero and within 1000000"),
  reason: z.string().min(1),
});
const financialAccountSchema = z.object({
  userId: z.string().min(1),
  creditLimit: z.coerce.number(),
  initialBalance: z.coerce.number().optional().default(0),
});
const materialSchema = z.object({
  name: z.string().min(1),
  productType: z.enum(["consumable", "reagent", "standard"]).optional().default("consumable"),
  category: z.string().min(1),
  subcategory: z.string().optional().default(""),
  spec: z.string().min(1),
  unit: z.string().min(1),
  unitPrice: z.coerce.number().nonnegative(),
  stock: z.coerce.number().int().nonnegative(),
  warningLine: z.coerce.number().int().nonnegative(),
  supplier: z.string(),
  manufacturer: z.string().optional().default(""),
  batchNo: z.string(),
  catalogNo: z.string().optional().default(""),
  casNo: z.string().optional().default(""),
  grade: z.string().optional().default(""),
  concentration: z.string().optional().default(""),
  parentMaterialId: z.string().optional().default(""),
  dilutionFactor: z.string().optional().default(""),
  preparationMethod: z.string().optional().default(""),
  storageCondition: z.string().optional().default(""),
  storageRoom: z.string().optional().default(""),
  storageCabinet: z.string().optional().default(""),
  storageLayer: z.string().optional().default(""),
  storageSlot: z.string().optional().default(""),
  tenderContract: z.string().optional().default(""),
  contractNo: z.string().optional().default(""),
  remark: z.string().optional().default(""),
  certificateUrl: z.string().optional().default(""),
  standardCertificateUrl: z.string().optional().default(""),
  attachmentUrl: z.string().optional().default(""),
  qrCode: z.string().optional().default(""),
  expiresAt: z.string().optional().default("").refine((value) => value.trim() === "" || !Number.isNaN(Date.parse(value)), "expiresAt must be a valid date"),
  openedAt: z.string().optional().default("").refine((value) => value.trim() === "" || !Number.isNaN(Date.parse(value)), "openedAt must be a valid date"),
  openExpireDays: z.coerce.number().int().nonnegative().optional().default(0),
  freezeThawCount: z.coerce.number().int().nonnegative().optional().default(0),
  freezeThawLimit: z.coerce.number().int().nonnegative().optional().default(0),
  approvalRequired: z.boolean().optional().default(false),
  nearExpiryDays: z.coerce.number().int().nonnegative().optional().default(30),
  status: z.enum(["normal", "near_expiry", "low", "expired", "open_expired", "freeze_thaw_exceeded", "damaged", "disabled"]),
});
const stockAdjustmentSchema = z.object({
  changeQty: z.coerce.number().int(),
  reason: z.string().min(1),
  batchId: z.string().optional().default(""),
  batchNo: z.string().optional().default(""),
  unitId: z.string().optional().default(""),
  expiresAt: z.string().optional().default("").refine((value) => value.trim() === "" || !Number.isNaN(Date.parse(value)), "expiresAt must be a valid date"),
  location: z.string().optional().default(""),
});
const materialRequestSchema = z.object({
  materialId: z.string().min(1),
  requesterId: z.string().optional().default(""),
  requester: z.string().optional().default(""),
  batchId: z.string().optional().default(""),
  unitId: z.string().min(1),
  quantity: z.coerce.number().int().refine((value) => value === 1, "quantity must be 1"),
  purpose: z.string().min(1),
});
const materialPurchaseSchema = z.object({
  materialId: z.string().optional().default(""),
  purchasableMaterialId: z.string().optional().default(""),
  requesterId: z.string().optional().default(""),
  requester: z.string().optional().default(""),
  quantity: z.coerce.number().int().positive(),
  estimatedUnitPrice: z.coerce.number().nonnegative(),
  supplier: z.string().optional().default(""),
  reason: z.string().min(1),
});
const procurementProjectSchema = z.object({
  name: z.string().min(1),
  expiresAt: z.string().optional().default(""),
  status: z.enum(["active", "disabled"]).optional().default("active"),
});
const materialDamageSchema = z.object({
  materialId: z.string().min(1),
  reporterId: z.string().optional().default(""),
  reporter: z.string().optional().default(""),
  unitId: z.string().min(1),
  quantity: z.coerce.number().int().refine((value) => value === 1, "quantity must be 1"),
  reason: z.string().min(1),
  photoUrl: z.string().optional().default(""),
  attachmentUrl: z.string().optional().default(""),
});
const materialCategorySchema = z.object({
  name: z.string().min(1),
  parentName: z.string().optional().default(""),
  displayOrder: z.coerce.number().int().optional().default(0),
  status: z.enum(["active", "disabled"]).optional().default("active"),
});
const materialAlertActionSchema = z.object({
  alertType: z.string().min(1),
  action: z.enum(["handled", "ignored"]),
  comment: z.string().optional().default(""),
});
const maintenanceSchema = z.object({
  instrumentId: z.string().min(1),
  type: z.enum(["routine", "fault", "emergency"]),
  priority: z.enum(["normal", "high", "urgent"]),
  handler: z.string(),
  description: z.string().min(1),
  startTime: z.string().datetime(),
  endTime: z.string().datetime(),
});
const maintenanceUpdateSchema = z.object({
  result: z.string().optional().default(""),
  reason: z.string().optional().default(""),
});
const businessConfigSchema = z.object({
  title: z.string().min(1),
  category: z.string().optional().default(""),
  scope: z.string().optional().default(""),
  status: z.enum(["draft", "active", "disabled", "archived"]).optional().default("active"),
  description: z.string().optional().default(""),
  configJson: z.string().optional().default("{}"),
});

app.post("/api/tenants", validateAndProxy(tenantSchema));
app.patch("/api/tenants/:id", validateAndProxy(tenantSchema));
app.patch("/api/notification-channel-settings/smtp", validateAndProxy(smtpSettingsSchema));
app.patch("/api/notification-channel-settings/wechat", validateAndProxy(wechatSettingsSchema));
app.get("/api/notification-channel-settings/dingtalk", proxyToGo);
app.patch("/api/notification-channel-settings/dingtalk", validateAndProxy(dingTalkSettingsSchema));
app.post("/api/notification-channel-settings/dingtalk/test", validateAndProxy(dingTalkTestSchema));
app.post("/api/dingtalk/events", proxyToGo);
app.post("/api/dingtalk/events/:tenant", proxyToGo);
app.post("/api/me/dingtalk-binding", validateAndProxy(dingTalkBindingSchema));
app.delete("/api/me/dingtalk-binding", proxyToGo);
app.get("/api/access-control-settings", proxyToGo);
app.patch("/api/access-control-settings", validateAndProxy(accessControlSettingsSchema));
app.post("/api/instruments", validateAndProxy(instrumentSchema));
app.patch("/api/instruments/:id", validateAndProxy(instrumentSchema));
app.delete("/api/instruments/:id", proxyToGo);
app.patch("/api/footer-settings", validateAndProxy(footerSettingsSchema));
app.patch("/api/copy-settings", validateAndProxy(copySettingsSchema));
app.post("/api/organization-units", validateAndProxy(organizationUnitSchema));
app.patch("/api/organization-units/:id", validateAndProxy(organizationUnitSchema));
app.delete("/api/organization-units/:id", proxyToGo);
app.post("/api/reservations", validateAndProxy(z.object({
  instrumentId: z.string().min(1),
  userName: z.string().optional().default(""),
  purpose: z.string().min(1),
  startTime: z.string().datetime(),
  endTime: z.string().datetime(),
  idempotencyKey: z.string().optional().default(""),
})));
app.post("/api/reservations/batch", validateAndProxy(z.object({
  instrumentId: z.string().min(1),
  userName: z.string().optional().default(""),
  purpose: z.string().min(1),
  periods: z.array(z.object({
    startTime: z.string().datetime(),
    endTime: z.string().datetime(),
  })).min(1).max(72),
  idempotencyKey: z.string().optional().default(""),
})));
app.patch("/api/reservations/:id/approve", validateAndProxy(reservationDecisionSchema));
app.patch("/api/reservations/:id/reject", validateAndProxy(reservationDecisionSchema));
app.patch("/api/reservations/:id/check-in", validateOptionalJsonAndProxy(emptyJsonObject));
app.patch("/api/reservations/:id/check-out", validateOptionalJsonAndProxy(emptyJsonObject));
app.patch("/api/reservations/:id/cancel", validateOptionalJsonAndProxy(reservationCancelSchema));
app.patch("/api/users/:id/review", validateAndProxy(userReviewSchema));
app.post("/api/users/:id/memberships", validateAndProxy(userMembershipSchema));
app.delete("/api/users/:id", proxyToGo);
app.post("/api/notifications", validateAndProxy(announcementSchema));
app.patch("/api/notifications/read-all", validateOptionalJsonAndProxy(emptyJsonObject));
app.patch("/api/notifications/:id", validateAndProxy(announcementSchema));
app.patch("/api/notifications/:id/read", validateOptionalJsonAndProxy(emptyJsonObject));
app.delete("/api/notifications/:id", proxyToGo);
app.post("/api/ledger/adjustments", validateAndProxy(ledgerAdjustmentSchema));
app.post("/api/financial-accounts", validateAndProxy(financialAccountSchema));
app.patch("/api/financial-accounts/:id", validateAndProxy(financialAccountSchema));
app.post("/api/materials", validateAndProxy(materialSchema));
app.post("/api/materials/import", proxyToGo);
app.post("/api/materials/import.csv", proxyToGo);
app.post("/api/materials/categories", validateAndProxy(materialCategorySchema));
app.patch("/api/materials/categories/:id", validateAndProxy(materialCategorySchema));
app.delete("/api/materials/categories/:id", proxyToGo);
app.patch("/api/materials/:id", validateAndProxy(materialSchema));
app.delete("/api/materials/:id", proxyToGo);
app.post("/api/materials/:id/stock-adjustments", validateAndProxy(stockAdjustmentSchema));
app.post("/api/materials/:id/alert-actions", validateAndProxy(materialAlertActionSchema));
app.post("/api/material-requests", validateAndProxy(materialRequestSchema));
app.patch("/api/material-requests/:id/approve", validateOptionalJsonAndProxy(reservationDecisionSchema));
app.patch("/api/material-requests/:id/reject", validateOptionalJsonAndProxy(reservationDecisionSchema));
app.patch("/api/material-requests/:id/outbound", validateOptionalJsonAndProxy(emptyJsonObject));
app.patch("/api/material-requests/:id/cancel", validateOptionalJsonAndProxy(emptyJsonObject));
app.post("/api/procurement-projects", validateAndProxy(procurementProjectSchema));
app.patch("/api/procurement-projects/:id", validateAndProxy(procurementProjectSchema));
app.delete("/api/procurement-projects/:id", proxyToGo);
app.post("/api/material-purchases", validateAndProxy(materialPurchaseSchema));
app.patch("/api/material-purchases/:id/approve", validateOptionalJsonAndProxy(reservationDecisionSchema));
app.patch("/api/material-purchases/:id/reject", validateOptionalJsonAndProxy(reservationDecisionSchema));
app.patch("/api/material-purchases/:id/order", validateOptionalJsonAndProxy(emptyJsonObject));
app.patch("/api/material-purchases/:id/receive", validateOptionalJsonAndProxy(emptyJsonObject));
app.patch("/api/material-purchases/:id/cancel", validateOptionalJsonAndProxy(emptyJsonObject));
app.post("/api/purchasable-materials/import", proxyToGo);
app.post("/api/material-damages", validateAndProxy(materialDamageSchema));
app.patch("/api/material-damages/:id/approve", validateOptionalJsonAndProxy(reservationDecisionSchema));
app.patch("/api/material-damages/:id/reject", validateOptionalJsonAndProxy(reservationDecisionSchema));
app.patch("/api/material-damages/:id/process", validateOptionalJsonAndProxy(emptyJsonObject));
app.patch("/api/material-damages/:id/cancel", validateOptionalJsonAndProxy(emptyJsonObject));
app.post("/api/maintenance", validateAndProxy(maintenanceSchema));
app.patch("/api/maintenance/:id/start", validateOptionalJsonAndProxy(emptyJsonObject));
app.patch("/api/maintenance/:id/cancel", validateAndProxy(z.object({ reason: z.string().optional().default("") })));
app.patch("/api/maintenance/:id/complete", validateAndProxy(maintenanceUpdateSchema));
app.post("/api/workflows/:kind", validateAndProxy(businessConfigSchema));
app.patch("/api/workflows/:kind/:id", validateAndProxy(businessConfigSchema));
app.post("/api/billing/:kind", validateAndProxy(businessConfigSchema));
app.patch("/api/billing/:kind/:id", validateAndProxy(businessConfigSchema));

app.post("/api/logout", proxyToGo);
app.post("/api/logout-all", proxyToGo);
app.get("/api/*", proxyToGo);
app.delete("/api/*", proxyToGo);
app.post("/api/*", validateOptionalJsonAndProxy(z.record(z.unknown())));
app.put("/api/*", validateOptionalJsonAndProxy(z.record(z.unknown())));
app.patch("/api/*", validateOptionalJsonAndProxy(z.record(z.unknown())));

serve({ fetch: app.fetch, port }, (info) => {
  console.log(`LIRS Hono API listening on ${info.port}`);
});

function validateAndProxy(schema: z.ZodTypeAny) {
  return async (c: Context) => {
    const body = await c.req.json().catch(() => null);
    const parsed = schema.safeParse(body);
    if (!parsed.success) {
      return c.json({ error: "invalid json payload", issues: parsed.error.flatten() }, 400);
    }
    return proxyToGo(c, parsed.data);
  };
}

function validateOptionalJsonAndProxy(schema: z.ZodTypeAny) {
  return async (c: Context) => {
    if (!isJsonRequest(c)) {
      return proxyToGo(c);
    }
    const body = await c.req.json().catch(() => ({}));
    const parsed = schema.safeParse(body);
    if (!parsed.success) {
      return c.json({ error: "invalid json payload", issues: parsed.error.flatten() }, 400);
    }
    return proxyToGo(c, parsed.data);
  };
}

function isJsonRequest(c: Context) {
  const contentType = c.req.header("content-type") ?? "";
  return contentType === "" || contentType.toLowerCase().includes("application/json");
}

async function proxyToGo(c: Context, overrideBody?: unknown) {
  const requestUrl = new URL(c.req.url);
  const upstreamUrl = `${goApiBaseUrl}${requestUrl.pathname}${requestUrl.search}`;
  const headers = forwardRequestHeaders(c.req.header());
  const method = c.req.method;
  const hasBody = !["GET", "HEAD"].includes(method);
  const hasOverrideBody = overrideBody !== undefined && typeof overrideBody !== "function";
  let body: BodyInit | undefined;
  if (hasOverrideBody) {
    body = JSON.stringify(overrideBody);
  } else if (hasBody) {
    const buffer = Buffer.from(await c.req.raw.arrayBuffer());
    body = buffer.byteLength === 0 ? undefined : buffer;
  }
  if (hasOverrideBody) {
    headers.set("Content-Type", "application/json");
  } else if (body !== undefined && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  let response: Response;
  try {
    response = await fetchWithTimeout(upstreamUrl, { method, headers, body }, proxyTimeoutMs);
  } catch (error) {
    const message = error instanceof Error && error.name === "AbortError" ? "后端接口超时，请稍后重试或联系管理员查看导入日志" : "后端接口不可用";
    return new Response(JSON.stringify({ error: message }), {
      status: 504,
      headers: { "Content-Type": "application/json; charset=utf-8" },
    });
  }

  const responseHeaders = new Headers(response.headers);
  responseHeaders.delete("content-encoding");
  responseHeaders.delete("content-length");
  return new Response(response.body, {
    status: response.status,
    headers: responseHeaders,
  });
}

function requestLogger(c: Context, next: Next) {
  const startedAt = Date.now();
  return next().finally(() => {
    console.info(`${c.req.method} ${new URL(c.req.url).pathname} ${c.res.status} ${Date.now() - startedAt}ms`);
  });
}

function requestBodySizeLimit(maxBytes: number) {
  return async (c: Context, next: Next) => {
    const contentLength = Number(c.req.header("content-length") ?? 0);
    if (contentLength > maxBytes) {
      return c.json({ error: "request body too large" }, 413);
    }
    await next();
  };
}

const rateLimitStore = new Map<string, { count: number; resetAt: number }>();

function rateLimit({ keyPrefix, limit, windowMs }: { keyPrefix: string; limit: number; windowMs: number }) {
  return async (c: Context, next: Next) => {
    const now = Date.now();
    if (rateLimitStore.size > 1000) {
      for (const [key, bucket] of rateLimitStore.entries()) {
        if (bucket.resetAt <= now) {
          rateLimitStore.delete(key);
        }
      }
    }
    const ip = clientIp(c);
    const key = `${keyPrefix}:${ip}`;
    const bucket = rateLimitStore.get(key);
    if (!bucket || bucket.resetAt <= now) {
      rateLimitStore.set(key, { count: 1, resetAt: now + windowMs });
      await next();
      return;
    }
    if (bucket.count >= limit) {
      return c.json({ error: "too many requests" }, 429);
    }
    bucket.count += 1;
    await next();
  };
}

function clientIp(c: Context) {
  return (c.req.header("x-forwarded-for") ?? c.req.header("cf-connecting-ip") ?? "unknown")
    .split(",")[0]
    .trim();
}

function forwardRequestHeaders(source: Record<string, string | undefined>) {
  const allowedHeaders = new Set([
    "accept",
    "accept-language",
    "authorization",
    "content-type",
    "cookie",
    "user-agent",
    "x-forwarded-for",
    "x-request-id",
  ]);
  const headers = new Headers();
  for (const [key, value] of Object.entries(source)) {
    if (value !== undefined && allowedHeaders.has(key.toLowerCase())) {
      headers.set(key, value);
    }
  }
  return headers;
}

async function fetchWithTimeout(input: string, init: RequestInit, timeoutMs: number) {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), timeoutMs);
  try {
    return await fetch(input, { ...init, signal: controller.signal });
  } finally {
    clearTimeout(timeout);
  }
}

function withTimeout<T>(promise: Promise<T>, timeoutMs: number, message: string) {
  return new Promise<T>((resolve, reject) => {
    const timer = setTimeout(() => reject(new Error(message)), timeoutMs);
    promise
      .then((value) => {
        clearTimeout(timer);
        resolve(value);
      })
      .catch((error) => {
        clearTimeout(timer);
        reject(error);
      });
  });
}
