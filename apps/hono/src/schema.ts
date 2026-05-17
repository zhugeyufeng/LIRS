import { boolean, integer, pgTable, text, timestamp, uuid } from "drizzle-orm/pg-core";

export const users = pgTable("users", {
  id: uuid("id").primaryKey(),
  tenantId: uuid("tenant_id").notNull(),
  name: text("name").notNull(),
  email: text("email").notNull(),
  phone: text("phone").notNull(),
  department: text("department").notNull(),
  groupName: text("group_name").notNull(),
  role: text("role").notNull(),
  status: text("status").notNull(),
  emailVerified: boolean("email_verified").notNull(),
  dingTalkUserId: text("dingtalk_user_id").notNull(),
  dingTalkUnionId: text("dingtalk_union_id").notNull(),
  dingTalkName: text("dingtalk_name").notNull(),
  authEpoch: integer("auth_epoch").notNull(),
});

export const tenants = pgTable("tenants", {
  id: uuid("id").primaryKey(),
  name: text("name").notNull(),
  code: text("code").notNull(),
  financeEnabled: boolean("finance_enabled").notNull(),
  status: text("status").notNull(),
});

export const sessions = pgTable("sessions", {
  id: uuid("id").primaryKey(),
  userId: uuid("user_id").notNull(),
  tokenHash: text("token_hash").notNull(),
  authEpoch: integer("auth_epoch").notNull(),
  revokedAt: timestamp("revoked_at", { withTimezone: true }),
  expiresAt: timestamp("expires_at", { withTimezone: true }).notNull(),
  lastUsedAt: timestamp("last_used_at", { withTimezone: true }).notNull(),
});
