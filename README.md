# analitix

`analitix` is being built as a ProjectDiscovery-style Go tool for analytics and tracking-code OSINT.

Current focus:

- keep the repository layout, option naming, and orchestration patterns close to ProjectDiscovery tools
- support three operator-facing modes:
  - `lookup`: live extraction from URL/domain (internal feature of the tool)
  - `history`: historical extraction through providers such as Wayback
  - `reverse`: reverse lookup from discovered or directly supplied tracking IDs
- support direct tracking-ID input for reverse lookup without starting from a domain
- centralize provider selection with `-s` and `-es`, and deduplicate results in the runner
- expose progress logs, per-provider stats, and provider capability listing in a cleaner CLI format
