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
| Escalation chain | `payment-demo` | `FDN21BG5BR5L7` |
| Outgoing webhook (preset `grafana_assistant`, trigger `Alert Group Created`) | `Grafana Assistant - payment-demo` | `WH8F1YZRGHL8PH` |

Integration URL (used by the contact point) — token redacted; fetch the real URL with:

```sh
gcx irm oncall integrations list --json spec.verbal_name,spec.integration_url
```

Webhook target URL (auto-set by preset):
`https://wbkprez.grafana.net/api/plugins/grafana-assistant-app/resources/api/v1/investigations/from-irm`

## What still needs the UI

- **Assistant Skill** — paste `grafana/skills/payment-error-spike.md` into Grafana Assistant → Skills → New Skill. Set **Visible to agents** ON, **Visibility** = Team.
- Optional: add a second outgoing webhook with **trigger type "Status change"** if you want investigation context to refresh as the alert evolves. The current webhook only triggers on initial alert group creation.

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
