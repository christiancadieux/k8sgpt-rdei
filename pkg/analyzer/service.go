/*
Copyright 2023 The K8sGPT Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package analyzer

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/k8sgpt-ai/k8sgpt/pkg/common"
	"github.com/k8sgpt-ai/k8sgpt/pkg/kubernetes"
	"github.com/k8sgpt-ai/k8sgpt/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type ServiceAnalyzer struct{}

func (ServiceAnalyzer) Analyze(a common.Analyzer) ([]common.Result, error) {

	kind := "Service"
	apiDoc := kubernetes.K8sApiReference{
		Kind: kind,
		ApiVersion: schema.GroupVersion{
			Group:   "",
			Version: "v1",
		},
		OpenapiSchema: a.OpenapiSchema,
	}

	AnalyzerErrorsMetric.DeletePartialMatch(map[string]string{
		"analyzer_name": kind,
	})

	// search all namespaces for pods that are not running
	list, err := a.Client.GetClient().CoreV1().Endpoints(a.Namespace).List(a.Context, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var preAnalysis = map[string]common.PreAnalysis{}

	for _, ep := range list.Items {
		if SkipNamespace(ep.Namespace) {
			continue
		}
		var failures []common.Failure

		// Check for empty service
		if len(ep.Subsets) == 0 {
			svc, err := a.Client.GetClient().CoreV1().Services(ep.Namespace).Get(a.Context, ep.Name, metav1.GetOptions{})
			if err != nil {
				color.Yellow("Service %s does not exist", ep.Name)
				continue
			}
			labels := ""
			for k, v := range svc.Spec.Selector {
				if labels != "" {
					labels += ", "
				}
				labels += fmt.Sprintf("%s=%s", k, v)
			}
			doc := apiDoc.GetApiDocV2("spec.selector")
			failures = append(failures, common.Failure{
				Text:          fmt.Sprintf("Service %s has no endpoints, expected labels [%s]", ep.Name, labels),
				KubernetesDoc: doc,
				Sensitive:     []common.Sensitive{},
			})

		} else {
			count := 0
			pods := []string{}

			// Check through container status to check for crashes
			for _, epSubset := range ep.Subsets {
				apiDoc.Kind = "Endpoints"

				if len(epSubset.NotReadyAddresses) > 0 {
					for _, addresses := range epSubset.NotReadyAddresses {
						count++
						pods = append(pods, addresses.TargetRef.Kind+"/"+addresses.TargetRef.Name)
					}

					doc := apiDoc.GetApiDocV2("subsets.notReadyAddresses")

					failures = append(failures, common.Failure{
						Text:          fmt.Sprintf("Service has not ready endpoints, pods: %s, expected %d", pods, count),
						KubernetesDoc: doc,
						Sensitive:     []common.Sensitive{},
					})
				}
			}
		}

		if len(failures) > 0 {
			preAnalysis[fmt.Sprintf("%s/%s", ep.Namespace, ep.Name)] = common.PreAnalysis{
				Namespace:      ep.Namespace,
				ResourceName:   ep.Name,
				Endpoint:       ep,
				FailureDetails: failures,
			}
			AnalyzerErrorsMetric.WithLabelValues(kind, ep.Name, ep.Namespace).Set(float64(len(failures)))
		}
	}

	for key, value := range preAnalysis {
		var currentAnalysis = common.Result{
			Namespace:    value.Namespace,
			ResourceName: value.ResourceName,
			Kind:         kind,
			Name:         key,
			Error:        value.FailureDetails,
		}

		parent, _ := util.GetParent(a.Client, value.Endpoint.ObjectMeta)
		currentAnalysis.ParentObject = parent
		a.Results = append(a.Results, currentAnalysis)
	}
	return a.Results, nil
}
