import assert from "node:assert/strict";
import test from "node:test";

const forbiddenFields = new Set(["card_number", "cardNumber", "pan", "primary_account_number", "cvv", "cvc", "expiry", "expiration"]);
const rawPanLike = /\b[0-9][0-9 -]{11,22}[0-9]\b/;

function rejectRawCardData(value) {
  if (typeof value === "string") return rawPanLike.test(value) ? "raw card-like value is not allowed" : null;
  if (Array.isArray(value)) return value.map(rejectRawCardData).find(Boolean) ?? null;
  if (value && typeof value === "object") {
    for (const [key, nested] of Object.entries(value)) {
      if (forbiddenFields.has(key)) return `raw card field ${key} is not allowed`;
      const err = rejectRawCardData(nested);
      if (err) return err;
    }
  }
  return null;
}

test("token-only payload is accepted", () => {
  assert.equal(rejectRawCardData({ payment_method_token: "btok_demo_token", amount_minor: 4200 }), null);
});

test("raw card field is rejected", () => {
  assert.match(rejectRawCardData({ card_number: "not allowed" }), /raw card field/);
});

test("long numeric string is rejected", () => {
  assert.match(rejectRawCardData({ payment_method_token: "1234 5678 9012 3" }), /raw card-like/);
});
