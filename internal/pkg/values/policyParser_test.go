package values

import (
	"testing"

	"github.com/caas-team/gokubedownscaler/api/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestPolicyValuesToAnnotations(t *testing.T) {
	t.Parallel()

	annotations := PolicyValuesToAnnotations(&v1alpha1.DownscalePolicyValues{
		Uptime:            "Mon-Fri 08:00-19:00 Europe/Paris",
		DownscaleReplicas: "1",
		GracePeriod:       "15m",
	})

	assert.Equal(t, map[string]string{
		annotationUptime:            "Mon-Fri 08:00-19:00 Europe/Paris",
		annotationDownscaleReplicas: "1",
		annotationGracePeriod:       "15m",
	}, annotations)
}
