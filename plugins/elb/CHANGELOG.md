# Changelog

## 0.1.0-beta.1 - 2026-09-23

### Added

- Add 114 commands for all APIs published by Elastic Load Balancing service 88, revision 82, covering classic and guaranteed load balancers, listeners, backend servers and groups, health checks, certificates, forwarding rules, access controls, monitoring, labels, access logs, gateway load balancers and IP listeners.
- Include Chinese and English help, official response fixtures, structured JSON inputs, profile-region fallback, deprecated ID options and confirmation for state changes; validate HTTPS certificates, mutual authentication, backend identity and external-load-balancer inputs.
- Add 12 bounded active/up/health waiters, with exact identity selection for array responses and no invented terminal failure states.

### Known limitations

- No live cloud interoperability has been verified. Legacy VM retrieval lacks documented state semantics, and classic list APIs accept CSV identities rather than an exact scalar selection; these paths do not infer readiness.
- Gateway instance, IP-listener and target-group responses do not document readiness states; asynchronous acceptance alone does not imply completion.
