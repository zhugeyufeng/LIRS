import { serve } from "@hono/node-server";
import { getConnInfo } from "@hono/node-server/conninfo";
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
const trustedProxyRanges = (process.env.TRUSTED_PROXY_CIDRS ?? "127.0.0.1/32,::1/128,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16")
  .split(",")
  .map((item) => parseCIDR(item.trim()))
  .filter((item): item is CIDRRange => item !== null);

const app = new Hono();

app.onError((error, c) => {
  if (error instanceof RequestBodyTooLargeError) {
    return c.json({ error: "request body too large" }, 413);
  }
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
const userCreateSchema = z.object({
  tenantId: z.string().optional().default(""),
  name: z.string().min(1),
  phone: z.string().min(1),
  email: z.string().email(),
  password: z.string().min(8),
  department: z.string().min(1),
  groupName: z.string().optional().default(""),
  role: z.enum(["unassigned", "student", "teacher", "researcher", "group_leader", "material_admin", "finance_admin", "tenant_admin", "lab_admin", "super_admin"]),
  status: z.enum(["pending_approval", "active", "disabled"]).optional().default("active"),
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
  purchaseSerialNo: z.string().optional().default(""),
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
  batchId: z.string().optional().default(""),
  unitId: z.string().min(1),
  quantity: z.coerce.number().int().positive(),
  purpose: z.string().min(1),
});
const materialPurchaseSchema = z.object({
  materialId: z.string().optional().default(""),
  purchasableMaterialId: z.string().optional().default(""),
  purchaseSerialNo: z.string().optional().default(""),
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
  quantity: z.coerce.number().int().positive(),
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
app.post("/api/users", validateAndProxy(userCreateSchema));
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
app.patch("/api/material-purchases/:id", validateAndProxy(materialPurchaseSchema));
app.patch("/api/material-purchases/:id/approve", validateOptionalJsonAndProxy(reservationDecisionSchema));
app.patch("/api/material-purchases/:id/reject", validateOptionalJsonAndProxy(reservationDecisionSchema));
app.patch("/api/material-purchases/:id/return", validateOptionalJsonAndProxy(reservationDecisionSchema));
app.patch("/api/material-purchases/:id/order", validateOptionalJsonAndProxy(emptyJsonObject));
app.patch("/api/material-purchases/:id/receive", validateOptionalJsonAndProxy(emptyJsonObject));
app.patch("/api/material-purchases/:id/cancel", validateOptionalJsonAndProxy(emptyJsonObject));
app.post("/api/material-purchases/monthly-confirmations", validateAndProxy(z.object({ month: z.string().regex(/^\d{4}-\d{2}$/) })));
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
    let body: unknown;
    try {
      body = await readLimitedJson(c.req.raw, null);
    } catch (error) {
      if (error instanceof RequestBodyTooLargeError) {
        return c.json({ error: "request body too large" }, 413);
      }
      return c.json({ error: "invalid json payload" }, 400);
    }
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
    let body: unknown;
    try {
      body = await readLimitedJson(c.req.raw, {});
    } catch (error) {
      if (error instanceof RequestBodyTooLargeError) {
        return c.json({ error: "request body too large" }, 413);
      }
      return c.json({ error: "invalid json payload" }, 400);
    }
    const parsed = schema.safeParse(body);
    if (!parsed.success) {
      return c.json({ error: "invalid json payload", issues: parsed.error.flatten() }, 400);
    }
    return proxyToGo(c, parsed.data);
  };
}

function isJsonRequest(c: Context) {
  const contentType = c.req.header("content-type") ?? "";
  return contentType.toLowerCase().includes("application/json");
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
    let buffer: ArrayBuffer;
    try {
      buffer = await readLimitedArrayBuffer(c.req.raw, maxRequestBodyBytes);
    } catch (error) {
      if (error instanceof RequestBodyTooLargeError) {
        return c.json({ error: "request body too large" }, 413);
      }
      throw error;
    }
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
    if (contentLengthExceeds(c.req.header("content-length"), maxBytes)) {
      return c.json({ error: "request body too large" }, 413);
    }
    await next();
  };
}

function contentLengthExceeds(value: string | undefined, maxBytes: number) {
  if (!value) {
    return false;
  }
  const contentLength = Number(value);
  return Number.isFinite(contentLength) && contentLength > maxBytes;
}

async function readLimitedJson(request: Request, emptyValue: unknown) {
  const text = await readLimitedText(request, maxRequestBodyBytes);
  if (text.trim() === "") {
    return emptyValue;
  }
  return JSON.parse(text) as unknown;
}

async function readLimitedText(request: Request, maxBytes: number) {
  const buffer = await readLimitedArrayBuffer(request, maxBytes);
  return new TextDecoder().decode(buffer);
}

async function readLimitedArrayBuffer(request: Request, maxBytes: number) {
  if (contentLengthExceeds(request.headers.get("content-length") ?? undefined, maxBytes)) {
    throw new RequestBodyTooLargeError();
  }
  if (!request.body) {
    return new ArrayBuffer(0);
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
        throw new RequestBodyTooLargeError();
      }
      chunks.push(value);
    }
  } finally {
    reader.releaseLock();
  }
  const result = new Uint8Array(received);
  let offset = 0;
  for (const chunk of chunks) {
    result.set(chunk, offset);
    offset += chunk.byteLength;
  }
  return result.buffer;
}

