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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-logr/logr"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

const (
	etcdNamespace     = "kube-system"
	etcdCACert        = "/var/lib/rancher/rke2/server/tls/etcd/server-ca.crt"
	etcdClientCert    = "/var/lib/rancher/rke2/server/tls/etcd/server-client.crt"
	etcdClientKey     = "/var/lib/rancher/rke2/server/tls/etcd/server-client.key"
	etcdctlPath       = "/usr/local/bin/etcdctl"
	etcdEndpoint      = "https://127.0.0.1:2379"
	etcdContainerName = "etcd"
)

// EtcdMemberListResponse represents the JSON output of etcdctl member list.
type EtcdMemberListResponse struct {
	Members []EtcdMember `json:"members"`
}

// EtcdMember represents a single etcd cluster member.
type EtcdMember struct {
	ID         uint64   `json:"ID"`
	Name       string   `json:"name"`
	PeerURLs   []string `json:"peerURLs"`
	ClientURLs []string `json:"clientURLs"`
}

// RemoveEtcdMember removes the etcd member corresponding to deletedNodeName from the
// workload cluster. This is a best-effort operation: all errors are logged as warnings
// and do not propagate, so it never blocks VM deletion.
func RemoveEtcdMember(ctx context.Context, logger logr.Logger, workloadConfig *rest.Config, deletedNodeName string) {
	clientset, err := kubernetes.NewForConfig(workloadConfig)
	if err != nil {
		logger.Info("Warning: failed to create workload cluster client for etcd cleanup", "error", err)

		return
	}

	pod, err := findHealthyEtcdPod(ctx, clientset, deletedNodeName)
	if err != nil {
		logger.Info("Warning: failed to find healthy etcd pod for cleanup", "error", err)

		return
	}

	members, err := listEtcdMembers(ctx, clientset, workloadConfig, pod)
	if err != nil {
		logger.Info("Warning: failed to list etcd members", "error", err)

		return
	}

	// RKE2 etcd member names follow the pattern: {nodeName}-{hash}
	var targetMember *EtcdMember

	for i := range members {
		if strings.HasPrefix(members[i].Name, deletedNodeName+"-") || members[i].Name == deletedNodeName {
			targetMember = &members[i]

			break
		}
	}

	if targetMember == nil {
		logger.Info("No etcd member found matching deleted node, nothing to remove",
			"deletedNode", deletedNodeName)

		return
	}

	err = removeEtcdMemberByID(ctx, clientset, workloadConfig, pod, targetMember.ID)
	if err != nil {
		logger.Info("Warning: failed to remove etcd member",
			"error", err, "memberName", targetMember.Name, "memberID", targetMember.ID)

		return
	}

	logger.Info("Successfully removed etcd member",
		"memberName", targetMember.Name, "memberID", targetMember.ID, "deletedNode", deletedNodeName)
}

// findHealthyEtcdPod finds a Running+Ready etcd pod on a node other than deletedNodeName.
func findHealthyEtcdPod(ctx context.Context, clientset kubernetes.Interface, deletedNodeName string) (*v1.Pod, error) {
	// Try label selector first (standard for kubeadm/RKE2)
	pods, err := clientset.CoreV1().Pods(etcdNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: "component=etcd,tier=control-plane",
	})
	if err != nil || len(pods.Items) == 0 {
		// Fallback: scan all pods in kube-system by prefix
		allPods, err2 := clientset.CoreV1().Pods(etcdNamespace).List(ctx, metav1.ListOptions{})
		if err2 != nil {
			return nil, fmt.Errorf("failed to list pods in %s: %w", etcdNamespace, err2)
		}

		pods = &v1.PodList{}

		for i := range allPods.Items {
			if strings.HasPrefix(allPods.Items[i].Name, "etcd-") {
				pods.Items = append(pods.Items, allPods.Items[i])
			}
		}
	}

	for i := range pods.Items {
		pod := &pods.Items[i]
		if pod.Spec.NodeName == deletedNodeName {
			continue
		}

		if isPodReady(pod) {
			return pod, nil
		}
	}

	return nil, fmt.Errorf("no healthy etcd pod found (excluding node %s)", deletedNodeName)
}

// isPodReady returns true if the pod is Running and all containers are Ready.
func isPodReady(pod *v1.Pod) bool {
	if pod.Status.Phase != v1.PodRunning {
		return false
	}

	for _, cond := range pod.Status.Conditions {
		if cond.Type == v1.PodReady && cond.Status == v1.ConditionTrue {
			return true
		}
	}

	return false
}

// listEtcdMembers executes etcdctl member list and parses the JSON response.
func listEtcdMembers(ctx context.Context, clientset kubernetes.Interface, config *rest.Config, pod *v1.Pod) ([]EtcdMember, error) {
	cmd := []string{
		etcdctlPath,
		"--cacert", etcdCACert,
		"--cert", etcdClientCert,
		"--key", etcdClientKey,
		"--endpoints", etcdEndpoint,
		"member", "list", "-w", "json",
	}

	stdout, stderr, err := execInPod(ctx, clientset, config, pod.Namespace, pod.Name, etcdContainerName, cmd)
	if err != nil {
		return nil, fmt.Errorf("etcdctl member list failed: %w (stderr: %s)", err, stderr)
	}

	var resp EtcdMemberListResponse

	err = json.Unmarshal([]byte(stdout), &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse etcdctl member list output: %w (raw: %s)", err, stdout)
	}

	return resp.Members, nil
}

// removeEtcdMemberByID removes an etcd member by its hex ID.
func removeEtcdMemberByID(ctx context.Context, clientset kubernetes.Interface, config *rest.Config, pod *v1.Pod, memberID uint64) error {
	hexID := strconv.FormatUint(memberID, 16)
	cmd := []string{
		etcdctlPath,
		"--cacert", etcdCACert,
		"--cert", etcdClientCert,
		"--key", etcdClientKey,
		"--endpoints", etcdEndpoint,
		"member", "remove", hexID,
	}

	_, stderr, err := execInPod(ctx, clientset, config, pod.Namespace, pod.Name, etcdContainerName, cmd)
	if err != nil {
		return fmt.Errorf("etcdctl member remove %s failed: %w (stderr: %s)", hexID, err, stderr)
	}

	return nil
}

// execInPod executes a command in a container via the Kubernetes exec API.
func execInPod(
	ctx context.Context, clientset kubernetes.Interface, config *rest.Config,
	namespace, podName, containerName string, cmd []string,
) (string, string, error) {
	req := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&v1.PodExecOptions{
			Container: containerName,
			Command:   cmd,
			Stdout:    true,
			Stderr:    true,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return "", "", fmt.Errorf("failed to create SPDY executor: %w", err)
	}

	var stdout, stderr bytes.Buffer

	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})

	return stdout.String(), stderr.String(), err
}
