import { query, mutation } from "./_generated/server";
import { v } from "convex/values";

export const list = query({
  args: {
    alertId: v.optional(v.id("alerts")),
    limit: v.optional(v.number()),
  },
  returns: v.array(
    v.object({
      _id: v.id("alertHistory"),
      _creationTime: v.number(),
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
  ),
  handler: async (ctx, args) => {
    const queryLimit = args.limit ?? 50;

    if (args.alertId) {
      return await ctx.db
        .query("alertHistory")
        .withIndex("by_alert_and_time", (q) => q.eq("alertId", args.alertId!))
        .order("desc")
        .take(queryLimit);
    }

    return await ctx.db
      .query("alertHistory")
      .withIndex("by_triggeredAt")
      .order("desc")
      .take(queryLimit);
  },
});

export const record = mutation({
  args: {
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
  },
  returns: v.id("alertHistory"),
  handler: async (ctx, args) => {
    return await ctx.db.insert("alertHistory", args);
  },
});
