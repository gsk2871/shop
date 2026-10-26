// Run in an isolated environment: node --experimental-strip-types --test tests/leads-route.test.mjs
import test from "node:test";
import assert from "node:assert/strict";
import { POST } from "../app/api/leads/route.ts";

const request = (body, headers = {}, url = "https://shop.example/api/leads") => new Request(url, {
  method: "POST", body: typeof body === "string" ? body : JSON.stringify(body),
  headers: { "content-type": "application/json", "x-client-ip": "203.0.113.1", origin: "https://shop.example", ...headers },
});
test("lead proxy bounds and sanitizes public requests", async () => {
  const original = globalThis.fetch;
  let calls = 0;
  globalThis.fetch = async (url, options) => {
    calls++;
    assert.equal(new URL(url).pathname, "/api/v1/leads");
    assert.equal(options.credentials, "omit");
    assert.equal(options.headers["X-Client-IP"], "203.0.113.1");
    assert.equal(Object.keys(options.headers).length, 2, "must not forward browser cookies or arbitrary headers");
    assert.deepEqual(JSON.parse(options.body), { contact: "a@example.com" });
    return Response.json({ ok: true });
  };
  try {
    for (const [req, code] of [
      [request({ contact: "a@example.com" }, { origin: "https://attacker.example" }), 403],
      [request({ contact: "a@example.com" }, { "sec-fetch-site": "cross-site" }), 403],
      [request({ contact: "a@example.com" }, { "x-client-ip": "bad" }), 400],
      [request({ contact: "a@example.com" }, { "content-type": "text/plain" }), 400],
      [request({ contact: "a@example.com", status: "completed" }), 400],
      [request({ contact: 123 }), 400],
      [request("null"), 400], [request("{invalid"), 400],
      [request("x".repeat(4097)), 413],
      [request({ contact: "a@example.com" }, {}, "https://shop.example/api/leads?source=ignored"), 400],
    ]) assert.equal((await POST(req)).status, code);
    assert.equal(calls, 0);
    const response = await POST(request({ contact: "a@example.com" }, { cookie: "secret=do-not-forward" }));
    assert.equal(response.status, 200);
    assert.equal(response.headers.get("cache-control"), "no-store");
    assert.deepEqual(await response.json(), { ok: true });
    assert.equal(calls, 1);
    globalThis.fetch = async () => Response.json({ code: "RATE_LIMITED" }, { status: 429 });
    const limited = await POST(request({ contact: "a@example.com" }));
    assert.equal(limited.status, 429);
    assert.equal(limited.headers.get("retry-after"), "600");
    globalThis.fetch = async () => { throw new Error("private upstream detail"); };
    const failed = await POST(request({ contact: "a@example.com" }));
    assert.equal(failed.status, 503);
    assert.deepEqual(await failed.json(), { code: "SERVICE_UNAVAILABLE" });
  } finally { globalThis.fetch = original; }
});
