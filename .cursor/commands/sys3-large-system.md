Run the unified **Large Distributed System Prompt**.

Use the user-provided input as:
- project description
- functional requirements
- non-functional requirements
- architecture constraints
- scale expectations

Do NOT generate implementation code.

Return the response using these sections:

1. SYSTEM SUMMARY
Explain goals, architecture style, trust boundaries, key quality attributes, and major challenges.

2. ARCHITECTURE OVERVIEW
Describe:
- services/modules
- request flow
- data flow
- deployment topology
- external integrations
- stateful vs stateless components

3. TECHNOLOGY DECISIONS
Explain stack choices and tradeoffs.

4. REPOSITORY STRUCTURE
Include locations for:
- services/apps
- shared packages
- infrastructure
- unit tests
- integration tests
- performance/load tests

5. DOMAIN MODEL

6. DEPENDENCY MAP
Explain sequencing, coupling points, and prerequisites.

7. SECURITY & ABUSE ANALYSIS
Cover:
- authentication/authorization
- injection risks
- secrets handling
- abuse prevention
- rate limiting
- tenant isolation where relevant
- webhook/callback security where relevant
- audit/security logging

8. PERFORMANCE & SCALING ANALYSIS
Cover:
- hot paths
- bottlenecks
- partitioning/sharding
- load balancing
- caching
- concurrency risks
- backpressure handling
- throughput/latency sensitivities

9. DATABASE & INDEXING STRATEGY
Define:
- storage model
- query patterns
- indexing strategy
- retention policy
- migration considerations
- consistency requirements where relevant

10. EDGE CASES & FAILURE MODES
Enumerate:
- duplicate events
- delayed events
- out-of-order events
- partial outages
- dependency failures
- retry storms
- alert duplication / missed processing
- race conditions
- clock skew where relevant

11. TESTING STRATEGY
Define:
- unit tests
- integration tests
- system/end-to-end tests
- regression tests
- security tests
- performance/load tests
- failure-injection tests where relevant

12. INITIAL IMPLEMENTATION PLAN
Create 30–80 sequential steps organized by phase.

Each step must include:
- Step number
- Phase
- Goal
- Files to inspect
- Allowed files
- Disallowed scope
- Requirements
- Security considerations
- Performance considerations
- Edge cases
- Testing requirements
- Dependencies
- Verification

Rules:
- prefer steps modifying no more than 2–3 files
- each step must be executable in one Cursor run
- include unit tests for core components where appropriate
- verification should be deterministic where possible

13. ARCHITECTURE CHALLENGER REVIEW
Stress-test the architecture and identify:
- failure scenarios
- scaling risks
- security weaknesses
- operational complexity
- hidden coupling
- overengineering
Include severity and mitigation.

14. TECH LEAD PLAN REVIEW
Refine the execution plan by fixing:
- oversized steps
- vague steps
- sequencing mistakes
- hidden dependencies
- missing test coverage
- missing deployment/configuration/observability work

15. REVISED EXECUTION PLAN
Each revised step must include:
- Step number
- Phase
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
Define system-level tests validating:
- core workflows
- service interactions
- database behavior
- messaging/event flows
- failure handling
- security-sensitive paths
- performance expectations where relevant

17. CURSOR EXECUTION PROMPTS

Important constraints:
- do NOT generate implementation code
- optimize for Cursor execution by Claude Sonnet
- keep scope narrow for each step
