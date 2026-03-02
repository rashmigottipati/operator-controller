//go:build !standard

// This file is excluded from standard builds because ClusterExtensionRevision
// is an experimental feature. Standard builds use Helm-based applier only.
// The experimental build includes BoxcutterRuntime which requires these factories
// for serviceAccount-scoped client creation and RevisionEngine instantiation.

package controllers

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/transport"
	"pkg.package-operator.run/boxcutter/machinery"
	machinerytypes "pkg.package-operator.run/boxcutter/machinery/types"
	"pkg.package-operator.run/boxcutter/managedcache"
	"pkg.package-operator.run/boxcutter/ownerhandling"
	"pkg.package-operator.run/boxcutter/validation"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ocv1 "github.com/operator-framework/operator-controller/api/v1"
	"github.com/operator-framework/operator-controller/internal/operator-controller/authentication"
	"github.com/operator-framework/operator-controller/internal/operator-controller/labels"
)

// RevisionEngine defines the interface for reconciling and tearing down revisions.
type RevisionEngine interface {
	Teardown(ctx context.Context, rev machinerytypes.Revision, opts ...machinerytypes.RevisionTeardownOption) (machinery.RevisionTeardownResult, error)
	Reconcile(ctx context.Context, rev machinerytypes.Revision, opts ...machinerytypes.RevisionReconcileOption) (machinery.RevisionResult, error)
}

// RevisionEngineFactory creates a RevisionEngine for a ClusterExtensionRevision.
type RevisionEngineFactory interface {
	CreateRevisionEngine(ctx context.Context, rev *ocv1.ClusterExtensionRevision) (RevisionEngine, error)
}

// defaultRevisionEngineFactory creates boxcutter RevisionEngines with serviceAccount-scoped clients.
type defaultRevisionEngineFactory struct {
	Scheme                      *runtime.Scheme
	TrackingCache               managedcache.TrackingCache
	DiscoveryClient             discovery.CachedDiscoveryInterface
	RESTMapper                  meta.RESTMapper
	FieldOwnerPrefix            string
	BaseConfig                  *rest.Config
	TokenGetter                 *authentication.TokenGetter
	SyntheticPermissionsEnabled bool
}

// CreateRevisionEngine constructs a boxcutter RevisionEngine for the given ClusterExtensionRevision.
// It reads the UserIdentity annotation and creates a scoped client with the appropriate authentication.
func (f *defaultRevisionEngineFactory) CreateRevisionEngine(_ context.Context, rev *ocv1.ClusterExtensionRevision) (RevisionEngine, error) {
	identity, err := f.getIdentity(rev)
	if err != nil {
		return nil, err
	}

	scopedClient, err := f.createScopedClient(identity)
	if err != nil {
		return nil, err
	}

	return machinery.NewRevisionEngine(
		machinery.NewPhaseEngine(
			machinery.NewObjectEngine(
				f.Scheme, f.TrackingCache, scopedClient,
				ownerhandling.NewNative(f.Scheme),
				machinery.NewComparator(ownerhandling.NewNative(f.Scheme), f.DiscoveryClient, f.Scheme, f.FieldOwnerPrefix),
				f.FieldOwnerPrefix, f.FieldOwnerPrefix,
			),
			validation.NewClusterPhaseValidator(f.RESTMapper, scopedClient),
		),
		validation.NewRevisionValidator(), scopedClient,
	), nil
}

