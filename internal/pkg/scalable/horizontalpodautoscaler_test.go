package scalable

import (
	"errors"
	"testing"

	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
)

func TestHorizontalPodAutoscaler_ScaleUp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		minReplicas          int32
		maxReplicas          int32
		originalReplicas     values.Replicas
		wantMinReplicas      values.Replicas
		wantOriginalReplicas values.Replicas
	}{
		{
			name:                 "clamps minReplicas to maxReplicas when original exceeds maxReplicas",
			minReplicas:          1,
			maxReplicas:          3,
			originalReplicas:     values.AbsoluteReplicas(5),
			wantMinReplicas:      values.AbsoluteReplicas(3),
			wantOriginalReplicas: nil,
		},
		{
			name:                 "minReplicas already at original value but annotation still present, removes annotation",
			minReplicas:          5,
			maxReplicas:          10,
			originalReplicas:     values.AbsoluteReplicas(5),
			wantMinReplicas:      values.AbsoluteReplicas(5),
			wantOriginalReplicas: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			hpaObj := &autoscalingv2.HorizontalPodAutoscaler{}
			hpaObj.Spec.MinReplicas = int32Ptr(test.minReplicas)
			hpaObj.Spec.MaxReplicas = test.maxReplicas

			workload := &replicaScaledWorkload{&horizontalPodAutoscaler{hpaObj}}

			if test.originalReplicas != nil {
				setOriginalReplicas(test.originalReplicas, workload)
			}

			err := workload.ScaleUp()
			require.NoError(t, err)

			gotReplicas, err := workload.getReplicas()
			require.NoError(t, err)
			assert.Equal(t, test.wantMinReplicas, gotReplicas)

			gotOriginal, err := getOriginalReplicas(workload)
			var unsetErr *OriginalReplicasUnsetError

			if !errors.As(err, &unsetErr) {
				require.NoError(t, err)
			}

			assert.Equal(t, test.wantOriginalReplicas, gotOriginal)
		})
	}
}
