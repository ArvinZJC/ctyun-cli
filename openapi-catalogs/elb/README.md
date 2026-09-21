# Elastic Load Balancing source evidence

The catalog captures all 114 APIs published by product `sid=24`, service `88`, revision `82`, including the legacy group and the gateway load balancer and IP-listener groups. The declared URI prefixes are `/v4/elb/`, `/v4/gwlb/` and `/v4/iplistener/`. The official endpoint is `https://ctelb-global.ctapi.ctyun.cn`.

`upstream-evidence.json` preserves service/version discovery, endpoint and status documentation, the complete overview and each API document. `coverage.json` maps every published API to its command and records example punctuation normalization. `source.json` is the normalized execution contract; `baseline.json` records the promoted contract. Every fixture comes from a captured successful response; malformed structural fullwidth or trailing commas are repaired without changing field values.

All documented root request parameters remain available, including structured arrays and objects and deprecated generic ID fields. API 8271 declares `healthCheck` as an array of objects while its request example supplies an object. The declared array contract is retained, and that invalid example is not presented as valid input. Unknown nested object members are passed through without inventing schemas.

Twelve waiters cover documented active/up states and IPv4/IPv6 health states. Array-valued show responses select exactly the requested modern identity; gateway target listing requires its scalar target identity. DOWN, INACTIVE, offline, unknown and null remain pending because the docs do not establish terminal failure. Legacy VM/group state semantics are missing; classic list filters accept CSV IDs; gateway instance, IP-listener and target-group responses have no readiness state. Those gaps do not produce speculative waiters.

The bundle is beta/generated. Pipeline review, fixture decoding and local request tests establish the captured metadata contract; no live cloud requests or production interoperability claims are made.
