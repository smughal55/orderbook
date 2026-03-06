import { useState } from "react";
import { AlertsTable } from "./components/AlertsTable";
import { AlertHistory } from "./components/AlertHistory";
import { CreateAlertModal } from "./components/CreateAlertModal";

export function App() {
  const [tab, setTab] = useState<"rules" | "history">("rules");
  const [showCreate, setShowCreate] = useState(false);

  return (
    <div className="app">
      <div className="header">
        <h1>Alert Configuration</h1>
        {tab === "rules" && (
          <button className="btn btn-primary" onClick={() => setShowCreate(true)}>
            + New Alert Rule
          </button>
        )}
      </div>

      <div className="tabs">
        <button
          className={`tab ${tab === "rules" ? "active" : ""}`}
          onClick={() => setTab("rules")}
        >
          Alert Rules
        </button>
        <button
          className={`tab ${tab === "history" ? "active" : ""}`}
          onClick={() => setTab("history")}
        >
          Alert History
        </button>
      </div>

      {tab === "rules" && <AlertsTable />}
      {tab === "history" && <AlertHistory />}

      {showCreate && <CreateAlertModal onClose={() => setShowCreate(false)} />}
    </div>
  );
}
