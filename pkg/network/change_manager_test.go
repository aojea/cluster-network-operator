package network

import (
	"testing"

	operv1 "github.com/openshift/api/operator/v1"
	"github.com/openshift/cluster-network-operator/pkg/bootstrap"
	"github.com/openshift/cluster-network-operator/pkg/names"
	"github.com/openshift/cluster-network-operator/pkg/util/k8s"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	uns "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestChangeManagerOVNDualStackConversion(t *testing.T) {

	tests := []struct {
		name              string
		conf              *operv1.NetworkSpec
		prev              *operv1.NetworkSpec
		bootNode          *appsv1.DaemonSet
		bootMaster        *appsv1.DaemonSet
		renderedNode      *appsv1.DaemonSet
		renderedMaster    *appsv1.DaemonSet
		checkAnnotationDS string
		want              bool
		wantErr           bool
	}{
		{
			name:    "all empty",
			want:    false,
			wantErr: false,
		},
		{
			name:       "no config changes",
			bootNode:   &appsv1.DaemonSet{},
			bootMaster: &appsv1.DaemonSet{},
			want:       false,
			wantErr:    false,
		},
		{
			name: "config changes but no dualstack",
			conf: &operv1.NetworkSpec{
				ClusterNetwork: []operv1.ClusterNetworkEntry{
					{
						CIDR:       "10.0.0.0/22",
						HostPrefix: 24,
					},
					{
						CIDR:       "10.2.0.0/22",
						HostPrefix: 23,
					},
				},
				ServiceNetwork: []string{"192.168.0.0/20"},
				DefaultNetwork: operv1.DefaultNetworkDefinition{
					Type: operv1.NetworkTypeOVNKubernetes,
					OVNKubernetesConfig: &operv1.OVNKubernetesConfig{
						GenevePort: ptrToUint32(8061),
					},
				},
			},
			prev: &operv1.NetworkSpec{
				ClusterNetwork: []operv1.ClusterNetworkEntry{
					{
						CIDR:       "10.0.0.0/22",
						HostPrefix: 24,
					},
					{
						CIDR:       "10.2.0.0/22",
						HostPrefix: 23,
					},
				},
				ServiceNetwork: []string{"192.168.0.0/20"},
				DefaultNetwork: operv1.DefaultNetworkDefinition{
					Type: operv1.NetworkTypeOVNKubernetes,
					OVNKubernetesConfig: &operv1.OVNKubernetesConfig{
						GenevePort: ptrToUint32(1061),
					},
				},
			},
			bootNode:   &appsv1.DaemonSet{},
			bootMaster: &appsv1.DaemonSet{},
			renderedNode: &appsv1.DaemonSet{
				TypeMeta: metav1.TypeMeta{APIVersion: "apps/v1", Kind: "DaemonSet"},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ovnkube-node",
					Namespace: "openshift-ovn-kubernetes",
				},
			},
			renderedMaster: &appsv1.DaemonSet{
				TypeMeta: metav1.TypeMeta{APIVersion: "apps/v1", Kind: "DaemonSet"},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ovnkube-master",
					Namespace: "openshift-ovn-kubernetes",
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "config changes single to dualstack",
			conf: &operv1.NetworkSpec{
				ClusterNetwork: []operv1.ClusterNetworkEntry{
					{
						CIDR:       "10.0.0.0/22",
						HostPrefix: 24,
					},
					{
						CIDR:       "fd00:3:2:1::/64",
						HostPrefix: 56,
					},
				},
				ServiceNetwork: []string{"192.168.0.0/20", "fd00:1:2:3::/112"},
				DefaultNetwork: operv1.DefaultNetworkDefinition{
					Type: operv1.NetworkTypeOVNKubernetes,
				},
			},
			prev: &operv1.NetworkSpec{
				ClusterNetwork: []operv1.ClusterNetworkEntry{
					{
						CIDR:       "10.0.0.0/22",
						HostPrefix: 24,
					},
				},
				ServiceNetwork: []string{"192.168.0.0/20"},
				DefaultNetwork: operv1.DefaultNetworkDefinition{
					Type: operv1.NetworkTypeOVNKubernetes,
				},
			},
			bootNode:   &appsv1.DaemonSet{},
			bootMaster: &appsv1.DaemonSet{},
			renderedNode: &appsv1.DaemonSet{
				TypeMeta: metav1.TypeMeta{APIVersion: "apps/v1", Kind: "DaemonSet"},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ovnkube-node",
					Namespace: "openshift-ovn-kubernetes",
				},
			},
			renderedMaster: &appsv1.DaemonSet{
				TypeMeta: metav1.TypeMeta{APIVersion: "apps/v1", Kind: "DaemonSet"},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ovnkube-master",
					Namespace: "openshift-ovn-kubernetes",
				},
			},
			checkAnnotationDS: "ovnkube-master",
			want:              true,
			wantErr:           false,
		},
		{
			name: "config changes single to dualstack and master rollout in progress",
			conf: &operv1.NetworkSpec{
				ClusterNetwork: []operv1.ClusterNetworkEntry{
					{
						CIDR:       "10.0.0.0/22",
						HostPrefix: 24,
					},
					{
						CIDR:       "fd00:3:2:1::/64",
						HostPrefix: 56,
					},
				},
				ServiceNetwork: []string{"192.168.0.0/20", "fd00:1:2:3::/112"},
				DefaultNetwork: operv1.DefaultNetworkDefinition{
					Type: operv1.NetworkTypeOVNKubernetes,
				},
			},
			prev: &operv1.NetworkSpec{
				ClusterNetwork: []operv1.ClusterNetworkEntry{
					{
						CIDR:       "10.0.0.0/22",
						HostPrefix: 24,
					},
				},
				ServiceNetwork: []string{"192.168.0.0/20"},
				DefaultNetwork: operv1.DefaultNetworkDefinition{
					Type: operv1.NetworkTypeOVNKubernetes,
				},
			},
			bootNode: &appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ovnkube-node",
					Namespace: "openshift-cluster-network-operator",
				}},
			bootMaster: &appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ovnkube-master",
					Namespace: "openshift-cluster-network-operator",
					Annotations: map[string]string{
						names.NetworkDualStackMigrationAnnotation: "true",
					},
					Generation: 1,
				},
				Spec: appsv1.DaemonSetSpec{
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{
								names.NetworkDualStackMigrationAnnotation: "true",
							},
						},
					},
				},
				Status: appsv1.DaemonSetStatus{
					CurrentNumberScheduled: 3,
					DesiredNumberScheduled: 3,
					NumberAvailable:        2,
					NumberUnavailable:      1,
					NumberMisscheduled:     0,
					NumberReady:            2,
					ObservedGeneration:     2,
					UpdatedNumberScheduled: 3,
				},
			},
			renderedNode: &appsv1.DaemonSet{
				TypeMeta: metav1.TypeMeta{APIVersion: "apps/v1", Kind: "DaemonSet"},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ovnkube-node",
					Namespace: "openshift-ovn-kubernetes",
				},
			},
			renderedMaster: &appsv1.DaemonSet{
				TypeMeta: metav1.TypeMeta{APIVersion: "apps/v1", Kind: "DaemonSet"},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ovnkube-master",
					Namespace: "openshift-ovn-kubernetes",
					Annotations: map[string]string{
						names.NetworkDualStackMigrationAnnotation: "true",
					},
				},
				Spec: appsv1.DaemonSetSpec{
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{
								names.NetworkDualStackMigrationAnnotation: "true",
							},
						},
					},
				},
			},
			checkAnnotationDS: "ovnkube-master",
			want:              true,
			wantErr:           false,
		},
		{
			name: "config changes single to dualstack and master finished rollout",
			conf: &operv1.NetworkSpec{
				ClusterNetwork: []operv1.ClusterNetworkEntry{
					{
						CIDR:       "10.0.0.0/22",
						HostPrefix: 24,
					},
					{
						CIDR:       "fd00:3:2:1::/64",
						HostPrefix: 56,
					},
				},
				ServiceNetwork: []string{"192.168.0.0/20", "fd00:1:2:3::/112"},
				DefaultNetwork: operv1.DefaultNetworkDefinition{
					Type: operv1.NetworkTypeOVNKubernetes,
				},
			},
			prev: &operv1.NetworkSpec{
				ClusterNetwork: []operv1.ClusterNetworkEntry{
					{
						CIDR:       "10.0.0.0/22",
						HostPrefix: 24,
					},
				},
				ServiceNetwork: []string{"192.168.0.0/20"},
				DefaultNetwork: operv1.DefaultNetworkDefinition{
					Type: operv1.NetworkTypeOVNKubernetes,
				},
			},
			bootNode: &appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ovnkube-node",
					Namespace: "openshift-cluster-network-operator",
				}},
			bootMaster: &appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ovnkube-master",
					Namespace: "openshift-cluster-network-operator",
					Annotations: map[string]string{
						names.NetworkDualStackMigrationAnnotation: "true",
					},
					Generation: 2,
				},
				Spec: appsv1.DaemonSetSpec{
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{
								names.NetworkDualStackMigrationAnnotation: "true",
							},
						},
					},
				},
				Status: appsv1.DaemonSetStatus{
					CurrentNumberScheduled: 3,
					DesiredNumberScheduled: 3,
					NumberAvailable:        3,
					NumberMisscheduled:     0,
					NumberReady:            3,
					ObservedGeneration:     2,
					UpdatedNumberScheduled: 3,
				},
			},
			renderedNode: &appsv1.DaemonSet{
				TypeMeta: metav1.TypeMeta{APIVersion: "apps/v1", Kind: "DaemonSet"},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ovnkube-node",
					Namespace: "openshift-ovn-kubernetes",
				},
			},
			renderedMaster: &appsv1.DaemonSet{
				TypeMeta: metav1.TypeMeta{APIVersion: "apps/v1", Kind: "DaemonSet"},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ovnkube-master",
					Namespace: "openshift-ovn-kubernetes",
					Annotations: map[string]string{
						names.NetworkDualStackMigrationAnnotation: "true",
					},
				},
				Spec: appsv1.DaemonSetSpec{
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{
								names.NetworkDualStackMigrationAnnotation: "true",
							},
						},
					},
				},
			},
			checkAnnotationDS: "ovnkube-node",
			want:              false,
			wantErr:           false,
		},
		{
			name: "config changes single to dualstack finished",
			conf: &operv1.NetworkSpec{
				ClusterNetwork: []operv1.ClusterNetworkEntry{
					{
						CIDR:       "10.0.0.0/22",
						HostPrefix: 24,
					},
					{
						CIDR:       "fd00:3:2:1::/64",
						HostPrefix: 56,
					},
				},
				ServiceNetwork: []string{"192.168.0.0/20", "fd00:1:2:3::/112"},
				DefaultNetwork: operv1.DefaultNetworkDefinition{
					Type: operv1.NetworkTypeOVNKubernetes,
				},
			},
			prev: &operv1.NetworkSpec{
				ClusterNetwork: []operv1.ClusterNetworkEntry{
					{
						CIDR:       "10.0.0.0/22",
						HostPrefix: 24,
					},
				},
				ServiceNetwork: []string{"192.168.0.0/20"},
				DefaultNetwork: operv1.DefaultNetworkDefinition{
					Type: operv1.NetworkTypeOVNKubernetes,
				},
			},
			bootNode: &appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ovnkube-node",
					Namespace: "openshift-cluster-network-operator",
				},
				Spec: appsv1.DaemonSetSpec{
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{
								names.NetworkDualStackMigrationAnnotation: "true",
							},
						},
					},
				},
			},
			bootMaster: &appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ovnkube-master",
					Namespace: "openshift-cluster-network-operator",
					Annotations: map[string]string{
						names.NetworkDualStackMigrationAnnotation: "true",
					},
					Generation: 2,
				},
				Spec: appsv1.DaemonSetSpec{
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{
								names.NetworkDualStackMigrationAnnotation: "true",
							},
						},
					},
				},
				Status: appsv1.DaemonSetStatus{
					CurrentNumberScheduled: 3,
					DesiredNumberScheduled: 3,
					NumberAvailable:        3,
					NumberMisscheduled:     0,
					NumberReady:            3,
					ObservedGeneration:     2,
					UpdatedNumberScheduled: 3,
				},
			},
			renderedNode: &appsv1.DaemonSet{
				TypeMeta: metav1.TypeMeta{APIVersion: "apps/v1", Kind: "DaemonSet"},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ovnkube-node",
					Namespace: "openshift-ovn-kubernetes",
				},
			},
			renderedMaster: &appsv1.DaemonSet{
				TypeMeta: metav1.TypeMeta{APIVersion: "apps/v1", Kind: "DaemonSet"},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ovnkube-master",
					Namespace: "openshift-ovn-kubernetes",
					Annotations: map[string]string{
						names.NetworkDualStackMigrationAnnotation: "true",
					},
				},
				Spec: appsv1.DaemonSetSpec{
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{
								names.NetworkDualStackMigrationAnnotation: "true",
							},
						},
					},
				},
			},
			checkAnnotationDS: "ovnkube-node",
			want:              false,
			wantErr:           false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bootstrapResult := &bootstrap.BootstrapResult{
				OVN: bootstrap.OVNBootstrapResult{
					ExistingMasterDaemonset: tt.bootMaster,
					ExistingNodeDaemonset:   tt.bootNode,
				},
			}
			objs := []*uns.Unstructured{}
			if tt.renderedNode != nil {
				obj, err := k8s.ToUnstructured(tt.renderedNode)
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				objs = append(objs, obj)
			}
			if tt.renderedMaster != nil {
				obj, err := k8s.ToUnstructured(tt.renderedMaster)
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				objs = append(objs, obj)
			}
			got, err := ChangeManager(tt.conf, tt.prev, bootstrapResult, objs)
			if (err != nil) != tt.wantErr {
				t.Errorf("ChangeManager() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ChangeManager InProgress received %v, want %v", got, tt.want)
			}

			if tt.checkAnnotationDS != "" {
				if !checkDaemonsetAnnotation(objs, tt.checkAnnotationDS, names.NetworkDualStackMigrationAnnotation, "true") {
					t.Errorf("ChangeManager() expected %s annotation on daemonset %s", names.NetworkDualStackMigrationAnnotation, tt.checkAnnotationDS)
				}
			}

		})
	}
}

// checkDaemonsetAnnotation check that all the daemonset have the annotation with the
// same key and value
func checkDaemonsetAnnotation(objs []*uns.Unstructured, name, key, value string) bool {
	if key == "" || value == "" || name == "" {
		return false
	}
	for _, obj := range objs {
		if obj.GetAPIVersion() == "apps/v1" && obj.GetKind() == "DaemonSet" && obj.GetName() == name {

			// check daemonset annotation
			anno := obj.GetAnnotations()
			if anno == nil {
				return false
			}
			v, ok := anno[key]
			if !ok || v != value {
				return false
			}
			// check template annotation
			anno, _, _ = uns.NestedStringMap(obj.Object, "spec", "template", "metadata", "annotations")
			if anno == nil {
				return false
			}
			v, ok = anno[key]
			if !ok || v != value {
				return false
			}
			return true
		}
	}
	return false
}
