import { createHash } from "node:crypto";
import { and, eq, gt, isNull, ne, sql } from "drizzle-orm";
import { db, pool } from "./db.js";
import { sessions, tenants, users } from "./schema.js";

export function tokenHash(token: string) {
  return createHash("sha256").update(token).digest("hex");
}

export async function currentUserFromToken(token: string) {
  const hash = tokenHash(token);
  const [row] = await db
    .select({
      id: users.id,
      tenantId: users.tenantId,
      tenantName: tenants.name,
      tenantCode: tenants.code,
      name: users.name,
      email: users.email,
      phone: users.phone,
      department: users.department,
      groupName: users.groupName,
      role: users.role,
      status: users.status,
      emailVerified: users.emailVerified,
      dingTalkUserId: users.dingTalkUserId,
      dingTalkUnionId: users.dingTalkUnionId,
      dingTalkName: users.dingTalkName,
      dingTalkBound: sql<boolean>`${users.dingTalkUserId} <> ''`,
      financeEnabled: tenants.financeEnabled,
      authEpoch: users.authEpoch,
    })
    .from(sessions)
    .innerJoin(users, eq(users.id, sessions.userId))
    .innerJoin(tenants, eq(tenants.id, users.tenantId))
    .where(
      and(
        eq(sessions.tokenHash, hash),
        isNull(sessions.revokedAt),
        gt(sessions.expiresAt, new Date()),
        eq(sessions.authEpoch, users.authEpoch),
        ne(users.status, "disabled"),
        eq(tenants.status, "active"),
      ),
    )
    .limit(1);

  if (!row) {
    return null;
  }
  await pool.query("UPDATE sessions SET last_used_at = now() WHERE token_hash = $1", [hash]);
  return row;
}

export function bearerToken(header: string | undefined) {
  if (!header?.startsWith("Bearer ")) {
    return "";
  }
  return header.slice("Bearer ".length).trim();
}
