# Provisioned demo assets (context: `wbkprez`)

What was created on 2026-05-11, with IDs for cleanup / re-use.

Manifests (sanitised, for recreation): `manifests/`.

## Folder

| Name | UID |
|------|-----|
| `demo` | `fflqs43gpv7cwa` |

## Alert rule

- Title: **`payment error rate high`**
- Group: `payment-demo`
- Folder UID: `fflqs43gpv7cwa`
- UID: `bflqs4pcrmi2oc`
- Condition (fires when ratio > 0.05 for 1m):
  ```promql
  (sum(rate(http_server_request_duration_seconds_count{service_name="payment",http_response_status_code=~"5.."}[5m])) or vector(0))
  /
  clamp_min(sum(rate(http_server_request_duration_seconds_count{service_name="payment"}[5m])), 0.001)
  ```
- Source: Beyla auto-instrumentation (OTel semconv on all 7 services in namespace `shop`)
- Labels: `service=payment`, `severity=critical`, `runbook=payment-error-spike`
- `noDataState: OK`, `execErrState: OK`

## Contact point

| Name | Type | UID |
|------|------|-----|
| `payment-demo` | `oncall` | `cflqs2ow6nugwd` |

Settings → URL: the IRM integration URL below.

## Notification policy

Added a route on the root tree:

- Matcher: `service = payment`
- Receiver: `payment-demo`
- `group_wait: 0s`, `group_interval: 1m`, `repeat_interval: 5m`

## IRM (OnCall) assets

| Asset | Name | ID |
|-------|------|-----|
| Integration (`grafana_alerting`) | `payment-demo` | `CA9SJ5X6ZJFVN` |
| Default route on the integration | — | `RQ3RQ452XC52X` |
| Non-default route (`.*` regex, position 0; gates `declare_incident`) | — | `RRS4VY9NEUZT2` |
| Escalation chain (bound to both routes) | `payment-demo` | `FDN21BG5BR5L7` |
| Escalation policy: **Declare Incident** (step 19, severity Major) | — | `ECX1SI1Q42BJK` |
| Outgoing webhook (preset `grafana_assistant`, trigger `Incident Declared`) | `Grafana Assistant - Incidents - incident declared` | `WHFRBPLXXU35IZ` |
| Outgoing webhook — **disabled**, kept for reference (trigger `Alert Group Created`) | `Grafana Assistant - payment-demo` | `WH8F1YZRGHL8PH` |

Severity IDs (discover via `gcx irm incidents severities list`):

| Label | ID |
|-------|-----|
| Critical | `75ac4153-4d2e-11f1-8501-06d3a20484c2` |
| Major (used by `ECX1SI1Q42BJK`) | `75ac44d9-4d2e-11f1-8501-06d3a20484c2` |
| Minor | `75ac4c15-4d2e-11f1-8501-06d3a20484c2` |
| Pending | `75ac3cdb-4d2e-11f1-8501-06d3a20484c2` |

> ⚠️ **Pass severity as the label string (`"Major"`), not the UUID.** The escalation-policy API accepts a UUID and stores it, but the declare-incident worker only resolves the label form — a UUID silently falls back to `Pending` on the resulting incident. The UI normalises to the label on save.

Integration URL (used by the contact point) — token redacted; fetch the real URL with:

```sh
gcx irm oncall integrations list --json spec.verbal_name,spec.integration_url
```

Webhook target URL (auto-set by preset):
`https://wbkprez.grafana.net/api/plugins/grafana-assistant-app/resources/api/v1/investigations/from-irm-incident`

### Auto-declare flow (how an alert becomes an incident)

1. Grafana managed alert routes via contact point `payment-demo` to OnCall integration `CA9SJ5X6ZJFVN`.
2. OnCall opens an alert group. The non-default route `RRS4VY9NEUZT2` (regex `.*`, position 0) catches it before the default route.
3. Chain `FDN21BG5BR5L7` runs its sole policy `ECX1SI1Q42BJK` (step 19 = **Declare Incident**, severity Major) — an IRM incident is created.
4. The outgoing webhook `WHFRBPLXXU35IZ` (trigger `Incident Changed`) fires and POSTs the incident payload to the Assistant's `from-irm-incident` endpoint.
5. The Assistant creates an Investigation attached to the incident.

To recreate the auto-declare wiring from scratch (gcx-able — no UI required):

```sh
# Non-default route (regex .* catches everything routed to this integration)
gcx api /api/plugins/grafana-irm-app/resources/channel_filters/ -X POST -H 'Content-Type: application/json' \
  -d '{"alert_receive_channel":"CA9SJ5X6ZJFVN","escalation_chain":"FDN21BG5BR5L7","filtering_term":".*","filtering_term_type":0}'

# Declare-incident escalation step (step 19, severity Major).
# Use the label "Major" — UUIDs are accepted but silently downgrade the incident to Pending.
gcx api /api/plugins/grafana-irm-app/resources/escalation_policies/ -X POST -H 'Content-Type: application/json' \
  -d '{"escalation_chain":"FDN21BG5BR5L7","step":19,"severity":"Major"}'
```

## What still needs the UI

- **Assistant Skill** — paste `grafana/skills/payment-error-spike.md` into Grafana Assistant → Skills → New Skill. Set **Visible to agents** ON, **Visibility** = Team.
- **Outgoing webhook** — the `grafana_assistant` preset is selected via the UI (IRM → Integrations → Outgoing webhooks → New webhook → pick preset). The preset locks `url` and `http_method`. Once created, the rest of the config (trigger, enabled state) is `gcx api`-patchable.

## End-to-end sanity check

```sh
gcx alert rules list --folder fflqs43gpv7cwa
gcx alert contact-points get cflqs2ow6nugwd
gcx alert notification-policies get -o yaml
gcx irm oncall integrations list
gcx irm oncall escalation-chains list
gcx api /api/plugins/grafana-irm-app/resources/webhooks/WH8F1YZRGHL8PH
```

## Cleanup

```sh
# Notification policy: remove the payment route via UI or `notification-policies set`
gcx alert contact-points delete cflqs2ow6nugwd
# Alert rule: via UI or provisioning API
gcx api /api/plugins/grafana-irm-app/resources/webhooks/WH8F1YZRGHL8PH -X DELETE
gcx api /api/plugins/grafana-irm-app/resources/escalation_chains/FDN21BG5BR5L7 -X DELETE
gcx api /api/plugins/grafana-irm-app/resources/alert_receive_channels/CA9SJ5X6ZJFVN -X DELETE
gcx api /api/folders/fflqs43gpv7cwa -X DELETE
```
