// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026-present Datadog, Inc.

//go:build kubeapiserver

package spot

import (
	"context"
	"errors"
	"fmt"

	"k8s.io/utils/clock"

	workloadmeta "github.com/DataDog/datadog-agent/comp/core/workloadmeta/def"
	"github.com/DataDog/datadog-agent/pkg/aggregator/sender"
	pkgconfigsetup "github.com/DataDog/datadog-agent/pkg/config/setup"
	"github.com/DataDog/datadog-agent/pkg/util/kubernetes/apiserver"
)

const senderID = "cluster_autoscaling_spot"

// StartSpotScheduling creates and starts the spot scheduler, returning a PodHandler for use in the admission webhook.
func StartSpotScheduling(ctx context.Context, wlm workloadmeta.Component, apiCl *apiserver.APIClient, isLeaderFunc func() bool, senderManager sender.SenderManager) (PodHandler, error) {
	if apiCl == nil {
		return nil, errors.New("impossible to start spot scheduling without valid APIClient")
	}

	localSender, err := senderManager.GetSender(senderID)
	if err != nil {
		return nil, fmt.Errorf("unable to get sender for spot scheduling: %w", err)
	}
	localSender.DisableDefaultHostname(true)

	cfg := ReadConfig(pkgconfigsetup.Datadog())
	tel := newTelemetry(localSender)
	s := newScheduler(cfg, clock.RealClock{}, wlm,
		newKubePodEvictor(apiCl.Cl),
		newKubeWorkloadPatcher(apiCl.DynamicInformerCl),
		apiCl.DynamicInformerCl,
		newWLMPodLister(wlm),
		isLeaderFunc,
		tel)
	s.Start(ctx)

	return s, nil
}
