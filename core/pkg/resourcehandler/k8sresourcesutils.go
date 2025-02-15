package resourcehandler

import (
	"strings"

	"github.com/armosec/kubescape/v2/core/cautils"
	"github.com/armosec/kubescape/v2/core/pkg/hostsensorutils"
	"github.com/armosec/opa-utils/reporthandling"
	"k8s.io/utils/strings/slices"

	"github.com/armosec/k8s-interface/k8sinterface"
)

var (
	ClusterDescribe = "ClusterDescribe"

	MapResourceToApiGroupCloud = map[string][]string{
		ClusterDescribe: {"container.googleapis.com/v1", "eks.amazonaws.com/v1", "management.azure.com/v1"}}
)

func setK8sResourceMap(frameworks []reporthandling.Framework) *cautils.K8SResources {
	k8sResources := make(cautils.K8SResources)
	complexMap := setComplexK8sResourceMap(frameworks)
	for group := range complexMap {
		for version := range complexMap[group] {
			for resource := range complexMap[group][version] {
				groupResources := k8sinterface.ResourceGroupToString(group, version, resource)
				for _, groupResource := range groupResources {
					k8sResources[groupResource] = nil
				}
			}
		}
	}
	return &k8sResources
}

func setArmoResourceMap(frameworks []reporthandling.Framework, resourceToControl map[string][]string) *cautils.ArmoResources {
	armoResources := make(cautils.ArmoResources)
	complexMap := setComplexArmoResourceMap(frameworks, resourceToControl)
	for group := range complexMap {
		for version := range complexMap[group] {
			for resource := range complexMap[group][version] {
				groupResources := k8sinterface.ResourceGroupToString(group, version, resource)
				for _, groupResource := range groupResources {
					armoResources[groupResource] = nil
				}
			}
		}
	}
	return &armoResources
}

func setComplexK8sResourceMap(frameworks []reporthandling.Framework) map[string]map[string]map[string]interface{} {
	k8sResources := make(map[string]map[string]map[string]interface{})
	for _, framework := range frameworks {
		for _, control := range framework.Controls {
			for _, rule := range control.Rules {
				for _, match := range rule.Match {
					insertResources(k8sResources, match)
				}
			}
		}
	}
	return k8sResources
}

// [group][versionn][resource]
func setComplexArmoResourceMap(frameworks []reporthandling.Framework, resourceToControls map[string][]string) map[string]map[string]map[string]interface{} {
	k8sResources := make(map[string]map[string]map[string]interface{})
	for _, framework := range frameworks {
		for _, control := range framework.Controls {
			for _, rule := range control.Rules {
				for _, match := range rule.DynamicMatch {
					insertArmoResourcesAndControls(k8sResources, match, resourceToControls, control)
				}
			}
		}
	}
	return k8sResources
}

func mapArmoResourceToApiGroup(resource string) []string {
	if val, ok := hostsensorutils.MapResourceToApiGroup[resource]; ok {
		return []string{val}
	}
	return MapResourceToApiGroupCloud[resource]
}

func insertControls(resource string, resourceToControl map[string][]string, control reporthandling.Control) {
	armoResources := mapArmoResourceToApiGroup(resource)
	for _, armoResource := range armoResources {
		group, version := k8sinterface.SplitApiVersion(armoResource)
		r := k8sinterface.JoinResourceTriplets(group, version, resource)
		if _, ok := resourceToControl[r]; !ok {
			resourceToControl[r] = append(resourceToControl[r], control.ControlID)
		} else {
			if !slices.Contains(resourceToControl[r], control.ControlID) {
				resourceToControl[r] = append(resourceToControl[r], control.ControlID)
			}
		}
	}
}

func insertResources(k8sResources map[string]map[string]map[string]interface{}, match reporthandling.RuleMatchObjects) {
	for _, apiGroup := range match.APIGroups {
		if v, ok := k8sResources[apiGroup]; !ok || v == nil {
			k8sResources[apiGroup] = make(map[string]map[string]interface{})
		}
		for _, apiVersions := range match.APIVersions {
			if v, ok := k8sResources[apiGroup][apiVersions]; !ok || v == nil {
				k8sResources[apiGroup][apiVersions] = make(map[string]interface{})
			}
			for _, resource := range match.Resources {
				if _, ok := k8sResources[apiGroup][apiVersions][resource]; !ok {
					k8sResources[apiGroup][apiVersions][resource] = nil
				}
			}
		}
	}
}

func insertArmoResourcesAndControls(k8sResources map[string]map[string]map[string]interface{}, match reporthandling.RuleMatchObjects, resourceToControl map[string][]string, control reporthandling.Control) {
	for _, apiGroup := range match.APIGroups {
		if v, ok := k8sResources[apiGroup]; !ok || v == nil {
			k8sResources[apiGroup] = make(map[string]map[string]interface{})
		}
		for _, apiVersions := range match.APIVersions {
			if v, ok := k8sResources[apiGroup][apiVersions]; !ok || v == nil {
				k8sResources[apiGroup][apiVersions] = make(map[string]interface{})
			}
			for _, resource := range match.Resources {
				if _, ok := k8sResources[apiGroup][apiVersions][resource]; !ok {
					k8sResources[apiGroup][apiVersions][resource] = nil
				}
				insertControls(resource, resourceToControl, control)
			}
		}
	}
}

func getGroupNVersion(apiVersion string) (string, string) {
	gv := strings.Split(apiVersion, "/")
	group, version := "", ""
	if len(gv) >= 1 {
		group = gv[0]
	}
	if len(gv) >= 2 {
		version = gv[1]
	}
	return group, version
}
