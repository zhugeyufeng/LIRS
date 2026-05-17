import { NextRequest, NextResponse } from "next/server";

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
  "/lims",
  "/eln",
  "/samples",
  "/iot",
  "/ai-assistant",
  "/data-center",
  "/settings",
];

export function middleware(request: NextRequest) {
  const pathname = request.nextUrl.pathname;
  if (!protectedPrefixes.some((prefix) => pathname === prefix || pathname.startsWith(`${prefix}/`))) {
    return NextResponse.next();
  }
  if (request.cookies.get("lirs.authToken")?.value) {
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
    "/lims/:path*",
    "/eln/:path*",
    "/samples/:path*",
    "/iot/:path*",
    "/ai-assistant/:path*",
    "/data-center/:path*",
    "/settings/:path*",
  ],
};
