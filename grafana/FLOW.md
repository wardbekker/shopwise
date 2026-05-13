# Demo flow — alert to autonomous fix

The story we tell the customer: **Grafana Cloud detects the critical
issue, kicks off an autonomous Investigation, and the Investigation
follows a runbook that tells it how to ship a fix.**

```mermaid
flowchart LR
    APP["🛒 Your application"]
    GC["☁️ <b>Grafana Cloud</b><br/>detects a critical issue"]
    INV["🤖 <b>Autonomous Investigation</b><br/>finds the root cause"]
    RB["📖 Runbook<br/>(how to ship a fix)"]
    PR["✅ <b>Pull Request</b><br/>with the fix"]

    APP -->|telemetry| GC
    GC -->|fires alert| INV
    RB -.->|guides| INV
    INV -->|opens| PR

    classDef cloud fill:#1f2937,stroke:#f59e0b,stroke-width:2px,color:#fde68a
    classDef agent fill:#1e3a5f,stroke:#60a5fa,stroke-width:2px,color:#dbeafe
    classDef artifact fill:#1f2937,stroke:#9ca3af,color:#e5e7eb
    class GC cloud
    class INV agent
    class APP,RB,PR artifact
```

## How it works in one paragraph

The application sends standard OpenTelemetry signals to Grafana Cloud.
When error rates cross a threshold, Grafana Cloud declares an incident
and launches an autonomous Investigation. The Investigation reads a
short runbook the team authored — describing the service, the repo, and
what a good fix looks like — and uses that guidance to analyse the
telemetry, locate the defect in code, and open a pull request with the
proposed fix. A human reviews and merges.

---

<details>
<summary>Full technical wiring (for the engineering audience)</summary>

```mermaid
flowchart TD
    subgraph shop["namespace: shop"]
      PAY[payment service<br/>Beyla auto-instrumented]
    end

    PAY -->|http_server_request_duration_<br/>seconds_count| PROM[(grafanacloud-prom)]
    PROM --> RULE["Alert rule<br/><b>payment error rate high</b><br/>5xx ratio &gt; 0.05"]
    RULE --> NP["Notification policy<br/>route: service = payment"]
    NP --> CP["Contact point<br/><b>payment-demo</b> (oncall)"]
    CP --> INT["IRM integration<br/><b>payment-demo</b>"]
    INT --> CHAIN["Escalation chain<br/>step 19: Declare Incident (Major)"]
    CHAIN --> INC[("IRM incident")]
    INC --> WH["Outgoing webhook<br/>preset: grafana_assistant"]
    WH --> INV["Investigation<br/>attached to incident"]
    SKILL["Skill: payment-error-spike<br/>(runbook)"] -.->|semantic match| INV

    classDef live fill:#1f2937,stroke:#60a5fa,color:#e5e7eb
    classDef manual fill:#3b2f1f,stroke:#f59e0b,color:#fde68a
    class PAY,PROM,RULE,NP,CP,INT,CHAIN,INC,WH,INV live
    class SKILL manual
```

Resource IDs and provisioning commands: `PROVISIONED.md`.

</details>
