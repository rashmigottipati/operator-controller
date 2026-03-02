package authentication

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	ocv1 "github.com/operator-framework/operator-controller/api/v1"
)

func TestBuildIdentity(t *testing.T) {
	tests := []struct {
		name     string
		ext      *ocv1.ClusterExtension
		expected string
	}{
		{
			name: "ServiceAccount identity",
			ext: &ocv1.ClusterExtension{
				Spec: ocv1.ClusterExtensionSpec{
					Namespace: "test-ns",
					ServiceAccount: ocv1.ServiceAccountReference{
						Name: "test-sa",
					},
				},
			},
			expected: "system:serviceaccounts:test-ns:test-sa",
		},
		{
			name: "Synthetic identity - no ServiceAccount",
			ext: &ocv1.ClusterExtension{
				Spec: ocv1.ClusterExtensionSpec{
					Namespace: "test-ns",
				},
			},
			expected: "olm:clusterextension:",
		},
		{
			name: "Synthetic identity - empty ServiceAccount name",
			ext: &ocv1.ClusterExtension{
				Spec: ocv1.ClusterExtensionSpec{
					Namespace: "test-ns",
					ServiceAccount: ocv1.ServiceAccountReference{
						Name: "",
					},
				},
			},
			expected: "olm:clusterextension:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.ext.Name = "" // ClusterExtension name affects synthetic identity
			result := BuildIdentity(tt.ext)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseIdentity(t *testing.T) {
	tests := []struct {
		name          string
		identityStr   string
		expectedType  IdentityType
		expectedErr   bool
		validateField func(*testing.T, *Identity)
	}{
		{
			name:         "Valid ServiceAccount identity",
			identityStr:  "system:serviceaccounts:my-ns:my-sa",
			expectedType: IdentityTypeServiceAccount,
			validateField: func(t *testing.T, id *Identity) {
				assert.Equal(t, "my-ns", id.ServiceAccountNamespace)
				assert.Equal(t, "my-sa", id.ServiceAccountName)
				assert.Equal(t, "system:serviceaccounts:my-ns:my-sa", id.String)
			},
		},
		{
			name:         "Valid synthetic identity",
			identityStr:  "olm:clusterextension:my-ext",
			expectedType: IdentityTypeSynthetic,
			validateField: func(t *testing.T, id *Identity) {
				assert.Equal(t, "my-ext", id.ClusterExtensionName)
				assert.Equal(t, "olm:clusterextension:my-ext", id.String)
			},
		},
		{
			name:         "Invalid ServiceAccount identity - missing namespace",
			identityStr:  "system:serviceaccounts:my-sa",
			expectedType: IdentityTypeUnknown,
			expectedErr:  true,
		},
		{
			name:         "Invalid - unknown prefix",
			identityStr:  "unknown:prefix:value",
			expectedType: IdentityTypeUnknown,
			expectedErr:  true,
		},
		{
			name:        "Invalid - empty string",
			identityStr: "",
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseIdentity(tt.identityStr)

			if tt.expectedErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedType, result.Type)
			if tt.validateField != nil {
				tt.validateField(t, result)
			}
		})
	}
}
