# Easy Mode Meeting - Quick Reference

## 🎯 What We Built

**UserIdentity annotation with unified format:**
```
system:serviceaccounts:<namespace>:<sa-name>  → ServiceAccount token auth
olm:clusterextension:<extension-name>         → Synthetic impersonation
```

**Code size:** 8 files modified, 2 new files, +184/-56 lines

---

## ✅ Alignment with Joe/Tayler Proposal

> "merge those SA identity and SA namespace fields into something like UserIdentity where the value stored is like 'system:serviceaccounts:namespace:user' or 'olm:clusterextensions:clusterextension'"

**Our implementation:** ✅ EXACTLY THIS!

---

## 🔑 Key Decisions Needed

### 1. Where does UserIdentity live?
- **Option A (current):** Annotation `olm.operatorframework.io/user-identity`
- **Option B:** Spec field `spec.userIdentity`

### 2. Format details?
- Separator: `:` or `/`? (currently `:`)
- Synthetic prefix: `olm:clusterextension:` or `olm:clusterextensions:`? (currently singular)

### 3. Backward compatibility?
- Keep legacy SA annotations? (currently yes)
- Migration timeline?

---

## ❓ Questions for Tayler

1. **What did PR #2429 add?** (fields that conflict)
2. **Your implementation?** (can we compare/merge?)
3. **Boxcutter meeting?** (when, who, what to prepare?)
4. **Format alignment?** (exact string format)
5. **Demo plans?** (joint demo Monday?)

---

## 📋 Examples Ready to Show

### With ServiceAccount:
```yaml
spec:
  serviceAccount:
    name: my-sa
  namespace: ops
```
→ Identity: `system:serviceaccounts:ops:my-sa`
→ Uses: Token authentication

### Without ServiceAccount (Synthetic):
```yaml
spec:
  namespace: ops
  # no serviceAccount
```
→ Identity: `olm:clusterextension:my-operator`
→ Uses: Impersonation as user `olm:clusterextension:my-operator` in group `olm:clusterextensions`

---

## 🚀 Next Steps

**Today:**
- [ ] Compare implementations
- [ ] Review PR #2429
- [ ] Align on format
- [ ] Schedule boxcutter meeting

**Before Meeting:**
- [ ] Finalize format
- [ ] Prepare demo
- [ ] Write proposal for boxcutter

---

## 📁 Code Locations

**Identity abstraction:**
- `internal/operator-controller/authentication/identity.go`
- `internal/operator-controller/authentication/identity_test.go`

**Usage:**
- `internal/operator-controller/applier/boxcutter.go` (sets annotation)
- `internal/operator-controller/controllers/revision_engine_factory.go` (reads annotation)

**Branch:** `easy-mode-install`
