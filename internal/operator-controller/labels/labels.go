package labels

const (
	// OwnerKindKey is the label key used to record the kind of the owner
	// resource responsible for creating or managing a ClusterExtensionRevision.
	OwnerKindKey = "olm.operatorframework.io/owner-kind"

	// OwnerNameKey is the label key used to record the name of the owner
	// resource responsible for creating or managing a ClusterExtensionRevision.
	OwnerNameKey = "olm.operatorframework.io/owner-name"

	// PackageNameKey is the label key used to record the package name
	// associated with a ClusterExtensionRevision.
	PackageNameKey = "olm.operatorframework.io/package-name"

	// BundleNameKey is the label key used to record the bundle name
	// associated with a ClusterExtensionRevision.
	BundleNameKey = "olm.operatorframework.io/bundle-name"

	// BundleVersionKey is the label key used to record the bundle version
	// associated with a ClusterExtensionRevision.
	BundleVersionKey = "olm.operatorframework.io/bundle-version"

	// BundleReferenceKey is the label key used to record an external reference
	// (such as an image or catalog reference) to the bundle for a
	// ClusterExtensionRevision.
	BundleReferenceKey = "olm.operatorframework.io/bundle-reference"

	// UserIdentityKey is the annotation key used to record the user identity
	// used for managing the ClusterExtensionRevision. It is applied as an
	// annotation on ClusterExtensionRevision resources.
	//
	// The identity string format varies based on the authentication method:
	//   - ServiceAccount token auth: "system:serviceaccounts:<namespace>:<sa-name>"
	//   - Synthetic identity: "olm:clusterextension:<extension-name>"
	//
	// This unified format allows the RevisionEngineFactory to determine the
	// appropriate authentication mechanism and supports both traditional
	// ServiceAccount-based authentication and synthetic identity impersonation.
	UserIdentityKey = "olm.operatorframework.io/user-identity"

	// DEPRECATED: ServiceAccountNameKey is deprecated in favor of UserIdentityKey.
	// Kept for backward compatibility during migration.
	ServiceAccountNameKey = "olm.operatorframework.io/service-account-name"

	// DEPRECATED: ServiceAccountNamespaceKey is deprecated in favor of UserIdentityKey.
	// Kept for backward compatibility during migration.
	ServiceAccountNamespaceKey = "olm.operatorframework.io/service-account-namespace"

	// MigratedFromHelmKey is the label key used to mark ClusterExtensionRevisions
	// that were created during migration from Helm releases. This label is used
	// to distinguish migrated revisions from those created by normal Boxcutter operation.
	MigratedFromHelmKey = "olm.operatorframework.io/migrated-from-helm"
)
