/*
Copyright 2025 SUSE.

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

package util

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	// uninitializedTaintKey is the taint added by kubelet when cloudProviderName=external.
	uninitializedTaintKey = "node.cloudprovider.kubernetes.io/uninitialized"
)

// InitializeWorkloadNode sets the providerID and removes the cloud-provider
// uninitialized taint on a workload cluster node. This bypasses the
// chicken-and-egg problem where the cloud-provider-harvester pod cannot
// schedule because CNI is blocked by the uninitialized taint.
//
// This is a best-effort operation: all errors are logged as warnings
// and do not propagate, so it never blocks the reconcile loop.
func InitializeWorkloadNode(ctx context.Context, logger logr.Logger, workloadConfig *rest.Config, nodeName, providerID string) {
	if providerID == "" {
		return
	}

	clientset, err := kubernetes.NewForConfig(workloadConfig)
	if err != nil {
		logger.Info("Warning: failed to create workload client for node init", "error", err)
		return
	}

	node, err := clientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Node not yet registered in workload cluster, will retry next reconcile
			return
		}
		logger.Info("Warning: failed to get workload node for init", "error", err, "node", nodeName)
		return
	}

	needsProviderID := node.Spec.ProviderID == ""
	needsTaintRemoval := hasUninitializedTaint(node)

	if !needsProviderID && !needsTaintRemoval {
		return
	}

	if needsProviderID {
		patch := fmt.Sprintf(`{"spec":{"providerID":%q}}`, providerID)
		_, err = clientset.CoreV1().Nodes().Patch(ctx, nodeName, types.MergePatchType, []byte(patch), metav1.PatchOptions{})
		if err != nil {
			logger.Info("Warning: failed to set providerID on workload node",
				"error", err, "node", nodeName, "providerID", providerID)
			return
		}
		logger.Info("Set providerID on workload node", "node", nodeName, "providerID", providerID)
	}

	if needsTaintRemoval {
		// Re-fetch node to avoid conflict after providerID patch
		node, err = clientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
		if err != nil {
			logger.Info("Warning: failed to re-fetch node after providerID patch", "error", err)
			return
		}

		newTaints := removeTaint(node.Spec.Taints, uninitializedTaintKey)
		if len(newTaints) != len(node.Spec.Taints) {
			taintsJSON, _ := json.Marshal(newTaints)
			patch := fmt.Sprintf(`{"spec":{"taints":%s}}`, taintsJSON)
			_, err = clientset.CoreV1().Nodes().Patch(ctx, nodeName, types.MergePatchType, []byte(patch), metav1.PatchOptions{})
			if err != nil {
				logger.Info("Warning: failed to remove uninitialized taint",
					"error", err, "node", nodeName)
				return
			}
			logger.Info("Removed cloud-provider uninitialized taint", "node", nodeName)
		}
	}
}

// hasUninitializedTaint returns true if the node has the cloud-provider uninitialized taint.
func hasUninitializedTaint(node *v1.Node) bool {
	for _, taint := range node.Spec.Taints {
		if taint.Key == uninitializedTaintKey {
			return true
		}
	}
	return false
}

// removeTaint returns a copy of the taint slice with the specified key removed.
func removeTaint(taints []v1.Taint, key string) []v1.Taint {
	result := make([]v1.Taint, 0, len(taints))
	for _, t := range taints {
		if t.Key != key {
			result = append(result, t)
		}
	}
	return result
}
