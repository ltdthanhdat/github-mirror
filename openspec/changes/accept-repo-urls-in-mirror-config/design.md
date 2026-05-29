## Context

Mirror creation currently collects `source_owner`, `source_repo`, `target_owner`, and `target_repo`, then the HTTP handler derives clone URLs from those pieces. This works with the existing persistence model, but it forces users to manually split repository information that they already have as GitHub URLs.

The requested change is a contract change at both the HTML form and HTTP API layers. Existing storage fields already contain the normalized URL plus the derived owner and repository names, so the change does not require a schema migration if the server continues storing those derived values.

## Goals / Non-Goals

**Goals:**
- Allow mirror creation to accept `source_url` and `target_url` as the canonical repository inputs.
- Parse and validate GitHub repository URLs on the server.
- Continue storing derived owner, repository, and normalized clone URL fields without changing the database schema.
- Update tests to cover valid and invalid URL-based creation flows.

**Non-Goals:**
- Supporting non-GitHub repository hosts.
- Changing mirror update flows in the same change unless they already share the same creation contract.
- Encrypting tokens as part of this change.
- Redesigning any list/detail pages beyond reflecting the created configuration data.

## Decisions

### Accept URL inputs and derive existing stored fields

The create-mirror request will change to use `source_url` and `target_url` instead of separate owner/repository fields. The handler will parse each URL, extract `owner` and `repo`, and populate the existing `SourceOwner`, `SourceRepo`, `SourceRepoURL`, `TargetOwner`, `TargetRepo`, and `TargetRepoURL` model fields before persistence.

Rationale:
- Keeps the storage model and downstream worker logic stable.
- Minimizes migration risk and keeps the change focused on input handling.

Alternative considered:
- Store only source and target URLs and derive owner/repo later. Rejected because current code and schema already expect the split fields and there is no clear benefit to rewriting that path now.

### Normalize only GitHub repository root URLs

The parser will accept GitHub repository URLs in common forms such as `https://github.com/<owner>/<repo>` and `https://github.com/<owner>/<repo>.git`, then normalize them to `https://github.com/<owner>/<repo>.git` for storage. URLs with other hosts, missing repository segments, or extra path segments will be rejected.

Rationale:
- Keeps validation deterministic and aligned with the product’s GitHub-only positioning.
- Prevents ambiguous parsing of URLs such as branch, issues, or tree links.

Alternative considered:
- Accept any URL and try best-effort parsing. Rejected because silent coercion would make failures harder to understand and test.

### Treat this as a breaking API change unless compatibility is intentionally added

The proposal assumes the canonical request contract becomes `source_url` / `target_url`. If backward compatibility is needed, it should be an explicit follow-up decision rather than an implicit side effect.

Rationale:
- Keeps the handler logic and UI contract simple.
- Avoids carrying two parallel input shapes without a clear requirement.

Alternative considered:
- Accept both the old and new input shapes. Rejected for now because it adds branching and validation complexity to a small UX change.

## Risks / Trade-offs

- [Existing API clients may still send owner/repo fields] -> Mitigation: treat this as a documented breaking change in the proposal/spec and update any internal tests or callers in the same change.
- [Users may paste non-canonical GitHub URLs such as issue or tree pages] -> Mitigation: return clear validation errors for unsupported URL shapes.
- [Parser behavior may drift from user expectations] -> Mitigation: cover accepted and rejected URL examples in tests.

## Migration Plan

1. Update the mirror creation spec to define URL-based inputs.
2. Update the create form and HTTP handler to accept and validate repository URLs.
3. Update automated tests to use the new input contract and cover validation failures.
4. Deploy without schema changes.

Rollback:
- Revert the UI and handler contract to owner/repo inputs if downstream clients cannot migrate in time.

## Open Questions

- Should the update-mirror flow adopt the same URL-based contract in the same change, or stay unchanged until there is a concrete UX need?
- Should validation error responses distinguish unsupported host vs malformed repository path, or is a single generic error sufficient?
