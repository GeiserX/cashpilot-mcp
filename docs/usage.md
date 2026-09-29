# Usage

| Type          | What for                                                           | MCP URI / Tool id                |
|---------------|--------------------------------------------------------------------|----------------------------------|
| **Resources** | Browse earnings, services, fleet status, and health scores read-only | `cashpilot://earnings/summary`<br>`cashpilot://earnings/breakdown`<br>`cashpilot://services/deployed`<br>`cashpilot://services/catalog`<br>`cashpilot://fleet/summary`<br>`cashpilot://workers`<br>`cashpilot://health/scores`<br>`cashpilot://collector-alerts` |
| **Tools**     | Query earnings, manage services, and trigger collection             | `get_earnings_daily`<br>`get_earnings_history`<br>`get_service_logs`<br>`restart_service`<br>`stop_service`<br>`start_service`<br>`deploy_service`<br>`remove_service`<br>`trigger_collection`<br>`get_compose` |

Everything is exposed over a single JSON-RPC endpoint (`/mcp`).
LLMs / Agents can: `initialize` -> `readResource` -> `listTools` -> `callTool` ... and so on.
