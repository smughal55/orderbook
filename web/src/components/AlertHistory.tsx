import { useQuery } from "convex/react";
import { api } from "../../../convex/_generated/api";

export function AlertHistory() {
  const history = useQuery(api.alertHistory.list, { limit: 100 });
  const alerts = useQuery(api.alerts.list, {});

  if (!history || !alerts) {
    return <div className="empty-state"><p>Loading...</p></div>;
  }

  if (history.length === 0) {
    return (
      <div className="empty-state">
        <p>No alerts have been triggered yet.</p>
        <p className="time-ago">Alerts will appear here once they fire.</p>
      </div>
    );
  }

  const alertMap = new Map(alerts.map((a) => [a._id, a]));

  const formatTime = (ts: number) => {
    const d = new Date(ts);
    return d.toLocaleString();
  };

  return (
    <div className="table-container">
      <table>
        <thead>
          <tr>
            <th>Time</th>
            <th>Rule</th>
            <th>Source</th>
            <th>Severity</th>
            <th>Matched Value</th>
            <th>Delivered To</th>
          </tr>
        </thead>
        <tbody>
          {history.map((entry) => {
            const rule = alertMap.get(entry.alertId);
            return (
              <tr key={entry._id}>
                <td className="time-ago">{formatTime(entry.triggeredAt)}</td>
                <td style={{ fontWeight: 500 }}>
                  {rule?.name ?? "Deleted Rule"}
                </td>
                <td className="condition-code">{entry.sourceId}</td>
                <td>
                  <span className={`badge badge-${entry.severity}`}>
                    {entry.severity}
                  </span>
                </td>
                <td>
                  <span className="condition-code">{entry.matchedValue}</span>
                </td>
                <td>
                  {entry.deliveredTo.map((ch, i) => (
                    <span key={i} className="badge badge-info" style={{ marginRight: 4 }}>
                      {ch}
                    </span>
                  ))}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
