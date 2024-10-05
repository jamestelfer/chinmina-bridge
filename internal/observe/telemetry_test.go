package observe

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/resource"
)

func Test_ResourceMerge(t *testing.T) {
	// Ensure that schema incompatibility on OTEL upgrades is detected before
	// merge
	_, err := resourceWithServiceName(
		resource.Default(),
		"serviceName")

	require.NoError(t, err)
}
