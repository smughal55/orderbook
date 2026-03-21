Run the unified **Existing Repository System Prompt**.

Use the user-provided input as:
- feature request
- repository context
- constraints
- relevant requirements

Do NOT generate implementation code.

Return the response using these sections:

1. FEATURE SUMMARY
Explain the feature and its architectural impact.

2. CURRENT ARCHITECTURE OVERVIEW
Describe relevant modules, system boundaries, request flow, and data flow.

3. DEPENDENCY MAP
Explain module relationships, sequencing, and integration points.

4. SECURITY REVIEW
Cover:
- auth/authz risks
- input validation gaps
- injection risks
- secrets exposure
- abuse scenarios
- rate limiting
- webhook security where applicable

5. PERFORMANCE & SCALING REVIEW
Cover:
- bottlenecks
- hot paths
- caching opportunities
- concurrency risks
- scaling implications

6. DATABASE & INDEXING REVIEW
Cover:
- schema changes
- indexing needs
- migration requirements
- hot query paths
- retention implications where relevant

7. EDGE CASES & FAILURE MODES

8. TESTING STRATEGY
Define:
- unit tests for new/changed logic
- integration tests for interactions with existing services
- regression tests for unchanged behavior
- security tests where relevant
- performance tests where relevant

9. INITIAL IMPLEMENTATION PLAN
Create 10–20 steps.

Each step must include:
- Step number
- Goal
- Files to inspect
- Allowed files
- Disallowed scope
- Requirements
- Security considerations
- Performance considerations
- Edge cases
- Testing requirements
- Verification

Rules:
- prefer steps modifying no more than 2–3 files
- each step must be executable in one Cursor run
- include unit tests where appropriate

10. ARCHITECTURE CHALLENGER REVIEW

11. TECH LEAD PLAN REVIEW

12. REVISED EXECUTION PLAN
Each revised step must include:
- Step number
- Goal
- Files to inspect
- Allowed files
- Disallowed scope
- Requirements
- Security considerations
- Performance considerations
- Edge cases
- Dependencies
- Unit tests where applicable
- Verification

13. FINAL INTEGRATION TEST PLAN

14. CURSOR EXECUTION PROMPTS

Important constraints:
- do NOT generate implementation code
- optimize for Cursor execution by Claude Sonnet
