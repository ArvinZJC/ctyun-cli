# ZOS public API review — 2026-09-24

The public inventory for product `sid=9`, catalog `data=105`, revision `vid=99` still contains the 106 tracked operations. Comparing all captured documentation sections with the earlier captures, excluding update timestamps, found 85 unchanged documents and 21 changed documents. All changed documents belong to `/v4/zms/`; the URI scope, operation count, and command count remain unchanged.

All 21 assessment, migration, and agent documents now include North China 2 alongside East China 1. The additional contract changes are:

- [Create migration, API 17827](https://eop.ctyun.cn/ebp/ctapiDocument/search?sid=9&api=17827&data=105&isNormal=1&vid=99): `rateLimitType`, `rateLimitNum`, `rateLimitPolicy`, and `consistencyCheck`; nested `Regex` and `File_list` source modes; `destinationPrefix` constraints. The complete request example is unchanged. The policy parameter example is captured from the parameter table, not constructed from an invented request.
- [Migration details, API 17829](https://eop.ctyun.cn/ebp/ctapiDocument/search?sid=9&api=17829&data=105&isNormal=1&vid=99): matching rate-limit and consistency response fields, plus nested `migrateRegex` and `migrateFileList` arrays. The captured response remains an object at `returnObj.result`.
- [Migration history, API 19873](https://eop.ctyun.cn/ebp/ctapiDocument/search?sid=9&api=19873&data=105&isNormal=1&vid=99): `errorColumn`, which identifies the failed object-list parsing line and is null in the captured response.

The CLI enforces the Global/Period conditional inputs, typed integer/Boolean/JSON inputs, rate-limit type choices, and the documented global range of 50–500000 Mbps. The API remains responsible for validating that enabled rate limiting uses semi-managed migration, the JSON policy's integer values and fallback entry, its maximum ten non-overlapping time ranges, and nested source/destination constraints. Help preserves these constraints. JSON objects do not reliably express a semantically significant row order, although the portal asks for `else` first; no additional ordering guarantee is invented.

All four existing waiters retain their bindings and terminal-state rules. The refreshed documents add no terminal states or safe new waiter targets. In particular, migration details still use `returnObj.result.migrationStatus`; the captured `create_failed` response remains a failure, and `agent_lost` remains bounded pending because the documentation does not identify it as a terminal result.

An unchanged upstream discrepancy remains in [agent listing, API 22334](https://eop.ctyun.cn/ebp/ctapiDocument/search?sid=9&api=22334&data=105&isNormal=1&vid=99): the response table spells `workerAvailableCount`, while its example spells `workeravailableCount`. The existing captured example is preserved. This review does not guess a response alias or add a table column with an unverified spelling.
