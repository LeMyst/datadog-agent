// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026-present Datadog, Inc.

//go:build kubeapiserver

package spot

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/DataDog/datadog-agent/pkg/aggregator/mocksender"
	checkid "github.com/DataDog/datadog-agent/pkg/collector/check/id"
	"github.com/DataDog/datadog-agent/pkg/util/kubernetes"
)

// mockAdapter adapts assert.CollectT to mock.TestingT.
type mockAdapter struct {
	*testing.T
	*assert.CollectT
}

// newTestTelemetry returns a telemetry instance backed by a mock sender for assertions.
func newTestTelemetry(t *testing.T) (*telemetry, *mocksender.MockSender) {
	t.Helper()
	s := mocksender.NewMockSender(checkid.ID("test-spot-" + t.Name()))
	s.SetupAcceptAll()
	return newTelemetry(s), s
}

func TestObserveWorkloadCounts(t *testing.T) {
	tel, s := newTestTelemetry(t)

	tel.observeWorkloadCounts(map[string]int{"Deployment": 3, "StatefulSet": 1})
	assert.EventuallyWithT(t, func(ct *assert.CollectT) {
		mt := &mockAdapter{t, ct}
		s.AssertCalled(mt, "Gauge", metricNameWorkloads, 3.0, "", []string{"workload_kind:deployment"})
		s.AssertCalled(mt, "Gauge", metricNameWorkloads, 1.0, "", []string{"workload_kind:statefulset"})
	}, time.Second, time.Millisecond)

	// Updating drops vanished kinds: StatefulSet should be re-emitted as zero.
	s.ResetCalls()
	tel.observeWorkloadCounts(map[string]int{"Deployment": 5})
	assert.EventuallyWithT(t, func(ct *assert.CollectT) {
		mt := &mockAdapter{t, ct}
		s.AssertCalled(mt, "Gauge", metricNameWorkloads, 5.0, "", []string{"workload_kind:deployment"})
		s.AssertCalled(mt, "Gauge", metricNameWorkloads, 0.0, "", []string{"workload_kind:statefulset"})
	}, time.Second, time.Millisecond)
}

func TestObserveWorkload(t *testing.T) {
	tel, s := newTestTelemetry(t)
	o := objectRef{Kind: kubernetes.DeploymentKind, Namespace: "ns", Name: t.Name()}

	tel.observeWorkload(o, workloadSnapshot{spot: 7, onDemand: 3, excessSpot: 1, excessOnDemand: 0})

	baseTags := []string{
		"kube_namespace:ns",
		"kube_deployment:" + t.Name(),
	}
	spotTags := append(append([]string{}, baseTags...), "capacity_type:spot")
	onDemandTags := append(append([]string{}, baseTags...), "capacity_type:on_demand")
	assert.EventuallyWithT(t, func(ct *assert.CollectT) {
		mt := &mockAdapter{t, ct}
		s.AssertCalled(mt, "Gauge", metricNamePods, 7.0, "", spotTags)
		s.AssertCalled(mt, "Gauge", metricNamePods, 3.0, "", onDemandTags)
		s.AssertCalled(mt, "Gauge", metricNameExcessPods, 1.0, "", spotTags)
		s.AssertCalled(mt, "Gauge", metricNameExcessPods, 0.0, "", onDemandTags)
	}, time.Second, time.Millisecond)
}

func TestObserveSchedulingDelay(t *testing.T) {
	tel, s := newTestTelemetry(t)

	tel.observeSchedulingDelay(12 * time.Second)
	tel.observeSchedulingDelay(45 * time.Second)
	assert.EventuallyWithT(t, func(ct *assert.CollectT) {
		mt := &mockAdapter{t, ct}
		s.AssertCalled(mt, "Histogram", metricNameSchedulingDelay, 12.0, "", []string(nil))
		s.AssertCalled(mt, "Histogram", metricNameSchedulingDelay, 45.0, "", []string(nil))
	}, time.Second, time.Millisecond)
}

func TestObserveActiveFallbacks(t *testing.T) {
	tel, s := newTestTelemetry(t)

	tel.observeActiveFallbacks(map[string]int{"Deployment": 2})
	assert.EventuallyWithT(t, func(ct *assert.CollectT) {
		mt := &mockAdapter{t, ct}
		s.AssertCalled(mt, "Gauge", metricNameActiveFallbacks, 2.0, "", []string{"workload_kind:deployment"})
		s.AssertCalled(mt, "Gauge", metricNameActiveFallbacks, 0.0, "", []string{"workload_kind:statefulset"})
	}, time.Second, time.Millisecond)
}

func TestObserveFallback(t *testing.T) {
	tel, s := newTestTelemetry(t)
	o := objectRef{Kind: kubernetes.DeploymentKind, Namespace: "ns", Name: t.Name()}

	tel.observeFallback(o)
	tel.observeFallback(o)

	tags := []string{"kube_namespace:ns", "kube_deployment:" + t.Name()}
	assert.EventuallyWithT(t, func(ct *assert.CollectT) {
		mt := &mockAdapter{t, ct}
		s.AssertNumberOfCalls(mt, "Count", 2)
		s.AssertCalled(mt, "Count", metricNameFallbacks, 1.0, "", tags)
	}, time.Second, time.Millisecond)
}

func TestObserveRebalanceEviction(t *testing.T) {
	tel, s := newTestTelemetry(t)
	o := objectRef{Kind: kubernetes.DeploymentKind, Namespace: "ns", Name: t.Name()}

	tel.observeRebalanceEviction(o, true)

	tags := []string{"kube_namespace:ns", "kube_deployment:" + t.Name()}
	assert.EventuallyWithT(t, func(ct *assert.CollectT) {
		mt := &mockAdapter{t, ct}
		s.AssertCalled(mt, "Count", metricNameRebalanceEvictions, 1.0, "", append(tags, "capacity_type:spot"))
	}, time.Second, time.Millisecond)
}
