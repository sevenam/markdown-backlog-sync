# GitHub Issues provider implementation

## Properties
Type
:   Feature
Area
:   GitHub Issues
State
:   Proposed
Priority
:   1
Phase
:   2

## Summary
Implement the `Provider` interface against the GitHub REST + GraphQL API
using `google/go-github` and `shurcooL/githubv4`.

## Scope
- PAT authentication (classic or fine-grained tokens with `issues: rw`).
- Read via REST `GET /repos/{owner}/{repo}/issues?since=<cursor>` and
  GraphQL for fields not exposed by REST.
- Write via REST create/edit endpoints. Use `If-Match` / conditional
  requests where supported; otherwise compare `updated_at` for optimistic
  concurrency.
- Excludes pull requests (filter on `pull_request == null`).

## Acceptance criteria
- Provider contract tests pass with recorded fixtures.
- Pagination handles repos with thousands of issues.
- Honors `X-RateLimit-*` and secondary rate-limit responses with backoff.
- 401/403 surfaced as exit code 3.
