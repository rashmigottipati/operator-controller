# Easy Mode Install - Sync with Tayler

**Date:** 2026-02-26
**Participants:** Rashmi, Tayler
**Purpose:** Compare implementations, align on approach, prepare for boxcutter meeting

---

## Executive Summary

Both Rashmi and Tayler have been independently implementing the easy-mode install RFC. Our implementations appear to align on the **core solution** proposed by Joe/Tayler:

> "merge those SA identity and SA namespace fields into something like UserIdentity where the value stored is like 'system:serviceaccounts:namespace:user' or 'olm:clusterextensions:clusterextension'"

**Key alignment:** Both implementations use a unified identity string format to support ServiceAccount token auth AND synthetic identity impersonation.

---

## Current Implementation (Rashmi's Branch: `easy-mode-install`)

### 1. UserIdentity Annotation Approach

**What we built:**
```go
// Annotation key
UserIdentityKey = "olm.operatorframework.io/user-identity"

// Identity string formats
"system:serviceaccounts:<namespace>:<sa-name>"  // ServiceAccount case
"olm:clusterextension:<extension-name>"         // Synthetic identity case
```

### 2. Identity Abstraction

**Core struct:**
```go
type Identity struct {
    Type                    IdentityType  // ServiceAccount | Synthetic | Unknown
    String                  string        // Full identity string
    ServiceAccountNamespace string        // Populated for SA type
    ServiceAccountName      string        // Populated for SA type
    ClusterExtensionName    string        // Populated for synthetic type
}
```

**Helper functions:**
```go
// Build identity string from ClusterExtension
BuildIdentity(ext *ClusterExtension) string

// Parse identity string into Identity struct
ParseIdentity(identityString string) (*Identity, error)
```

### 3. Implementation Flow

**Creating ClusterExtensionRevision (boxcutter.go):**
```go
// Set the unified UserIdentity annotation
annotations[labels.UserIdentityKey] = authentication.BuildIdentity(ext)

// Keep deprecated annotations for backward compatibility
if ext.Spec.ServiceAccount.Name != "" {
    annotations[labels.ServiceAccountNameKey] = ext.Spec.ServiceAccount.Name
    annotations[labels.ServiceAccountNamespaceKey] = ext.Spec.Namespace
}
```

**Using ClusterExtensionRevision (revision_engine_factory.go):**
```go
func (f *Factory) CreateRevisionEngine(rev *ClusterExtensionRevision) {
    // 1. Get identity from annotations
    identity := f.getIdentity(rev)  // Tries UserIdentity first, falls back to legacy

    // 2. Create scoped client based on identity type
    client := f.createScopedClient(identity)

    // 3. Return RevisionEngine with scoped client
    return machinery.NewRevisionEngine(..., client)
}
```

**Authentication switching:**
```go
switch identity.Type {
case IdentityTypeServiceAccount:
    // Use TokenInjectingRoundTripper for SA token auth

case IdentityTypeSynthetic:
    // Use ImpersonatingRoundTripper with user/group:
    //   User: "olm:clusterextension:<name>"
    //   Group: "olm:clusterextensions"
}
```

### 4. API Changes

**ClusterExtension API (experimental channel only):**
```go
// ServiceAccount field made optional
ServiceAccount ServiceAccountReference `json:"serviceAccount,omitzero"`

// When empty → synthetic identity
// When set → ServiceAccount token auth
```

### 5. Files Changed

```
8 files changed, 184 insertions(+), 56 deletions(-)

Modified:
- api/v1/clusterextension_types.go
- helm/olmv1/base/operator-controller/crd/experimental/...
- helm/olmv1/base/operator-controller/crd/standard/...
- internal/operator-controller/action/restconfig.go
- internal/operator-controller/action/restconfig_test.go
- internal/operator-controller/applier/boxcutter.go
- internal/operator-controller/controllers/revision_engine_factory.go
- internal/operator-controller/labels/labels.go

New:
- internal/operator-controller/authentication/identity.go (89 LOC)
- internal/operator-controller/authentication/identity_test.go (124 LOC)
```

---

## Key Decision Points for Boxcutter Meeting

### Decision 1: Annotations vs Spec Field

**Option A: Annotations (Current Implementation)**
```go
// ClusterExtensionRevision
metadata:
  annotations:
    olm.operatorframework.io/user-identity: "system:serviceaccounts:ns:sa"
```

**Pros:**
- ✅ No ClusterExtensionRevision CRD changes needed
- ✅ Less invasive to boxcutter
- ✅ Backward compatible with legacy annotations
- ✅ Quick to implement

