// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025-present Datadog, Inc.

//go:build windows

// Package windowseventlogchannels provides an issue module for Windows Event Log channel misconfiguration.
package windowseventlogchannels

import (
	"github.com/DataDog/agent-payload/v5/healthplatform"
	"github.com/DataDog/datadog-agent/comp/core/config"
	"github.com/DataDog/datadog-agent/comp/healthplatform/issues"
)

func init() {
	issues.RegisterModuleFactory(NewModule)
}

const (
	// CheckID is the unique identifier for the built-in health check
	CheckID = "windows-eventlog-channels"

	// CheckName is the human-readable name for the health check
	CheckName = "Windows Event Log Channels"
)

// windowsEventLogChannelsModule implements issues.Module
type windowsEventLogChannelsModule struct {
	template *WindowsEventLogChannelsIssue
	conf     config.Component
}

// NewModule creates a new Windows Event Log channels issue module
func NewModule(conf config.Component) issues.Module {
	return &windowsEventLogChannelsModule{
		template: NewWindowsEventLogChannelsIssue(),
		conf:     conf,
	}
}

// IssueID returns the unique identifier for this issue type
func (m *windowsEventLogChannelsModule) IssueID() string {
	return IssueID
}

// IssueTemplate returns the template for building complete issues
func (m *windowsEventLogChannelsModule) IssueTemplate() issues.IssueTemplate {
	return m.template
}

// BuiltInHealthCheck returns the built-in health check configuration.
// Once is true so this check runs only once at startup.
func (m *windowsEventLogChannelsModule) BuiltInHealthCheck() *issues.BuiltInHealthCheck {
	return &issues.BuiltInHealthCheck{
		ID:   CheckID,
		Name: CheckName,
		CheckFn: func() (*healthplatform.IssueReport, error) {
			return Check(m.conf)
		},
		Once: true,
	}
}
