# Usage

## Tools

Read tools change nothing and are the only ones registered with `CASHPILOT_READ_ONLY=true`. Tools that return records or lists send them as `structuredContent` (with an `outputSchema`) and as JSON text.

| Tool | Parameters | Notes |
|------|------------|-------|
| `get_earnings_summary` | none | Totals in USD: today, month, all time |
| `get_earnings_breakdown` | none | One record per service, with cashout eligibility |
| `get_earnings_daily` | `days` 1-365 (default 7) | One total per day |
| `get_earnings_history` | `period`: week, month, year, all | Earnings records over the period |
| `list_deployed_services` | none | Slug, worker, status, balance, CPU, memory |
| `list_available_services` | none | Catalog minus broken or gone services; includes each provider's signup link and the env keys a deploy needs |
| `get_fleet_summary` | none | Workers and containers |
| `list_workers` | none | Worker ids for `worker_id` |
| `get_health_scores` | `days` 1-90 (default 7) | Score 0-100 per service |
| `get_collector_alerts` | none | Errors from the last collection run |
| `get_service_logs` | `slug`, `lines` 1-1000 (default 50), `worker_id`, `instance` | Third-party output, fenced as untrusted. Needs the admin key |
| `get_compose` | `slug`, `arch`: amd64, arm64, arm | docker-compose.yml text |
| `trigger_collection` | none | Runs in the background; read earnings again a minute later |
| `start_service`, `restart_service`, `stop_service` | `slug`, `worker_id`, `instance` 1-10 | `stop_service` is marked destructive: the service stops earning |
| `deploy_service` | `slug`, `env` (object of strings), `hostname`, `instances` 1-10, `worker_id` | Deploys or redeploys; can take minutes while the image is pulled |
| `remove_service` | `slug`, `worker_id` | Deletes the containers and keeps the volumes |

`worker_id` is needed whenever more than one worker is online; CashPilot picks the worker itself when only one is. Each mutating call is logged to stderr (tool, slug, worker, instance; never env values).

## Resources

| URI | Content |
|-----|---------|
| `cashpilot://earnings/summary` | Same as `get_earnings_summary` |
| `cashpilot://earnings/breakdown` | Same as `get_earnings_breakdown` |
| `cashpilot://services/deployed` | Same as `list_deployed_services` |
| `cashpilot://services/catalog` | The full catalog, broken services included |
| `cashpilot://fleet/summary` | Same as `get_fleet_summary` |
| `cashpilot://workers` | Same as `list_workers` |
| `cashpilot://health/scores` | Health scores over 7 days |
| `cashpilot://collector-alerts` | Same as `get_collector_alerts` |

Everything is served on stdio or on one Streamable HTTP endpoint (`/mcp`).
