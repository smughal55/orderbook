import { useState } from "react";
import { useMutation } from "convex/react";
import { api } from "../../../convex/_generated/api";

interface Props {
  onClose: () => void;
}

interface Channel {
  type: "email" | "sms" | "webhook";
  target: string;
}

export function CreateAlertModal({ onClose }: Props) {
  const create = useMutation(api.alerts.create);

  const [name, setName] = useState("");
  const [sourceFilter, setSourceFilter] = useState("*");
  const [evalType, setEvalType] = useState<"single" | "trend">("single");
  const [condition, setCondition] = useState("");
  const [windowSeconds, setWindowSeconds] = useState(300);
  const [aggregation, setAggregation] = useState<
    "avg" | "max" | "min" | "rate_of_change"
  >("avg");
  const [field, setField] = useState("price");
  const [cooldownSeconds, setCooldownSeconds] = useState(60);
  const [channels, setChannels] = useState<Channel[]>([
    { type: "webhook", target: "" },
  ]);
  const [submitting, setSubmitting] = useState(false);

  const addChannel = () => {
    setChannels([...channels, { type: "webhook", target: "" }]);
  };

  const removeChannel = (idx: number) => {
    setChannels(channels.filter((_, i) => i !== idx));
  };

  const updateChannel = (idx: number, updates: Partial<Channel>) => {
    setChannels(
      channels.map((ch, i) => (i === idx ? { ...ch, ...updates } : ch))
    );
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name || !condition || channels.some((c) => !c.target)) return;

    setSubmitting(true);
    try {
      await create({
        name,
        sourceFilter,
        evalType,
        condition,
        windowSeconds: evalType === "trend" ? windowSeconds : undefined,
        aggregation: evalType === "trend" ? aggregation : undefined,
        field: evalType === "trend" ? field : undefined,
        deliveryChannels: channels,
        cooldownSeconds,
        enabled: true,
      });
      onClose();
    } catch (err) {
      console.error("Failed to create alert:", err);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <h2>Create Alert Rule</h2>
        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label>Name</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g., High Price Alert"
              required
            />
          </div>

          <div className="form-row">
            <div className="form-group">
              <label>Evaluation Type</label>
              <select
                value={evalType}
                onChange={(e) =>
                  setEvalType(e.target.value as "single" | "trend")
                }
              >
                <option value="single">Single Record</option>
                <option value="trend">Trend (Windowed)</option>
              </select>
            </div>
            <div className="form-group">
              <label>Source Filter</label>
              <input
                type="text"
                value={sourceFilter}
                onChange={(e) => setSourceFilter(e.target.value)}
                placeholder="* for all sources"
              />
            </div>
          </div>

          <div className="form-group">
            <label>Condition (expression)</label>
            <input
              type="text"
              value={condition}
              onChange={(e) => setCondition(e.target.value)}
              placeholder={
                evalType === "single"
                  ? "payload.price > 150"
                  : "value > 120"
              }
              required
            />
          </div>

          {evalType === "trend" && (
            <>
              <div className="form-row">
                <div className="form-group">
                  <label>Payload Field</label>
                  <input
                    type="text"
                    value={field}
                    onChange={(e) => setField(e.target.value)}
                    placeholder="price"
                  />
                </div>
                <div className="form-group">
                  <label>Aggregation</label>
                  <select
                    value={aggregation}
                    onChange={(e) =>
                      setAggregation(
                        e.target.value as
                          | "avg"
                          | "max"
                          | "min"
                          | "rate_of_change"
                      )
                    }
                  >
                    <option value="avg">Average</option>
                    <option value="max">Maximum</option>
                    <option value="min">Minimum</option>
                    <option value="rate_of_change">Rate of Change</option>
                  </select>
                </div>
              </div>
              <div className="form-group">
                <label>Window (seconds)</label>
                <input
                  type="number"
                  value={windowSeconds}
                  onChange={(e) => setWindowSeconds(Number(e.target.value))}
                  min={1}
                />
              </div>
            </>
          )}

          <div className="form-group">
            <label>Cooldown (seconds)</label>
            <input
              type="number"
              value={cooldownSeconds}
              onChange={(e) => setCooldownSeconds(Number(e.target.value))}
              min={0}
            />
          </div>

          <div className="form-group">
            <label>
              Delivery Channels{" "}
              <button
                type="button"
                className="btn btn-ghost btn-sm"
                onClick={addChannel}
                style={{ marginLeft: 8 }}
              >
                + Add
              </button>
            </label>
            <div className="channels-list">
              {channels.map((ch, idx) => (
                <div key={idx} className="channel-row">
                  <select
                    value={ch.type}
                    onChange={(e) =>
                      updateChannel(idx, {
                        type: e.target.value as Channel["type"],
                      })
                    }
                  >
                    <option value="email">Email</option>
                    <option value="sms">SMS</option>
                    <option value="webhook">Webhook</option>
                  </select>
                  <input
                    type="text"
                    value={ch.target}
                    onChange={(e) =>
                      updateChannel(idx, { target: e.target.value })
                    }
                    placeholder={
                      ch.type === "email"
                        ? "ops@example.com"
                        : ch.type === "sms"
                          ? "+1234567890"
                          : "https://hooks.example.com/alert"
                    }
                    required
                  />
                  {channels.length > 1 && (
                    <button
                      type="button"
                      className="btn btn-ghost btn-sm"
                      onClick={() => removeChannel(idx)}
                    >
                      x
                    </button>
                  )}
                </div>
              ))}
            </div>
          </div>

          <div className="form-actions">
            <button type="button" className="btn btn-ghost" onClick={onClose}>
              Cancel
            </button>
            <button
              type="submit"
              className="btn btn-primary"
              disabled={submitting}
            >
              {submitting ? "Creating..." : "Create Rule"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