**Cons:**
- ❌ Boxcutter may want stronger typing
- ❌ Less explicit than spec field

---

**Option B: Spec Field (Boxcutter May Prefer)**
```go
// ClusterExtensionRevision
type ClusterExtensionRevisionSpec struct {
    UserIdentity string `json:"userIdentity,omitempty"`
    // ... existing fields
}
```

**Pros:**
- ✅ Strongly typed in spec
- ✅ More explicit contract
- ✅ May align better with boxcutter's vision

**Cons:**
- ❌ Requires ClusterExtensionRevision CRD changes
- ❌ More coordination with boxcutter team
- ❌ May conflict with PR #2429 changes

---

**Rashmi's recommendation:** Start with annotations (Option A) for MVP, can migrate to spec field later if needed.

---

### Decision 2: Identity String Format

**Current format:**
```
ServiceAccount: "system:serviceaccounts:<namespace>:<sa-name>"
Synthetic:      "olm:clusterextension:<extension-name>"
```

**Questions:**
- Is the separator `:` or `/`? (e.g., `ns:sa` vs `ns/sa`)
- Is the prefix correct? (`system:serviceaccounts:` vs something else)
- Synthetic prefix: `olm:clusterextension:` or `olm:clusterextensions:` (plural)?

**Need alignment with Tayler and boxcutter team.**

---

### Decision 3: Backward Compatibility Strategy

**Current approach:**
```go
// Priority order in getIdentity():
1. Check UserIdentity annotation (new)
2. Fall back to legacy SA name/namespace annotations
3. Fall back to inferring from owner labels

// When creating revisions:
- ALWAYS set UserIdentity annotation
- ALSO set legacy annotations for backward compat (if SA present)
```

**Questions:**
- How long to maintain legacy annotation support?
- Migration path for existing ClusterExtensionRevisions?
- Do we need a migration tool?

---

## Code Examples

### Example 1: ClusterExtension WITH ServiceAccount

**Input:**
```yaml
apiVersion: olm.operatorframework.io/v1
kind: ClusterExtension
metadata:
  name: argocd
spec:
  serviceAccount:
    name: argocd-installer-sa
  namespace: operators
  source:
    sourceType: Catalog
    catalog:
      packageName: argocd-operator
```

**ClusterExtensionRevision Created:**
```yaml
apiVersion: olm.operatorframework.io/v1
kind: ClusterExtensionRevision
metadata:
  annotations:
    olm.operatorframework.io/user-identity: "system:serviceaccounts:operators:argocd-installer-sa"
    # Legacy annotations (backward compat):
    olm.operatorframework.io/service-account-name: "argocd-installer-sa"
    olm.operatorframework.io/service-account-namespace: "operators"
  labels:
    olm.operatorframework.io/owner-name: "argocd"
spec:
  # ... revision spec
```

**Authentication Used:** ServiceAccount token via TokenInjectingRoundTripper

**RBAC Setup:**
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: argocd-installer
subjects:
- kind: ServiceAccount
  name: argocd-installer-sa
  namespace: operators
roleRef:
  kind: ClusterRole
  name: cluster-admin
  apiGroup: rbac.authorization.k8s.io
```

---

### Example 2: ClusterExtension WITHOUT ServiceAccount (Synthetic)

**Input:**
```yaml
apiVersion: olm.operatorframework.io/v1
kind: ClusterExtension
metadata:
  name: prometheus
spec:
  namespace: monitoring
  # serviceAccount field OMITTED (experimental only)
  source:
    sourceType: Catalog
    catalog:
      packageName: prometheus-operator
```

**ClusterExtensionRevision Created:**
```yaml
apiVersion: olm.operatorframework.io/v1
kind: ClusterExtensionRevision
metadata:
  annotations:
    olm.operatorframework.io/user-identity: "olm:clusterextension:prometheus"
    # No legacy SA annotations
  labels:
    olm.operatorframework.io/owner-name: "prometheus"
spec:
  # ... revision spec
```

**Authentication Used:** Impersonation with:
- User: `olm:clusterextension:prometheus`
- Groups: `["olm:clusterextensions"]`

**RBAC Setup (grant to specific extension):**
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: prometheus-extension-admin
subjects:
- kind: User
  name: "olm:clusterextension:prometheus"
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: cluster-admin
  apiGroup: rbac.authorization.k8s.io
```

**RBAC Setup (grant to ALL extensions):**
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: all-extensions-admin
subjects:
- kind: Group
  name: "olm:clusterextensions"
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: cluster-admin
  apiGroup: rbac.authorization.k8s.io
