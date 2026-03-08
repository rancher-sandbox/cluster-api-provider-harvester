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

// Package metrics registers custom Prometheus metrics for CAPHV controllers.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	namespace = "caphv"
)

var (
	// Machine lifecycle metrics.

	// MachineCreateTotal counts total VM creation attempts.
	MachineCreateTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "machine_create_total",
		Help:      "Total number of HarvesterMachine VM creation attempts.",
	})

	// MachineCreateErrorsTotal counts failed VM creation attempts.
	MachineCreateErrorsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "machine_create_errors_total",
		Help:      "Total number of failed HarvesterMachine VM creation attempts.",
	})

	// MachineCreationDuration tracks VM creation duration in seconds.
	MachineCreationDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: namespace,
		Name:      "machine_creation_duration_seconds",
		Help:      "Duration of HarvesterMachine VM creation in seconds.",
		Buckets:   prometheus.ExponentialBuckets(1, 2, 10), //nolint:mnd // 1s to ~512s
	})

	// MachineDeleteTotal counts total VM deletion attempts.
	MachineDeleteTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "machine_delete_total",
		Help:      "Total number of HarvesterMachine VM deletion attempts.",
	})

	// MachineDeleteErrorsTotal counts failed VM deletion attempts.
	MachineDeleteErrorsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "machine_delete_errors_total",
		Help:      "Total number of failed HarvesterMachine VM deletion attempts.",
	})

	// MachineStatus reports the current status of managed machines.
	MachineStatus = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "machine_status",
		Help:      "Current status of HarvesterMachine (1=ready, 0=not ready).",
	}, []string{"cluster", "machine"})

	// IP pool metrics.

	// IPPoolAllocationsTotal counts total IP allocation attempts.
	IPPoolAllocationsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "ippool_allocations_total",
		Help:      "Total number of VM IP pool allocation attempts.",
	})

	// IPPoolAllocationErrorsTotal counts failed IP allocation attempts.
	IPPoolAllocationErrorsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "ippool_allocation_errors_total",
		Help:      "Total number of failed VM IP pool allocation attempts.",
	})

	// IPPoolReleasesTotal counts total IP releases.
	IPPoolReleasesTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "ippool_releases_total",
		Help:      "Total number of VM IP pool releases.",
	})

	// Cluster lifecycle metrics.

	// ClusterReconcileDuration tracks cluster reconciliation duration.
	ClusterReconcileDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Name:      "cluster_reconcile_duration_seconds",
		Help:      "Duration of HarvesterCluster reconciliation in seconds.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"operation"}) // operation: "normal" or "delete"

	// ClusterReady reports the current ready status of managed clusters.
	ClusterReady = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "cluster_ready",
		Help:      "Whether HarvesterCluster is ready (1=ready, 0=not ready).",
	}, []string{"cluster"})

	// etcd member management metrics.

	// EtcdMemberRemoveTotal counts etcd member removal attempts.
	EtcdMemberRemoveTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "etcd_member_remove_total",
		Help:      "Total number of etcd member removal attempts.",
	})

	// EtcdMemberRemoveErrorsTotal counts failed etcd member removals.
	EtcdMemberRemoveErrorsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "etcd_member_remove_errors_total",
		Help:      "Total number of failed etcd member removal attempts.",
	})

	// Node initialization metrics.

	// NodeInitTotal counts node initialization attempts.
	NodeInitTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "node_init_total",
		Help:      "Total number of workload node initialization attempts.",
	})

	// NodeInitErrorsTotal counts failed node initializations.
	NodeInitErrorsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "node_init_errors_total",
		Help:      "Total number of failed workload node initialization attempts.",
	})

	// NodeInitDuration tracks node initialization duration.
	NodeInitDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: namespace,
		Name:      "node_init_duration_seconds",
		Help:      "Duration of workload node initialization in seconds.",
		Buckets:   prometheus.ExponentialBuckets(0.5, 2, 8), //nolint:mnd // 0.5s to ~64s
	})

	// MachineReconcileDuration tracks machine reconciliation duration.
	MachineReconcileDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Name:      "machine_reconcile_duration_seconds",
		Help:      "Duration of HarvesterMachine reconciliation in seconds.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"operation"}) // operation: "normal" or "delete"
)

func init() {
	metrics.Registry.MustRegister(
		// Machine lifecycle
		MachineCreateTotal,
		MachineCreateErrorsTotal,
		MachineCreationDuration,
		MachineDeleteTotal,
		MachineDeleteErrorsTotal,
		MachineStatus,
		MachineReconcileDuration,
		// IP pool
		IPPoolAllocationsTotal,
		IPPoolAllocationErrorsTotal,
		IPPoolReleasesTotal,
		// Cluster
		ClusterReconcileDuration,
		ClusterReady,
		// etcd
		EtcdMemberRemoveTotal,
		EtcdMemberRemoveErrorsTotal,
		// Node init
		NodeInitTotal,
		NodeInitErrorsTotal,
		NodeInitDuration,
	)
}
