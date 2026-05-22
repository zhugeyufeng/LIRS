import assert from "node:assert/strict";
import { readFileSync } from "node:fs";

const source = readFileSync(new URL("./middleware.ts", import.meta.url), "utf8");
const publicInstrumentPattern = /\/\^\\\/instruments\\\/\[\^\/\]\+\(\?:\\\/calendar\)\?\$\/\.test\(pathname\)/;

assert.match(source, /pathname === "\/instruments"/, "仪器预约大厅必须允许未登录查看");
assert.ok(publicInstrumentPattern.test(source), "仪器详情和仪器日历必须允许未登录查看");
assert.match(source, /"\/instruments\/:path\*"/, "仪器子路径仍需经过 middleware，预约提交页必须保持登录拦截");
assert.doesNotMatch(source, /protectedPrefixes[\s\S]*"\/instruments"[\s\S]*\];/, "仪器根路径不能作为整段受保护前缀拦截");
