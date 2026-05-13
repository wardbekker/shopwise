# Demo flow — alert to Investigation

End-to-end wiring for the webinar demo on the `wbkprez` Grafana Cloud stack.
Resource IDs are listed in `PROVISIONED.md`.

```mermaid
flowchart TD
    subgraph shop["namespace: shop"]
      PAY[payment service<br/>Beyla auto-instrumented]
    end

    PAY -->|http_server_request_duration_<br/>seconds_count| PROM[(grafanacloud-prom)]

    PROM --> RULE["Alert rule<br/><b>payment error rate high</b><br/>bflqs4pcrmi2oc<br/>5xx ratio &gt; 0.05 for 0s<br/>folder: demo (fflqs43gpv7cwa)<br/>labels: service=payment, severity=critical"]

    RULE -->|fires| NP["Notification policy<br/>route: service = payment<br/>group_wait 0s / interval 1m / repeat 5m"]
    NP --> CP["Contact point<br/><b>payment-demo</b> (oncall)<br/>cflqs2ow6nugwd"]

    CP -->|integration_url| INT["IRM integration<br/><b>payment-demo</b><br/>CA9SJ5X6ZJFVN<br/>(grafana_alerting)"]

    INT --> RT0["Non-default route<br/>RRS4VY9NEUZT2<br/>regex: .*  pos 0"]
    INT -.->|fallback| RTD["Default route<br/>RQ3RQ452XC52X"]

    RT0 --> CHAIN["Escalation chain<br/><b>payment-demo</b><br/>FDN21BG5BR5L7"]
    RTD --> CHAIN

    CHAIN --> ESC["Escalation policy<br/>step 19: <b>Declare Incident</b><br/>severity: Major<br/>ECX1SI1Q42BJK"]

    ESC --> INC[("IRM incident<br/>severity: Major")]

    INC --> WH["Outgoing webhook<br/><b>Grafana Assistant</b><br/>WHFRBPLXXU35IZ<br/>preset: grafana_assistant<br/>trigger_type: 11 (Incident Declared)"]

    WH -->|POST| EP[/"Assistant endpoint<br/>/api/v1/investigations/<br/>from-irm-incident"/]

    EP --> INV["Investigation<br/>(lodestone)<br/>attached to incident"]

    SKILL["Skill: payment-error-spike<br/>grafana/skills/payment-error-spike.md<br/>(UI-loaded, visible to agents)"]
    SKILL -.->|semantic match| INV

    classDef live fill:#1f2937,stroke:#60a5fa,color:#e5e7eb
    classDef manual fill:#3b2f1f,stroke:#f59e0b,color:#fde68a
    class PAY,PROM,RULE,NP,CP,INT,RT0,RTD,CHAIN,ESC,INC,WH,EP,INV live
    class SKILL manual
```

Orange node = needs UI/manual setup. Everything else is provisioned via `gcx`
(see `PROVISIONED.md` for the exact commands and resource IDs).
