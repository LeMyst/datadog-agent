// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026-present Datadog, Inc.

//go:build kubeapiserver

package spot

import (
	"strings"
	"time"

	"github.com/DataDog/datadog-agent/pkg/aggregator/sender"
)

const (
	metricPrefix = "datadog.cluster_agent.autoscaling.cluster.spot."

	metricNamePods               = metricPrefix + "pods"
	metricNameExcessPods         = metricPrefix + "excess_pods"
	metricNameWorkloads          = metricPrefix + "workloads"
	metricNameFallbacks          = metricPrefix + "fallbacks"
	metricNameActiveFallbacks    = metricPrefix + "active_fallbacks"
	metricNameRebalanceEvictions = metricPrefix + "rebalance_evictions"
	metricNameSchedulingDelay    = metricPrefix + "scheduling_delay_seconds"
)

const (
	capacityTypeSpot     = "spot"
	capacityTypeOnDemand = "on_demand"
)

// workloadSnapshot is the per-workload aggregate telemetry emits.
type workloadSnapshot struct {
	spot, onDemand, excessSpot, excessOnDemand int
}

// telemetry emits the spot scheduler's metrics.
//
// All observe* methods run the sender calls in a goroutine to avoid blocking callers.
type telemetry struct {
	sender sender.Sender
}

func newTelemetry(s sender.Sender) *telemetry {
	return &telemetry{sender: s}
}

// observeWorkload updates pod gauges for a workload.
func (t *telemetry) observeWorkload(o objectRef, snap workloadSnapshot) {
	go func() {
		baseTags := workloadTags(o)
		spotTags := append(append([]string{}, baseTags...), "capacity_type:"+capacityTypeSpot)
		onDemandTags := append(append([]string{}, baseTags...), "capacity_type:"+capacityTypeOnDemand)
		t.sender.Gauge(metricNamePods, float64(snap.spot), "", spotTags)
		t.sender.Gauge(metricNamePods, float64(snap.onDemand), "", onDemandTags)
		t.sender.Gauge(metricNameExcessPods, float64(snap.excessSpot), "", spotTags)
		t.sender.Gauge(metricNameExcessPods, float64(snap.excessOnDemand), "", onDemandTags)
		t.sender.Commit()
	}()
}

// observeFallback records one fallback event for a workload.
func (t *telemetry) observeFallback(o objectRef) {
	go func() {
		t.sender.Count(metricNameFallbacks, 1, "", workloadTags(o))
		t.sender.Commit()
	}()
}

// observeRebalanceEviction records one pod eviction by the rebalancer for a workload.
func (t *telemetry) observeRebalanceEviction(o objectRef, isSpot bool) {
	go func() {
		capacityType := capacityTypeOnDemand
		if isSpot {
			capacityType = capacityTypeSpot
		}
		tags := append(workloadTags(o), "capacity_type:"+capacityType)
		t.sender.Count(metricNameRebalanceEvictions, 1, "", tags)
		t.sender.Commit()
	}()
}

// observeSchedulingDelay records the time a spot pod spent in the Pending phase.
func (t *telemetry) observeSchedulingDelay(d time.Duration) {
	go func() {
		t.sender.Histogram(metricNameSchedulingDelay, d.Seconds(), "", nil)
		t.sender.Commit()
	}()
}

// observeWorkloadCounts updates workload kind-count gauges
// including an explicit zero for kinds with no managed workloads.
func (t *telemetry) observeWorkloadCounts(byKind map[string]int) {
	go func() {
		for _, r := range spotWorkloadResources {
			t.sender.Gauge(metricNameWorkloads, float64(byKind[r.kind]), "", kindTags(r.kind))
		}
		t.sender.Commit()
	}()
}

// observeActiveFallbacks updates the active-fallback count gauges per kind,
// including an explicit zero for kinds with no workloads in fallback.
func (t *telemetry) observeActiveFallbacks(byKind map[string]int) {
	go func() {
		for _, r := range spotWorkloadResources {
			t.sender.Gauge(metricNameActiveFallbacks, float64(byKind[r.kind]), "", kindTags(r.kind))
		}
		t.sender.Commit()
	}()
}

func workloadTags(o objectRef) []string {
	kind := strings.ToLower(o.Kind)
	return []string{
		"kube_namespace:" + o.Namespace,
		"kube_" + kind + ":" + o.Name,
	}
}

func kindTags(kind string) []string {
	return []string{
		"workload_kind:" + strings.ToLower(kind),
	}
}
