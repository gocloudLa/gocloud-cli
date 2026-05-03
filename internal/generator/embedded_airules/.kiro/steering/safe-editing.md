---
description: Minimize write commands and forbid mass file edits without explicit approval.
alwaysApply: true
---

# Safe editing policy

- Default to read-only shell usage.
- Never perform mass file modifications with commands/scripts (`python`, `sed`, `awk`, `perl`, `xargs`, loops, bulk replace).
- Never run write-capable shell commands unless the user explicitly authorizes them in the current conversation.
- Treat as write-capable: redirections (`>`/`>>`), `sed -i`, script-generated writes, `mv`, `cp` (when writing), `rm`, `mkdir`, generators, and autofix format/lint commands.
- Prefer targeted per-file edits via patch workflow.
- If scope expands beyond the approved files/commands, stop and request a new authorization.
