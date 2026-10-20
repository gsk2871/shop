// Node.js 22.6+: node --experimental-strip-types --test tests/ads-consent.test.mjs
import test from "node:test";
import assert from "node:assert/strict";
import { consentKey, consentLifetime, readConsent, saveConsent } from "../lib/ads-consent.ts";

test("consent is scoped, expires, rejects corrupt records and persists rejection", () => {
  const values = new Map();
  globalThis.localStorage = {
    getItem: key => values.get(key) ?? null,
    setItem: (key, value) => values.set(key, value),
    removeItem: key => values.delete(key),
  };
  globalThis.window = new EventTarget();
  const scope = "account-and-notice-v1";
  assert.equal(readConsent(scope), null);
  values.set("shop-cookie-choice-v1", JSON.stringify({ choice: "accepted", savedAt: Date.now() }));
  assert.equal(readConsent(scope), null, "legacy acceptance must not authorize advertising");
  assert.equal(saveConsent(scope, "accepted"), true);
  assert.equal(readConsent(scope).choice, "accepted");
  assert.equal(readConsent("different-recipient"), null);
  assert.equal(saveConsent(scope, "essential"), true);
  assert.equal(readConsent(scope).choice, "essential");
  for (const record of [
    "{invalid", "null",
    JSON.stringify({ scope, choice: "accepted", savedAt: Date.now() + 100000 }),
    JSON.stringify({ scope, choice: "accepted", savedAt: Date.now() - consentLifetime }),
    JSON.stringify({ scope, choice: "accepted", savedAt: String(Date.now()) }),
    JSON.stringify({ scope, choice: "unexpected", savedAt: Date.now() }),
  ]) {
    values.set(consentKey, record);
    assert.equal(readConsent(scope), null);
  }
  saveConsent(scope, "accepted");
  globalThis.localStorage.setItem = () => { throw new Error("storage unavailable"); };
  assert.equal(saveConsent(scope, "essential"), false);
  assert.equal(readConsent(scope), null, "failed persistence must disable reporting in this tab");
});
