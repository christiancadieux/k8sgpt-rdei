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

	"github.com/k8sgpt-ai/k8sgpt/pkg/common"
	"github.com/k8sgpt-ai/k8sgpt/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PodAnalyzer struct {
}

func isSystemNamespace(ns string) bool {
	if ns == "default" || ns == "kube-node-lease" || ns == "kube-public" || ns == "kube-system" || ns == "platform-load-balancer" ||
		ns == "rdei-system" {
		return true
	}
	return false
}

func VolumeWithoutGreenSelector(pod corev1.Pod) bool {

	hasVolume := false
	if pod.Spec.Volumes != nil {
		for _, v := range pod.Spec.Volumes {
			if v.PersistentVolumeClaim != nil {
				hasVolume = true
				break
			}
		}
	}
	if !hasVolume {
		return false
	}
	// there is a volume - need to be green
	if pod.Spec.NodeSelector == nil {
		return true
	}

	if _, ok := pod.Spec.NodeSelector["rdei.io/sec-zone-green"]; !ok {
		return true
	}
	return false
}

func HasPBSVolume(a common.Analyzer, pod corev1.Pod, pvcList *corev1.PersistentVolumeClaimList) string {

	if pod.Spec.Volumes != nil {
		for _, v := range pod.Spec.Volumes {
			if v.PersistentVolumeClaim != nil {
				for _, pvc := range pvcList.Items {
					if pvc.Name == v.PersistentVolumeClaim.ClaimName {
						if pvc.Spec.StorageClassName != nil {
							pv, err := a.Client.GetClient().CoreV1().PersistentVolumes().Get(a.Context, pvc.Spec.VolumeName, metav1.GetOptions{})
							if err != nil {
								fmt.Printf("Error reading PersistentVolume %s - %v \n ", pvc.Spec.VolumeName, err)
								return ""
							}
							if pv.Spec.PortworxVolume != nil {
								return pvc.Spec.VolumeName
							}
						}
					}
				}
			}
		}
	}
	return ""
}

func (PodAnalyzer) Analyze(a common.Analyzer) ([]common.Result, error) {

	kind := "Pod"

	AnalyzerErrorsMetric.DeletePartialMatch(map[string]string{
		"analyzer_name": kind,
	})

	pvcList, err := a.Client.GetClient().CoreV1().PersistentVolumeClaims(a.Namespace).List(a.Context, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// search all namespaces for pods that are not running
	list, err := a.Client.GetClient().CoreV1().Pods(a.Namespace).List(a.Context, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var preAnalysis = map[string]common.PreAnalysis{}

	nodeSelectorMissing := 0
	for _, pod := range list.Items {
		if SkipNamespace(pod.Namespace) {
			continue
		}
		var failures []common.Failure
		// Check for pending pods

		if !isSystemNamespace(pod.Namespace) {
			if pod.Spec.SchedulerName != "stork" {
				vol := HasPBSVolume(a, pod, pvcList)
				if vol != "" {
					failures = append(failures, common.Failure{
						Text: fmt.Sprintf(`Pod %s is accessing PBS volume %s and need to run the stork scheduler. `, pod.Name, vol),
						Sensitive: []common.Sensitive{
							{
								Unmasked: pod.Name, Masked: util.MaskString(pod.Name),
							},
						},
					})
				}
			}
			if VolumeWithoutGreenSelector(pod) {
				failures = append(failures, common.Failure{
					Text: fmt.Sprintf(`Pod %s is accessing a volume and need to run in the green zone. `, pod.Name),
					Sensitive: []common.Sensitive{
						{
							Unmasked: pod.Name, Masked: util.MaskString(pod.Name),
						},
					},
				})
			}

			if pod.Spec.NodeSelector == nil && nodeSelectorMissing == 0 {
				nodeSelectorMissing++
				failures = append(failures, common.Failure{
					Text: `Pods need a spec.nodeSelector. Add rdei.io/sec-zone-green: "true" to the pod or deployment to access the green zone. Same for blue or origin.`,
					Sensitive: []common.Sensitive{
						{
							Unmasked: pod.Name, Masked: util.MaskString(pod.Name),
						},
					},
				})
			}
		}

		if pod.Status.Phase == "Pending" {

			// Check through container status to check for crashes
			for _, containerStatus := range pod.Status.Conditions {
				if containerStatus.Type == "PodScheduled" && containerStatus.Reason == "Unschedulable" {
					if containerStatus.Message != "" {
						failures = append(failures, common.Failure{
							Text:      containerStatus.Message,
							Sensitive: []common.Sensitive{},
						})
					}
				}
			}
		}

		// Check through container status to check for crashes or unready
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if containerStatus.State.Waiting != nil {
				if containerStatus.State.Waiting.Reason == "CrashLoopBackOff" || containerStatus.State.Waiting.Reason == "ImagePullBackOff" {
					if containerStatus.State.Waiting.Message != "" {
						failures = append(failures, common.Failure{
							Text:      containerStatus.State.Waiting.Message,
							Sensitive: []common.Sensitive{},
						})
					}
				}
				// This represents a container that is still being created or blocked due to conditions such as OOMKilled
				if containerStatus.State.Waiting.Reason == "ContainerCreating" && pod.Status.Phase == "Pending" {

					// parse the event log and append details
					evt, err := FetchLatestEvent(a.Context, a.Client, pod.Namespace, pod.Name)
					if err != nil || evt == nil {
						continue
					}
					if evt.Reason == "FailedCreatePodSandBox" && evt.Message != "" {
						failures = append(failures, common.Failure{
							Text:      evt.Message,
							Sensitive: []common.Sensitive{},
						})
					}
				}
			} else {
				// when pod is Running but its ReadinessProbe fails
				if !containerStatus.Ready && pod.Status.Phase == "Running" {
					// parse the event log and append details
					evt, err := FetchLatestEvent(a.Context, a.Client, pod.Namespace, pod.Name)
					if err != nil || evt == nil {
						continue
					}
					if evt.Reason == "Unhealthy" && evt.Message != "" {
						failures = append(failures, common.Failure{
							Text:      evt.Message,
							Sensitive: []common.Sensitive{},
						})

					}

				}
			}
		}
		if len(failures) > 0 {
			preAnalysis[fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)] = common.PreAnalysis{
				Namespace:      pod.Namespace,
				ResourceName:   pod.Name,
				Pod:            pod,
				FailureDetails: failures,
			}
			AnalyzerErrorsMetric.WithLabelValues(kind, pod.Name, pod.Namespace).Set(float64(len(failures)))
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

		parent, _ := util.GetParent(a.Client, value.Pod.ObjectMeta)
		currentAnalysis.ParentObject = parent
		a.Results = append(a.Results, currentAnalysis)
	}

	return a.Results, nil
}
