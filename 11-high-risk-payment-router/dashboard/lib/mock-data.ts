export const mockPspHealth = [
  { code: "nuvei", label: "Nuvei", state: "closed", authRate: 87.4, p95: 410, region: "lhr" },
  { code: "trust", label: "Trust Payments", state: "closed", authRate: 84.1, p95: 380, region: "lhr" },
  { code: "worldpay", label: "Worldpay HR", state: "half_open", authRate: 78.8, p95: 620, region: "ams" },
  { code: "mollie", label: "Mollie", state: "closed", authRate: 91.2, p95: 290, region: "ams" }
];

export const mockRules = [
  { priority: 10, predicate: "GB + Visa + amount <= GBP 200", action: "Nuvei -> Worldpay -> Trust" },
  { priority: 20, predicate: "GB + Mastercard", action: "Worldpay -> Trust -> Nuvei" },
  { priority: 30, predicate: "DE, FR, NL, IE", action: "Trust -> Nuvei -> Worldpay" },
  { priority: 1000, predicate: "Fallback", action: "Nuvei -> Worldpay -> Trust -> Mollie" }
];

export const mockTransactions = [
  { id: "txn_demo_001", state: "captured", psp: "nuvei" },
  { id: "txn_demo_002", state: "failed_terminal", psp: "worldpay" },
  { id: "txn_demo_003", state: "requires_3ds", psp: "trust" }
];

export const mockWebhookDeliveries = [
  { id: "whout_001", event: "transaction.captured", status: "delivered", nextAttempt: "-" },
  { id: "whout_002", event: "transaction.failed", status: "pending", nextAttempt: "5m" },
  { id: "whout_003", event: "transaction.refunded", status: "dead", nextAttempt: "manual replay" }
];

