# Exit codes

`mbs` follows a documented exit-code contract so scripts can react
predictably.

| Code | Name                 | Meaning                                                                 |
| ---- | -------------------- | ----------------------------------------------------------------------- |
| 0    | OK                   | Success.                                                                |
| 1    | Generic              | Unclassified failure.                                                   |
| 2    | Usage                | Bad CLI flags, bad config file, or invalid arguments.                   |
| 3    | Auth                 | Missing/invalid token; permission denied; rejected by remote auth.      |
| 4    | Conflict             | Sync surfaced a conflict that requires user action.                     |
| 5    | Network              | Network or remote API failure (5xx, DNS, timeout).                      |
| 6    | WorkspaceIntegrity   | Workspace file system or `.sync/` state is missing or corrupt.          |
