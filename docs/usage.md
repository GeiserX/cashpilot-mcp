# Usage

## Tools

Read tools change nothing. `CASHPILOT_READ_ONLY=true` registers only the five the read-only key allows: `get_earnings_summary`, `get_earnings_breakdown`, `list_deployed_services`, `get_fleet_summary` and `get_health_scores`. CashPilot refuses that key everywhere else. Tools that return records or lists send them as `structuredContent` (with an `outputSchema`) and as JSON text.

| Tool | Parameters | Notes |
|------|------------|-------|
| `get_earnings_summary` | none | Totals in USD: today, month, all time |
| `get_earnings_breakdown` | none | One record per service, with cashout eligibility; each balance is in the unit its `currency` field names (USD, a coin or points) |
| `get_earnings_daily` | `days` 1-365 (default 7) | One total per day |
| `get_earnings_history` | `period`: week, month, year, all | Earnings records over the period, each in the unit its `currency` field names |
| `list_deployed_services` | none | Slug, worker, status, balance, CPU, memory |
| `list_available_services` | none | Catalog minus broken or gone services; includes each provider's signup link and the env keys a deploy needs |
| `get_fleet_summary` | none | Workers and containers |
| `list_workers` | none | Worker ids for `worker_id` |
| `get_health_scores` | `days` 1-90 (default 7) | Score 0-100 per service |
| `get_collector_alerts` | none | Errors from the last collection run |
| `get_service_logs` | `slug`, `lines` 1-1000 (default 50), `worker_id`, `instance` | Third-party output, fenced as untrusted |
| `get_compose` | `slug`, `arch`: amd64, arm64, arm | docker-compose.yml text |
| `trigger_collection` | none | Runs in the background; read earnings again a minute later |
| `start_service`, `restart_service`, `stop_service` | `slug`, `worker_id`, `instance` 1-10 | `stop_service` is marked destructive: the service stops earning |
| `deploy_service` | `slug`, `env` (object of strings), `hostname`, `instances` 1-10, `worker_id`, `replace_containers` | Deploys; can take minutes while the image is pulled. See [Redeploy a running service](#redeploy-a-running-service) |
| `remove_service` | `slug`, `worker_id` | Deletes the containers and keeps the volumes |

`worker_id` is needed whenever more than one worker is online; CashPilot picks the worker itself when only one is. Each mutating call is logged to stderr (tool, slug, worker, instance; never env values).

## Redeploy a running service

A redeploy, or a new number of instances for a running service, replaces its containers. CashPilot 3.0 and later do that only after the owner agrees:

1. The agent calls `deploy_service`. CashPilot replaces nothing and lists the containers it would replace. The answer gives the `replace_containers` value to send back, for example `{"1":"0123456789abcdef..."}`.
2. The agent shows that list to the owner and asks.
3. If the owner agrees, the agent calls `deploy_service` again with the same arguments plus that `replace_containers`. A 3.0 worker replaces exactly those containers.

A worker older than 3.0 cannot check the IDs. For such a worker, the list holds the containers it last reported, so it could replace a different one. The answer from `deploy_service` says so, and the agent must tell the owner before asking.

If a container changed between the two calls, CashPilot stops at that container. Copies it reached before it may already run the new settings. The agent then starts again at step 1. A service that does not run yet deploys on the first call. CashPilot before 3.0 redeploys on the first call.

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

With `CASHPILOT_READ_ONLY=true` only the earnings, deployed services, fleet summary and health scores resources are registered.

Everything is served on stdio or on one Streamable HTTP endpoint (`/mcp`).
