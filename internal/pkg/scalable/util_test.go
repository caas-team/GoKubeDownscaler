package scalable

import (
	"regexp"
	"testing"

	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFilterExcluded(t *testing.T) {
	// define some example objects to use
	type ns struct {
		deployment1       deployment
		deployment2       deployment
		labeledDeployment deployment
	}
	ns1 := ns{
		deployment1: deployment{Deployment: &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "Deployment1",
				Namespace: "Namespace1",
			},
		}},
		deployment2: deployment{Deployment: &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "Deployment2",
				Namespace: "Namespace1",
			},
		}},
		labeledDeployment: deployment{Deployment: &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "LabeledDeployment",
				Namespace: "Namespace1",
				Labels: map[string]string{
					"label": "value",
				},
			},
		}},
	}
	ns2 := ns{
		deployment1: deployment{Deployment: &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "Deployment1",
				Namespace: "Namespace2",
			},
		}},
		deployment2: deployment{Deployment: &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "Deployment2",
				Namespace: "Namespace2",
			},
		}},
		labeledDeployment: deployment{Deployment: &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "LabeledDeployment",
				Namespace: "Namespace2",
				Labels: map[string]string{
					"label2": "value",
				},
			},
		}},
	}
	tests := []struct {
		name               string
		workloads          []Workload
		includeLabels      values.RegexList
		excludedNamespaces values.RegexList
		excludedWorkloads  values.RegexList
		want               []Workload
	}{
		{
			name:               "none set",
			workloads:          []Workload{&ns1.deployment1, &ns1.deployment2, &ns2.deployment1},
			includeLabels:      nil,
			excludedNamespaces: nil,
			excludedWorkloads:  nil,
			want:               []Workload{&ns1.deployment1, &ns1.deployment2, &ns2.deployment1},
		},
		{
			name:               "includeLabels",
			workloads:          []Workload{&ns1.deployment1, &ns1.deployment2, &ns1.labeledDeployment},
			includeLabels:      values.RegexList{regexp.MustCompile(".*")}, // match any label
			excludedNamespaces: nil,
			excludedWorkloads:  nil,
			want:               []Workload{&ns1.labeledDeployment},
		},
		{
			name:               "excludeNamespaces",
			workloads:          []Workload{&ns1.deployment1, &ns1.deployment2, &ns2.deployment1},
			includeLabels:      nil,
			excludedNamespaces: values.RegexList{regexp.MustCompile("Namespace1")}, // exclude Namespace1
			excludedWorkloads:  nil,
			want:               []Workload{&ns2.deployment1},
		},
		{
			name:               "excludeWorkloads",
			workloads:          []Workload{&ns1.deployment1, &ns1.deployment2, &ns2.deployment1},
			includeLabels:      nil,
			excludedNamespaces: nil,
			excludedWorkloads:  values.RegexList{regexp.MustCompile("Deployment1")}, // exclude Deployment1
			want:               []Workload{&ns1.deployment2},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := FilterExcluded(test.workloads, test.includeLabels, test.excludedNamespaces, test.excludedWorkloads)
			assert.Equal(t, test.want, got)
		})
	}
}
