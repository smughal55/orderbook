Run the unified **Greenfield System Prompt**.

Use the user-provided input as:
- project description
- functional requirements
- non-functional requirements
- constraints

Do NOT generate implementation code.

Return the response using these sections:

1. SYSTEM OVERVIEW
Explain:
- major components
- request flow
- data flow
- trust boundaries
- key technical risks

2. TECHNOLOGY STACK
Explain:
- language/runtime
- backend framework
- frontend framework if relevant
- database
- messaging systems if relevant
- observability tools
- testing frameworks
- tradeoffs

3. REPOSITORY STRUCTURE
Design the layout and include locations for:
- source code
- configuration
- unit tests
- integration tests
- infrastructure

4. DOMAIN MODEL
Define core entities and relationships.

5. DEPENDENCY MAP
Describe module interactions, sequencing, and coupling points.

6. SECURITY ANALYSIS
Cover:
- authentication / authorization
- input validation
- injection risks
- secrets handling
- webhook security if applicable
- abuse scenarios
- rate limiting
- tenant isolation if applicable

7. PERFORMANCE & SCALING ANALYSIS
Cover:
- hot paths
- throughput bottlenecks
- latency-sensitive flows
- load balancing strategy
- partitioning/sharding strategy
- caching opportunities
- concurrency risks
- backpressure handling where relevant

8. DATABASE & INDEXING STRATEGY
Define:
- schema design
- query patterns
- indexing strategy
- retention/archival
- migration strategy

9. EDGE CASES & FAILURE MODES
Enumerate:
- malformed inputs
- duplicate events
- delayed events
- dependency failures
- partial system failures
- retry storms
- race conditions

10. TESTING STRATEGY
Define:
- unit tests
- integration tests
- regression tests where relevant
- security tests where relevant
- performance tests where relevant

11. INITIAL IMPLEMENTATION PLAN
Create 15–25 steps.

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
- verification should be deterministic where possible

12. ARCHITECTURE CHALLENGER REVIEW
Critically challenge the architecture and identify:
- failure scenarios
- scaling risks
- security weaknesses
- operational complexity
- unnecessary complexity
Include severity and mitigation.

13. TECH LEAD PLAN REVIEW
Review the implementation plan and fix:
- oversized steps
- sequencing issues
- hidden dependencies
- missing test coverage
- missing deployment/configuration/observability work

14. REVISED ARCHITECTURE

15. REVISED EXECUTION PLAN
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

16. FINAL INTEGRATION TEST PLAN

17. CURSOR EXECUTION PROMPTS
Generate ready-to-use prompts for each step.

Important constraints:
- do NOT generate implementation code
- optimize for Cursor execution by Claude Sonnet
- keep step scope narrow