```

---

## Questions for Tayler

### 1. Implementation Comparison
- [ ] What does your implementation look like?
- [ ] Are you using annotations or spec fields?
- [ ] Do we have the same identity string format?
- [ ] Can we merge our implementations?

### 2. PR #2429 Details
- [ ] What fields did boxcutter add in PR #2429?
- [ ] Are they spec fields or annotations?
- [ ] How does it conflict with easy-mode?
- [ ] Can we see the PR diff?

### 3. Format Alignment
- [ ] Confirm identity string format: `system:serviceaccounts:<ns>:<sa>`?
- [ ] Separator: colon `:` or slash `/`?
- [ ] Synthetic prefix: `olm:clusterextension:` (singular) or `olm:clusterextensions:` (plural)?

### 4. Boxcutter Meeting
- [ ] When is the meeting scheduled?
- [ ] Who needs to attend? (OLM team + boxcutter team)
- [ ] What decisions need to be made?
- [ ] What do we need to prepare/demo?

### 5. Migration Strategy
- [ ] How to handle existing ClusterExtensionRevisions?
- [ ] Do we need a migration tool?
- [ ] Timeline for deprecating legacy annotations?

### 6. Demo Plans
- [ ] Can we do a joint demo on Monday?
- [ ] What should we showcase?
- [ ] Do we need e2e tests working first?

---

## Technical Risks & Concerns

### 1. Boxcutter Compatibility
**Risk:** PR #2429 may have introduced fields that conflict with our approach.

**Mitigation:**
- Need to review PR #2429 before merging
- Coordinate with boxcutter team in upcoming meeting
- May need to refactor based on their requirements

### 2. Identity String Format
**Risk:** If format isn't agreed upon, we may need to change it later.

**Mitigation:**
- Align with Tayler and Joe on exact format
- Document the format clearly
- Add validation/parsing tests

### 3. Backward Compatibility
**Risk:** Breaking existing ClusterExtensionRevisions.

**Mitigation:**
- Fall back to legacy annotations if UserIdentity not present
- Keep legacy annotations during migration period
- Clear deprecation timeline

### 4. RBAC Complexity
**Risk:** Users may not understand synthetic identity RBAC setup.

**Mitigation:**
- Clear documentation with examples
- Error messages that explain RBAC requirements
- Example ClusterRoleBindings in docs

---

## Next Steps

### Immediate (Today's Sync)
1. ✅ Compare implementations with Tayler
2. ✅ Review PR #2429 together
3. ✅ Align on identity string format
4. ✅ Decide on annotations vs spec field
5. ✅ Plan for boxcutter meeting

### Before Boxcutter Meeting
1. ⬜ Finalize identity format with OLM team
2. ⬜ Prepare demo/examples
3. ⬜ Document RBAC setup clearly
4. ⬜ Write up proposal for boxcutter team

### After Alignment
1. ⬜ Merge/coordinate implementations
2. ⬜ Write e2e tests
3. ⬜ Update documentation
4. ⬜ Demo on Monday (if ready)

---

## Appendix: Code Reference

### Identity Builder (authentication/identity.go:44-51)
```go
func BuildIdentity(ext *ocv1.ClusterExtension) string {
    if ext.Spec.ServiceAccount.Name == "" {
        // Synthetic identity
        return fmt.Sprintf("%s%s", SyntheticIdentityPrefix, ext.Name)
    }
    // ServiceAccount identity
    return fmt.Sprintf("%s%s:%s", ServiceAccountIdentityPrefix,
        ext.Spec.Namespace, ext.Spec.ServiceAccount.Name)
}
```

### Identity Parser (authentication/identity.go:54-89)
```go
func ParseIdentity(identityString string) (*Identity, error) {
    // Validates format and extracts:
    // - Type (ServiceAccount or Synthetic)
    // - ServiceAccount namespace/name (if SA)
    // - ClusterExtension name (if synthetic)
    // Returns error for invalid format
}
```

### Scoped Client Creation (revision_engine_factory.go:129-172)
```go
func (f *Factory) createScopedClient(identity *Identity) (client.Client, error) {
    switch identity.Type {
    case IdentityTypeSynthetic:
        // Use impersonation
    case IdentityTypeServiceAccount:
        // Use token injection
    }
}
```

---

## Questions? Concerns?

**Contact:**
- Rashmi: Slack @rashmi
- Implementation branch: `easy-mode-install`
- Temp commit SHA: 5e6fc14 (has all the context)

**References:**
- Easy Mode RFC: [Link if available]
- PR #2429: https://github.com/operator-framework/operator-controller/pull/2429
- Joe's reference PR: [Link if available]
