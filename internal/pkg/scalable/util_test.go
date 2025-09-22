package scalable

import (
	"regexp"
	"testing"

	"github.com/caas-team/gokubedownscaler/internal/pkg/util"
	"github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFilterExcluded(t *testing.T) {
	t.Parallel()

	// define some example objects to use
	type ns struct {
		deployment1       Workload
		deployment2       Workload
		labeledDeployment Workload
		scaledObject      Workload
		job1              Workload
		job2              Workload
	}

	ns1 := ns{
		deployment1: &replicaScaledWorkload{&deployment{Deployment: &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "Deployment1",
				Namespace: "Namespace1",
			},
		}}},
		deployment2: &replicaScaledWorkload{&deployment{Deployment: &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "Deployment2",
				Namespace: "Namespace1",
			},
		}}},
		labeledDeployment: &replicaScaledWorkload{&deployment{Deployment: &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "LabeledDeployment",
				Namespace: "Namespace1",
				Labels: map[string]string{
					"label": "value",
				},
			},
		}}},
	}
	ns2 := ns{
		deployment1: &replicaScaledWorkload{&deployment{Deployment: &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "Deployment1",
				Namespace: "Namespace2",
			},
		}}},
	}
	ns3 := ns{
		deployment1: &replicaScaledWorkload{&deployment{Deployment: &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "Deployment1",
				Namespace: "Namespace3",
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "Deployment",
				APIVersion: "apps/v1",
			},
		}}},
		deployment2: &replicaScaledWorkload{&deployment{Deployment: &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "Deployment2",
				Namespace: "Namespace3",
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "Deployment",
				APIVersion: "apps/v1",
			},
		}}},
		scaledObject: &replicaScaledWorkload{&scaledObject{ScaledObject: &v1alpha1.ScaledObject{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ScaledObject1",
				Namespace: "Namespace3",
			},
			TypeMeta: metav1.TypeMeta{
				APIVersion: "keda.sh/v1alpha1",
				Kind:       "ScaledObject",
			},
			Spec: v1alpha1.ScaledObjectSpec{
				ScaleTargetRef: &v1alpha1.ScaleTarget{
					Name:       "Deployment2",
					APIVersion: "apps/v1",
					Kind:       "",
				},
			},
		}}},
		job1: &suspendScaledWorkload{&job{Job: &v1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "Job1",
				Namespace: "Namespace3",
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion:         "batch/v1",
						Kind:               "CronJob",
						Name:               "CronJob1",
						UID:                "4a6e4a30-c474-4bc5-9bf5-47f29430fb41",
						Controller:         func() *bool { b := true; return &b }(),
						BlockOwnerDeletion: func() *bool { b := true; return &b }(),
					},
				},
			},
			TypeMeta: metav1.TypeMeta{
				APIVersion: "batch/v1",
				Kind:       "Job",
			},
		}}},
		job2: &suspendScaledWorkload{&job{Job: &v1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "Job2",
				Namespace: "Namespace3",
			},
			TypeMeta: metav1.TypeMeta{
				APIVersion: "batch/v1",
				Kind:       "Job",
			},
		}}},
	}
	ns4 := ns{
		scaledObject: &replicaScaledWorkload{&scaledObject{
			ScaledObject: &v1alpha1.ScaledObject{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "scaledSlash",
					Namespace: "Namespace4",
				},
				TypeMeta: metav1.TypeMeta{
					APIVersion: "keda.sh/v1alpha1", // with slash here
					Kind:       "ScaledObject",
				},
				Spec: v1alpha1.ScaledObjectSpec{
					ScaleTargetRef: &v1alpha1.ScaleTarget{
						Name:       "replicationControllerSlash",
						APIVersion: "v1", // with slash to trigger that branch
						Kind:       "ReplicationController",
					},
				},
			},
		}},
	}
	ns5 := ns{
		scaledObject: &replicaScaledWorkload{&scaledObject{
			ScaledObject: &v1alpha1.ScaledObject{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "scaledWithNoKindApiVersion",
					Namespace: "Namespace4",
				},
				TypeMeta: metav1.TypeMeta{
					APIVersion: "keda.sh/v1alpha1", // with slash here
					Kind:       "ScaledObject",
				},
				Spec: v1alpha1.ScaledObjectSpec{
					ScaleTargetRef: &v1alpha1.ScaleTarget{
						Name: "defaultDeploy",
					},
				},
			},
		}},
	}

	tests := []struct {
		name               string
		workloads          []Workload
		includeLabels      util.RegexList
		excludedNamespaces util.RegexList
		excludedWorkloads  util.RegexList
		want               []Workload
	}{
		{
			name:               "none set",
			workloads:          []Workload{ns1.deployment1, ns1.deployment2, ns2.deployment1},
			includeLabels:      nil,
			excludedNamespaces: nil,
			excludedWorkloads:  nil,
			want:               []Workload{ns1.deployment1, ns1.deployment2, ns2.deployment1},
		},
		{
			name:               "apiVersion without slash",
			workloads:          []Workload{ns4.scaledObject},
			includeLabels:      nil,
			excludedNamespaces: nil,
			excludedWorkloads:  nil,
			want:               []Workload{ns4.scaledObject},
		},
		{
			name:               "apiVersion and kind not specified",
			workloads:          []Workload{ns5.scaledObject},
			includeLabels:      nil,
			excludedNamespaces: nil,
			excludedWorkloads:  nil,
			want:               []Workload{ns5.scaledObject},
		},
		{
			name:               "includeLabels",
			workloads:          []Workload{ns1.deployment1, ns1.deployment2, ns1.labeledDeployment},
			includeLabels:      util.RegexList{regexp.MustCompile(".*")}, // match any label
			excludedNamespaces: nil,
			excludedWorkloads:  nil,
			want:               []Workload{ns1.labeledDeployment},
		},
		{
			name:               "excludeNamespaces",
			workloads:          []Workload{ns1.deployment1, ns1.deployment2, ns2.deployment1},
			includeLabels:      nil,
			excludedNamespaces: util.RegexList{regexp.MustCompile("Namespace1")}, // exclude Namespace1
			excludedWorkloads:  nil,
			want:               []Workload{ns2.deployment1},
		},
		{
			name:               "excludeWorkloads",
			workloads:          []Workload{ns1.deployment1, ns1.deployment2, ns2.deployment1},
			includeLabels:      nil,
			excludedNamespaces: nil,
			excludedWorkloads:  util.RegexList{regexp.MustCompile("Deployment1")}, // exclude Deployment1
			want:               []Workload{ns1.deployment2},
		},
		{
			name:               "exclude scaled object scaled",
			workloads:          []Workload{ns3.deployment1, ns3.deployment2, ns3.scaledObject, ns1.deployment1, ns1.deployment2, ns2.deployment1},
			includeLabels:      nil,
			excludedNamespaces: nil,
			excludedWorkloads:  nil,
			want:               []Workload{ns3.deployment1, ns3.scaledObject, ns1.deployment1, ns1.deployment2, ns2.deployment1},
		},
		{
			name:               "exclude managed by ownerReference",
			workloads:          []Workload{ns3.deployment1, ns3.deployment2, ns3.scaledObject, ns1.deployment1, ns3.job1, ns3.job2},
			includeLabels:      nil,
			excludedNamespaces: nil,
			excludedWorkloads:  nil,
			want:               []Workload{ns3.deployment1, ns3.scaledObject, ns1.deployment1, ns3.job2},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := FilterExcluded(
				test.workloads,
				test.includeLabels,
				test.excludedNamespaces,
				test.excludedWorkloads,
			)

			assert.Equal(t, test.want, got)
		})
	}
}