// getIdentity extracts the user identity from the revision's annotations.
// It supports both the new UserIdentity annotation and legacy ServiceAccount annotations for backward compatibility.
func (f *defaultRevisionEngineFactory) getIdentity(rev *ocv1.ClusterExtensionRevision) (*authentication.Identity, error) {
	annotations := rev.GetAnnotations()
	if annotations == nil {
		return nil, fmt.Errorf("revision %q is missing annotations", rev.Name)
	}

	// Try the new UserIdentity annotation first
	if identityStr, ok := annotations[labels.UserIdentityKey]; ok && identityStr != "" {
		identity, err := authentication.ParseIdentity(identityStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse user identity for revision %q: %w", rev.Name, err)
		}
		return identity, nil
	}

	// Fall back to legacy ServiceAccount annotations for backward compatibility
	saName := strings.TrimSpace(annotations[labels.ServiceAccountNameKey])
	saNamespace := strings.TrimSpace(annotations[labels.ServiceAccountNamespaceKey])

	if saName != "" && saNamespace != "" {
		// Legacy ServiceAccount identity
		return &authentication.Identity{
			Type:                    authentication.IdentityTypeServiceAccount,
			String:                  fmt.Sprintf("%s%s:%s", authentication.ServiceAccountIdentityPrefix, saNamespace, saName),
			ServiceAccountNamespace: saNamespace,
			ServiceAccountName:      saName,
		}, nil
	}

	// If we have owner labels, we can infer synthetic identity
	revLabels := rev.GetLabels()
	if revLabels != nil {
		if ownerName := strings.TrimSpace(revLabels[labels.OwnerNameKey]); ownerName != "" {
			return &authentication.Identity{
				Type:                 authentication.IdentityTypeSynthetic,
				String:               fmt.Sprintf("%s%s", authentication.SyntheticIdentityPrefix, ownerName),
				ClusterExtensionName: ownerName,
			}, nil
		}
	}

	return nil, fmt.Errorf("revision %q is missing identity annotations", rev.Name)
}

// createScopedClient creates a client with the appropriate authentication based on the identity type.
// For ServiceAccount identities, it uses token-based authentication.
// For synthetic identities, it uses Kubernetes impersonation.
func (f *defaultRevisionEngineFactory) createScopedClient(identity *authentication.Identity) (client.Client, error) {
	var scopedConfig *rest.Config

	switch identity.Type {
	case authentication.IdentityTypeSynthetic:
		// Use synthetic identity impersonation
		scopedConfig = rest.CopyConfig(f.BaseConfig)
		scopedConfig.Wrap(func(rt http.RoundTripper) http.RoundTripper {
			// Create a minimal ClusterExtension object for SyntheticImpersonationConfig
			ext := &ocv1.ClusterExtension{}
			ext.Name = identity.ClusterExtensionName
			return transport.NewImpersonatingRoundTripper(authentication.SyntheticImpersonationConfig(*ext), rt)
		})

	case authentication.IdentityTypeServiceAccount:
		// Use token-based authentication with ServiceAccount
		scopedConfig = rest.AnonymousClientConfig(f.BaseConfig)
		scopedConfig.Wrap(func(rt http.RoundTripper) http.RoundTripper {
			return &authentication.TokenInjectingRoundTripper{
				Tripper:     rt,
				TokenGetter: f.TokenGetter,
				Key: types.NamespacedName{
					Name:      identity.ServiceAccountName,
					Namespace: identity.ServiceAccountNamespace,
				},
			}
		})

	default:
		return nil, fmt.Errorf("unsupported identity type: %v", identity.Type)
	}

	scopedClient, err := client.New(scopedConfig, client.Options{
		Scheme: f.Scheme,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create client for identity %q: %w", identity.String, err)
	}

	return scopedClient, nil
}

// NewDefaultRevisionEngineFactory creates a new defaultRevisionEngineFactory.
func NewDefaultRevisionEngineFactory(
	scheme *runtime.Scheme,
	trackingCache managedcache.TrackingCache,
	discoveryClient discovery.CachedDiscoveryInterface,
	restMapper meta.RESTMapper,
	fieldOwnerPrefix string,
	baseConfig *rest.Config,
	tokenGetter *authentication.TokenGetter,
) (RevisionEngineFactory, error) {
	if baseConfig == nil {
		return nil, fmt.Errorf("baseConfig is required but not provided")
	}
	if tokenGetter == nil {
		return nil, fmt.Errorf("tokenGetter is required but not provided")
	}
	return &defaultRevisionEngineFactory{
		Scheme:           scheme,
		TrackingCache:    trackingCache,
		DiscoveryClient:  discoveryClient,
		RESTMapper:       restMapper,
		FieldOwnerPrefix: fieldOwnerPrefix,
		BaseConfig:       baseConfig,
		TokenGetter:      tokenGetter,
	}, nil
}
