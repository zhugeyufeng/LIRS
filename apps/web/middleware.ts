import { NextRequest, NextResponse } from "next/server";

import { authTokenKey } from "./lib/auth-cookie";

const protectedPrefixes = [
  "/admin",
  "/approvals",
  "/dashboard",
  "/finance",
  "/maintenance",
  "/materials",
  "/notifications",
  "/instruments",
  "/operations",
  "/reservations",
  "/training",
  "/spaces",
  "/samples",
  "/iot",
  "/ai-assistant",
  "/data-center",
  "/settings",
];

export async function middleware(request: NextRequest) {
  const pathname = request.nextUrl.pathname;
  if (!protectedPrefixes.some((prefix) => pathname === prefix || pathname.startsWith(`${prefix}/`))) {
    return NextResponse.next();
  }
  const token = request.cookies.get(authTokenKey)?.value ?? "";
  if (token) {
    return NextResponse.next();
  }
  const url = request.nextUrl.clone();
  url.pathname = "/login";
  url.searchParams.set("next", pathname);
  return NextResponse.redirect(url);
}

export const config = {
  matcher: [
    "/admin/:path*",
    "/approvals/:path*",
    "/dashboard/:path*",
    "/finance/:path*",
    "/instruments/:path*",
    "/maintenance/:path*",
    "/materials/:path*",
    "/notifications/:path*",
    "/operations/:path*",
    "/reservations/:path*",
    "/training/:path*",
    "/spaces/:path*",
    "/samples/:path*",
    "/iot/:path*",
    "/ai-assistant/:path*",
    "/data-center/:path*",
    "/settings/:path*",
  ],
};
