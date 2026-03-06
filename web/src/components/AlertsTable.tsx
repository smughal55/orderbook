import { useQuery, useMutation } from "convex/react";
import { api } from "../../../convex/_generated/api";
import type { Id } from "../../../convex/_generated/dataModel";

export function AlertsTable() {
  const alerts = useQuery(api.alerts.list, {});
  const toggleEnabled = useMutation(api.alerts.toggleEnabled);
  const removeAlert = useMutation(api.alerts.remove);

  if (!alerts) {
    return <div className="empty-state"><p>Loading...</p></div>;
  }

  if (alerts.length === 0) {
    return (
      <div className="empty-state">
        <p>No alert rules configured yet.</p>
        <p className="time-ago">Create your first alert rule to get started.</p>
      </div>
    );
  }

  const handleToggle = (id: Id<"alerts">) => {
    toggleEnabled({ id });
  };

  const handleDelete = (id: Id<"alerts">, name: string) => {
    if (confirm(`Delete alert rule "${name}"?`)) {
      removeAlert({ id });
    }
  };

  return (
    <div className="table-container">
      <table>
        <thead>
          <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Source</th>
            <th>Condition</th>
            <th>Channels</th>
            <th>Cooldown</th>
            <th>Status</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {alerts.map((alert) => (
            <tr key={alert._id}>
              <td style={{ fontWeight: 500 }}>{alert.name}</td>
              <td>
                <span className={`badge badge-${alert.evalType}`}>
                  {alert.evalType}
                </span>
              </td>
              <td className="condition-code">{alert.sourceFilter}</td>
              <td>
                <span className="condition-code">{alert.condition}</span>
                {alert.evalType === "trend" && alert.windowSeconds && (
                  <span className="time-ago"> / {alert.aggregation} over {alert.windowSeconds}s</span>
                )}
              </td>
              <td>
                {alert.deliveryChannels.map((ch, i) => (
                  <span key={i} className="badge badge-info" style={{ marginRight: 4 }}>
                    {ch.type}
                  </span>
                ))}
              </td>
              <td className="time-ago">{alert.cooldownSeconds}s</td>
              <td>
                <button
                  className={`toggle ${alert.enabled ? "on" : ""}`}
                  onClick={() => handleToggle(alert._id)}
                  title={alert.enabled ? "Disable" : "Enable"}
                />
              </td>
              <td>
                <div className="actions-cell">
                  <button
                    className="btn btn-danger btn-sm"
                    onClick={() => handleDelete(alert._id, alert.name)}
                  >
                    Delete
                  </button>
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