function rateLimit({ keyPrefix, limit, windowMs }: { keyPrefix: string; limit: number; windowMs: number }) {
  return async (c: Context, next: Next) => {
    const ip = clientIp(c);
    const key = `lirs:rate-limit:${keyPrefix}:${ip}`;
    const count = await incrementRateLimit(key, windowMs);
    if (count > limit) {
      return c.json({ error: "too many requests" }, 429);
    }
    await next();
  };
}

async function incrementRateLimit(key: string, windowMs: number) {
  const redis = await ensureRedis();
  const count = Number(await redis.sendCommand([
    "EVAL",
    "local count = redis.call('INCR', KEYS[1]); if count == 1 then redis.call('PEXPIRE', KEYS[1], ARGV[1]); end; return count",
    "1",
    key,
    String(windowMs),
  ]));
  return count;
}

function clientIp(c: Context) {
  const remoteIP = normalizeIP(getConnInfo(c).remote.address ?? "");
  const cfIP = normalizeIP(c.req.header("cf-connecting-ip") ?? "");
  if (cfIP && remoteIP && isTrustedProxy(remoteIP)) {
    return cfIP;
  }
  const forwardedFor = (c.req.header("x-forwarded-for") ?? "")
    .split(",")
    .map((item) => normalizeIP(item))
    .filter(Boolean);
  if (remoteIP && isTrustedProxy(remoteIP) && forwardedFor.length > 0) {
    for (let index = forwardedFor.length - 1; index >= 0; index--) {
      const candidate = forwardedFor[index];
      if (!isTrustedProxy(candidate)) {
        return candidate;
      }
    }
    return forwardedFor[0] ?? remoteIP;
  }
  return remoteIP || "unknown";
}

class RequestBodyTooLargeError extends Error {}

type CIDRRange = {
  base: bigint;
  mask: bigint;
  bits: 32 | 128;
};

function parseCIDR(value: string): CIDRRange | null {
  if (!value) {
    return null;
  }
  const [rawIP, rawPrefix] = value.includes("/") ? value.split("/", 2) : [value, ""];
  const parsed = ipToBigInt(rawIP);
  if (!parsed) {
    return null;
  }
  const prefix = rawPrefix === "" ? parsed.bits : Number(rawPrefix);
  if (!Number.isInteger(prefix) || prefix < 0 || prefix > parsed.bits) {
    return null;
  }
  const mask = cidrMask(parsed.bits, prefix);
  return { base: parsed.value & mask, mask, bits: parsed.bits };
}

function isTrustedProxy(ip: string) {
  const parsed = ipToBigInt(ip);
  if (!parsed) {
    return false;
  }
  return trustedProxyRanges.some((range) => range.bits === parsed.bits && (parsed.value & range.mask) === range.base);
}

function normalizeIP(value: string) {
  value = value.trim();
  if (!value) {
    return "";
  }
  if (value.startsWith("::ffff:")) {
    return value.slice("::ffff:".length);
  }
  if (value.startsWith("[") && value.includes("]")) {
    return value.slice(1, value.indexOf("]"));
  }
  const colonCount = (value.match(/:/g) ?? []).length;
  if (colonCount === 1 && value.includes(".")) {
    return value.split(":", 1)[0];
  }
  return value;
}

function ipToBigInt(value: string): { value: bigint; bits: 32 | 128 } | null {
  value = normalizeIP(value);
  const ipv4 = ipv4ToBigInt(value);
  if (ipv4 !== null) {
    return { value: ipv4, bits: 32 };
  }
  const ipv6 = ipv6ToBigInt(value);
  if (ipv6 !== null) {
    return { value: ipv6, bits: 128 };
  }
  return null;
}

function ipv4ToBigInt(value: string) {
  const parts = value.split(".");
  if (parts.length !== 4) {
    return null;
  }
  let result = 0n;
  for (const part of parts) {
    if (!/^\d+$/.test(part)) {
      return null;
    }
    const number = Number(part);
    if (number < 0 || number > 255) {
      return null;
    }
    result = (result << 8n) + BigInt(number);
  }
  return result;
}

function ipv6ToBigInt(value: string) {
  if (!value.includes(":")) {
    return null;
  }
  const sections = value.split("::");
  if (sections.length > 2) {
    return null;
  }
  const left = sections[0] ? sections[0].split(":") : [];
  const right = sections.length === 2 && sections[1] ? sections[1].split(":") : [];
  if (left.some((item) => item === "") || right.some((item) => item === "")) {
    return null;
  }
  const missing = 8 - left.length - right.length;
  if (missing < 0 || (sections.length === 1 && missing !== 0)) {
    return null;
  }
  const groups = [...left, ...Array(missing).fill("0"), ...right];
  if (groups.length !== 8) {
    return null;
  }
  let result = 0n;
  for (const group of groups) {
    if (!/^[0-9a-fA-F]{1,4}$/.test(group)) {
      return null;
    }
    result = (result << 16n) + BigInt(Number.parseInt(group, 16));
  }
  return result;
}

function cidrMask(bits: 32 | 128, prefix: number) {
  if (prefix === 0) {
    return 0n;
  }
  return ((1n << BigInt(prefix)) - 1n) << BigInt(bits - prefix);
}

function forwardRequestHeaders(source: Record<string, string | undefined>) {
  const allowedHeaders = new Set([
    "accept",
    "accept-language",
    "authorization",
    "content-type",
    "cookie",
    "user-agent",
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
