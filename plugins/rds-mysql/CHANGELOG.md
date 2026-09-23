# Changelog

## 0.1.0-beta.1 - 2026-09-23

### Added

- Add the MySQL plugin with 224 commands from 224 APIs captured in the official first-class-node documentation, including its published II-type resource pool family. Cover instances, proxies, accounts, databases, backups, recovery, monitoring, logs, parameters, security, and orders.
- Include Simplified Chinese, British English, and American English help, captured response fixtures, dangerous-operation confirmation, explicit response contracts, local validation of required recovery alternatives, and seven bounded waiters for instance provisioning, stop, destruction, health, idle state, backup completion, and recovery completion.
- Require CTyun core 0.5.0 or later for explicit response contracts and identity-selected waiters.

### Known limitations

- The retired cross-region destination API remains available with a deprecation warning but has no offline success fixture: upstream publishes only a business-error example. Its documented success check and captured error evidence are preserved.
- Preserve published deprecated APIs with warnings; no live cloud calls have been used to verify compatibility.
