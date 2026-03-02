package authentication

import (
	"fmt"
	"strings"

	ocv1 "github.com/operator-framework/operator-controller/api/v1"
)

const (
	// ServiceAccountIdentityPrefix is the prefix for ServiceAccount-based identities
	ServiceAccountIdentityPrefix = "system:serviceaccounts:"
	// SyntheticIdentityPrefix is the prefix for synthetic user identities
	SyntheticIdentityPrefix = "olm:clusterextension:"
)

// IdentityType represents the type of identity used for authentication
type IdentityType int

const (
	// IdentityTypeServiceAccount indicates token-based ServiceAccount authentication
	IdentityTypeServiceAccount IdentityType = iota
	// IdentityTypeSynthetic indicates synthetic user impersonation
	IdentityTypeSynthetic
	// IdentityTypeUnknown indicates an unrecognized identity format
	IdentityTypeUnknown
)

// Identity represents a user identity for authentication, either ServiceAccount or synthetic
type Identity struct {
	// Type indicates whether this is a ServiceAccount or synthetic identity
	Type IdentityType
	// String is the full identity string (e.g., "system:serviceaccounts:ns:sa" or "olm:clusterextension:name")
	String string
	// ServiceAccountNamespace is populated for ServiceAccount identities
	ServiceAccountNamespace string
	// ServiceAccountName is populated for ServiceAccount identities
	ServiceAccountName string
	// ClusterExtensionName is populated for synthetic identities
	ClusterExtensionName string
}

// BuildIdentity creates an identity string from a ClusterExtension
// Returns the identity string suitable for annotation storage
func BuildIdentity(ext *ocv1.ClusterExtension) string {
	if ext.Spec.ServiceAccount.Name == "" {
		// Synthetic identity
		return fmt.Sprintf("%s%s", SyntheticIdentityPrefix, ext.Name)
	}
	// ServiceAccount identity
	return fmt.Sprintf("%s%s:%s", ServiceAccountIdentityPrefix, ext.Spec.Namespace, ext.Spec.ServiceAccount.Name)
}

// ParseIdentity parses an identity string and returns an Identity struct
func ParseIdentity(identityString string) (*Identity, error) {
	if identityString == "" {
		return nil, fmt.Errorf("identity string is empty")
	}

	identity := &Identity{
		String: identityString,
	}

	// Check for ServiceAccount identity
	if strings.HasPrefix(identityString, ServiceAccountIdentityPrefix) {
		identity.Type = IdentityTypeServiceAccount
		// Format: system:serviceaccounts:<namespace>:<sa-name>
		parts := strings.TrimPrefix(identityString, ServiceAccountIdentityPrefix)
		namespaceParts := strings.SplitN(parts, ":", 2)
		if len(namespaceParts) != 2 {
			return nil, fmt.Errorf("invalid ServiceAccount identity format: %q, expected 'system:serviceaccounts:<namespace>:<name>'", identityString)
		}
		identity.ServiceAccountNamespace = namespaceParts[0]
		identity.ServiceAccountName = namespaceParts[1]
		return identity, nil
	}

	// Check for synthetic identity
	if strings.HasPrefix(identityString, SyntheticIdentityPrefix) {
		identity.Type = IdentityTypeSynthetic
		identity.ClusterExtensionName = strings.TrimPrefix(identityString, SyntheticIdentityPrefix)
		return identity, nil
	}

	// Unknown format
	identity.Type = IdentityTypeUnknown
	return identity, fmt.Errorf("unrecognized identity format: %q, must start with %q or %q",
		identityString, ServiceAccountIdentityPrefix, SyntheticIdentityPrefix)
}
