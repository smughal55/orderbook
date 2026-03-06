import { query, mutation } from "./_generated/server";
import { v } from "convex/values";

export const list = query({
  args: {
    enabledOnly: v.optional(v.boolean()),
  },
  returns: v.array(
    v.object({
      _id: v.id("alerts"),
      _creationTime: v.number(),
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
  ),
  handler: async (ctx, args) => {
    if (args.enabledOnly) {
      return await ctx.db
        .query("alerts")
        .withIndex("by_enabled", (q) => q.eq("enabled", true))
        .collect();
    }
    return await ctx.db.query("alerts").collect();
  },
});

export const get = query({
  args: { id: v.id("alerts") },
  returns: v.union(
    v.object({
      _id: v.id("alerts"),
      _creationTime: v.number(),
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
    }),
    v.null()
  ),
  handler: async (ctx, args) => {
    return await ctx.db.get(args.id);
  },
});

export const create = mutation({
  args: {
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
  },
  returns: v.id("alerts"),
  handler: async (ctx, args) => {
    return await ctx.db.insert("alerts", {
      ...args,
      createdBy: "system",
      createdAt: Date.now(),
    });
  },
});

export const update = mutation({
  args: {
    id: v.id("alerts"),
    name: v.optional(v.string()),
    sourceFilter: v.optional(v.string()),
    evalType: v.optional(v.union(v.literal("single"), v.literal("trend"))),
    condition: v.optional(v.string()),
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
    deliveryChannels: v.optional(
      v.array(
        v.object({
          type: v.union(
            v.literal("email"),
            v.literal("sms"),
            v.literal("webhook")
          ),
          target: v.string(),
        })
      )
    ),
    cooldownSeconds: v.optional(v.number()),
    enabled: v.optional(v.boolean()),
  },
  returns: v.id("alerts"),
  handler: async (ctx, args) => {
    const { id, ...updates } = args;
    const existing = await ctx.db.get(id);
    if (!existing) throw new Error("Alert not found");

    const filteredUpdates: Record<string, unknown> = {};
    for (const [key, value] of Object.entries(updates)) {
      if (value !== undefined) {
        filteredUpdates[key] = value;
      }
    }
    filteredUpdates.updatedAt = Date.now();

    await ctx.db.patch(id, filteredUpdates);
    return id;
  },
});

export const remove = mutation({
  args: { id: v.id("alerts") },
  returns: v.null(),
  handler: async (ctx, args) => {
    const existing = await ctx.db.get(args.id);
    if (!existing) throw new Error("Alert not found");
    await ctx.db.delete(args.id);
    return null;
  },
});

export const toggleEnabled = mutation({
  args: { id: v.id("alerts") },
  returns: v.boolean(),
  handler: async (ctx, args) => {
    const existing = await ctx.db.get(args.id);
    if (!existing) throw new Error("Alert not found");
    const newEnabled = !existing.enabled;
    await ctx.db.patch(args.id, {
      enabled: newEnabled,
      updatedAt: Date.now(),
    });
    return newEnabled;
  },
});
