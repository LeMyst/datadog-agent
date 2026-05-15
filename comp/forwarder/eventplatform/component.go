// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

// Package eventplatform contains the logic for forwarding events to the event platform
// Deprecated: use comp/forwarder/eventplatform/def instead.
package eventplatform

import (
	def "github.com/DataDog/datadog-agent/comp/forwarder/eventplatform/def"
)

// team: agent-log-pipelines

const (
	// EventTypeNetworkDevicesMetadata is the event type for network devices metadata
	// Deprecated: use comp/forwarder/eventplatform/def instead.
	EventTypeNetworkDevicesMetadata = def.EventTypeNetworkDevicesMetadata

	// EventTypeSnmpTraps is the event type for snmp traps
	// Deprecated: use comp/forwarder/eventplatform/def instead.
	EventTypeSnmpTraps = def.EventTypeSnmpTraps

	// EventTypeNetworkDevicesNetFlow is the event type for network devices NetFlow data
	// Deprecated: use comp/forwarder/eventplatform/def instead.
	EventTypeNetworkDevicesNetFlow = def.EventTypeNetworkDevicesNetFlow

	// EventTypeNetworkPath is the event type for network devices Network Path data
	// Deprecated: use comp/forwarder/eventplatform/def instead.
	EventTypeNetworkPath = def.EventTypeNetworkPath

	// EventTypeSynthetics is the event type for Synthetics test results
	// Deprecated: use comp/forwarder/eventplatform/def instead.
	EventTypeSynthetics = def.EventTypeSynthetics

	// EventTypeNetworkConfigManagement is the event type for network device configuration management
	// Deprecated: use comp/forwarder/eventplatform/def instead.
	EventTypeNetworkConfigManagement = def.EventTypeNetworkConfigManagement

	// EventTypeContainerLifecycle represents a container lifecycle event
	// Deprecated: use comp/forwarder/eventplatform/def instead.
	EventTypeContainerLifecycle = def.EventTypeContainerLifecycle

	// EventTypeContainerImages represents a container images event
	// Deprecated: use comp/forwarder/eventplatform/def instead.
	EventTypeContainerImages = def.EventTypeContainerImages

	// EventTypeContainerSBOM represents a container SBOM event
	// Deprecated: use comp/forwarder/eventplatform/def instead.
	EventTypeContainerSBOM = def.EventTypeContainerSBOM

	// EventTypeSoftwareInventory represents a software inventory event
	// Deprecated: use comp/forwarder/eventplatform/def instead.
	EventTypeSoftwareInventory = def.EventTypeSoftwareInventory

	// EventTypeEventManagement represents an event for the Event Management API
	// Deprecated: use comp/forwarder/eventplatform/def instead.
	EventTypeEventManagement = def.EventTypeEventManagement

	// EventTypeKubeActions represents a kubernetes action result event
	// Deprecated: use comp/forwarder/eventplatform/def instead.
	EventTypeKubeActions = def.EventTypeKubeActions
)

// Component is the interface of the event platform forwarder component.
// Deprecated: use comp/forwarder/eventplatform/def instead.
type Component = def.Component

// Forwarder is the interface of the event platform forwarder.
// Deprecated: use comp/forwarder/eventplatform/def instead.
type Forwarder = def.Forwarder
