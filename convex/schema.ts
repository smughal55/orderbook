import { defineSchema, defineTable } from "convex/server";
import { v } from "convex/values";

export default defineSchema({
  alerts: defineTable({
    name: v.string(),
    sourceFilter: v.string(),
    evalType: v.union(v.literal("single"), v.literal("trend")),
    condition: v.string(),
    windowSeconds: v.optional(v.number()),
    aggregation: v.optional(
      v.union(
        v.literal("avg"),
        v.literal("max"),
        v.literal("min"),
        v.literal("rate_of_change")
      )
    ),
    field: v.optional(v.string()),
    deliveryChannels: v.array(
      v.object({
        type: v.union(
          v.literal("email"),
          v.literal("sms"),
          v.literal("webhook")
        ),
        target: v.string(),
      })
    ),
    cooldownSeconds: v.number(),
    enabled: v.boolean(),
    createdBy: v.string(),
    createdAt: v.number(),
    updatedAt: v.optional(v.number()),
  })
    .index("by_enabled", ["enabled"])
    .index("by_evalType", ["evalType"])
    .index("by_createdBy", ["createdBy"]),

  alertHistory: defineTable({
    alertId: v.id("alerts"),
    triggeredAt: v.number(),
    matchedValue: v.string(),
    recordId: v.string(),
    sourceId: v.string(),
    severity: v.union(
      v.literal("info"),
      v.literal("warning"),
      v.literal("critical")
    ),
    deliveredTo: v.array(v.string()),
  })
    .index("by_alert", ["alertId"])
    .index("by_triggeredAt", ["triggeredAt"])
    .index("by_alert_and_time", ["alertId", "triggeredAt"]),
});
