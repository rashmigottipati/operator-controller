# Easy Mode Install - Implementation Documentation

This directory contains documentation for the "Easy Mode Install" feature implementation, which allows ClusterExtensions to optionally use synthetic identity impersonation instead of requiring a ServiceAccount.

## Documents

### [sync-notes.md](./sync-notes.md)
**Comprehensive sync document for team alignment**

Detailed documentation covering:
- Current implementation overview
- UserIdentity annotation approach
- Code examples (with and without ServiceAccount)
- Decision points for boxcutter meeting
- Questions for team discussion
- Technical risks and mitigations
- RBAC setup examples

**Use for:** Preparation, team meetings, design reviews

---

### [quick-reference.md](./quick-reference.md)
**One-page quick reference**

Quick lookup guide containing:
- What we built (summary)
- Key decisions needed
- Questions for Tayler
- Examples ready to demo
- Next steps checklist

**Use for:** During meetings, quick reference

---

## Feature Overview

**Goal:** Make ServiceAccount optional on experimental channel. When omitted, use synthetic identity impersonation.

**Implementation:**
- **UserIdentity annotation:** Unified format for both ServiceAccount and synthetic identities
- **Identity formats:**
  - ServiceAccount: `system:serviceaccounts:<namespace>:<sa-name>`
  - Synthetic: `olm:clusterextension:<extension-name>`
- **Authentication:** Automatically switches between token auth (SA) and impersonation (synthetic)

**Status:** ✅ Implementation complete, ready for team sync and boxcutter alignment

---

## Key Files

**Identity abstraction:**
- `internal/operator-controller/authentication/identity.go`
- `internal/operator-controller/authentication/identity_test.go`

**Implementation:**
- `internal/operator-controller/applier/boxcutter.go` - Creates UserIdentity annotation
- `internal/operator-controller/controllers/revision_engine_factory.go` - Reads UserIdentity and creates scoped client
- `internal/operator-controller/action/restconfig.go` - Action layer authentication

**API:**
- `api/v1/clusterextension_types.go` - ServiceAccount made optional (experimental)

---

## Branch

**Current work:** `easy-mode-install`

**Stats:** 8 files modified, 2 new files, +184/-56 lines

---

## Next Steps

1. ✅ Sync with Tayler on implementation
2. ⬜ Review PR #2429 (boxcutter compatibility)
3. ⬜ Align on identity format with team
4. ⬜ Boxcutter team meeting
5. ⬜ Write e2e tests
6. ⬜ Prepare demo

---

## Contact

**Owner:** Rashmi (@rashmi)
**Reviewers:** Tayler, Joe
**Related:** PR #2429, Easy Mode RFC
