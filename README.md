# webinar-demo

E-commerce microservices on k3d, instrumented end-to-end with Grafana Cloud
(Beyla metrics, Loki logs, Tempo traces, IRM, Assistant). The repo doubles as
a demo stage: a flippable fault in the `payment` service drives a real alert
through Grafana managed alerting → IRM → Grafana Assistant Investigation.

## What's in here

- `services/` — Go microservices (`frontend`, `cart`, `checkout`, `payment`,
  `order`, `product-catalog`, `loadgen`)
- `deploy/` — k3d cluster + k8s manifests + Synthetic Monitoring probe & checks
- `grafana/` — Grafana Cloud assets that back the demo
  - `manifests/` — sanitised source-of-truth YAML for the alert rule, contact
    point, notification policy, OnCall integration, escalation chain, and the
    Grafana-Assistant outgoing webhook
  - `skills/payment-error-spike.md` — Assistant Skill content (UI-only —
    paste into Grafana Assistant → Skills)
  - `PROVISIONED.md` — live IDs and UIDs in the `wbkprez` stack
- `DEMO.md` — the end-to-end narrative (alert → Investigation → Skill → PR)
- `CLAUDE.md` — repo rules for AI coding agents

## Bringing the stack up

```sh
make up                # k3d cluster + build all images + deploy + wait ready
make monitoring-install # Grafana Cloud onboarding (needs .env, see .env.example)
make sm-probe-install  # private Synthetic Monitoring probe (optional)
```

## Running the demo

Prereqs: cluster is `make up`, `gcx` context is `wbkprez`, the Assistant Skill
in `grafana/skills/payment-error-spike.md` is pasted into the UI (one-time).

Trigger the fault:

```sh
kubectl -n shop set env deploy/payment FAIL_RATE=0.5
```

Clear the fault:

```sh
kubectl -n shop set env deploy/payment FAIL_RATE-
```

Expected timing (steady-state load from `loadgen`):

| t (approx) | Event |
|------------|-------|
| 0:00 | `FAIL_RATE=0.5` set, pod rolls |
| 1:00–2:00 | Beyla discovers the new pod; 5xx appear in metrics |
| 2:30 | Ratio crosses 5% threshold |
| 3:30 | Alert transitions `inactive → pending` (then `→ firing` after `for: 1m`) |
| 3:45 | IRM alert group created, `grafana_assistant` webhook fires |
| 3:46 | Assistant Investigation auto-created with the alert context |

After clearing the fault, the alert resolves automatically once the 5xx samples
drain out of the 5-minute rate window (~5 min).

### Starting a Claude Code session

Open Claude Code in this repo and paste:

> Run the payment-error demo scenario. `grafana/PROVISIONED.md` lists the
> live Grafana Cloud assets; `DEMO.md` describes the flow. The fault toggle
> is `kubectl -n shop set env deploy/payment FAIL_RATE=0.5` (clear with
> `FAIL_RATE-`). Verify the cluster and gcx context `wbkprez` are healthy,
> then watch the payment 5xx ratio and the alert state, and walk me through
> alert → IRM → Investigation.

Claude reads `CLAUDE.md` on startup, then the three pointers above are enough
to operate the demo.

## Cleanup

```sh
kubectl -n shop set env deploy/payment FAIL_RATE-   # ensure fault is off
make down                                            # tear down the cluster
# Grafana Cloud assets: see grafana/PROVISIONED.md for the delete commands.
```
