package network

import (
	operv1 "github.com/openshift/api/operator/v1"
	"github.com/openshift/cluster-network-operator/pkg/bootstrap"
	uns "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog/v2"
)

// ChangeManager handles the configuration changes and modifies the rendered objects accordingly.
// The function has access to the rendered and bootstrap objects, and to the previous
// and new configurations.
// It can set the configurationChange in the reconciler object, in order to perform
// complex configuration changes that require multiple iterations (i.e. daemonset rollouts)
func ChangeManager(conf, prev *operv1.NetworkSpec, bootstrapResult *bootstrap.BootstrapResult, obj []*uns.Unstructured) (bool, error) {
	if conf == nil || prev == nil {
		return false, nil
	}
	if conf.DefaultNetwork.Type == operv1.NetworkTypeOVNKubernetes &&
		len(prev.ServiceNetwork) != len(conf.ServiceNetwork) {
		klog.V(2).Infof("Configuration change detected, dual-stack conversion in progress")
		return convertDualStackOVN(bootstrapResult, obj)
	}
	return false, nil
}
