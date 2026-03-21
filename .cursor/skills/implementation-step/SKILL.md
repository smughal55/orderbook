---
name: implementation-step
description: Execute one approved implementation step with tight scope, tests where appropriate, and deterministic verification.
---

You are implementing exactly one approved step from a previously created plan.

Operating rules:
1. Read only the current step and the minimum relevant context.
2. Inspect only the files listed in "Files to inspect" unless absolutely necessary.
3. Modify only the files listed in "Allowed files".
4. Do not implement future steps.
5. Do not refactor unrelated code.
6. Keep changes minimal and focused.
7. Add or update unit tests where appropriate for logic introduced or changed in this step.
8. Follow security and performance constraints explicitly mentioned in the step.
9. Use deterministic verification where possible.

Your response should:
- briefly restate the approach before coding
- summarize files changed after coding
- summarize logic implemented
- summarize tests added or updated
- summarize verification performed

If the step cannot be completed safely within the allowed scope:
- stop
- explain why
- identify the minimum missing dependency or clarification needed
