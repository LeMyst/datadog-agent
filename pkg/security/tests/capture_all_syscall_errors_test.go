// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux && functionaltests

// Package tests holds tests related files
package tests

import (
	"context"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/DataDog/datadog-agent/pkg/security/secl/model"
	"github.com/DataDog/datadog-agent/pkg/security/secl/rules"
)

// TestCaptureAllSyscallErrors verifies that with
// runtime_security_config.syscalls.capture_all_errors.enabled set to true,
// chmod() and open() syscalls failing with ENOENT (normally filtered by
// IS_UNHANDLED_ERROR) still produce events in userspace.
func TestCaptureAllSyscallErrors(t *testing.T) {
	SkipIfNotAvailable(t)

	ruleDefs := []*rules.RuleDefinition{
		// To access the path, we need to use the syscall context instead of the chmod event
		{
			ID:         "test_chmod_capture_enoent",
			Expression: `chmod.syscall.path == "{{.Root}}/does-not-exist" && chmod.retval == ENOENT`,
		},
		{
			ID:         "test_open_capture_enoent",
			Expression: `open.syscall.path == "{{.Root}}/does-not-exist" && open.retval == ENOENT`,
		},
	}

	test, err := newTestModule(t, nil, ruleDefs, withStaticOpts(testOpts{
		captureAllSyscallErrorsEnabled: true,
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer test.Close()

	syscallTester, err := loadSyscallTester(t, test, "syscall_tester")
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(test.Root(), "does-not-exist")

	t.Run("chmod-enoent", func(t *testing.T) {
		test.WaitSignalFromRule(t, func() error {
			return runSyscallTesterFunc(context.Background(), t, syscallTester, "chmod-error", path)
		}, func(event *model.Event, rule *rules.Rule) {
			assertTriggeredRule(t, rule, "test_chmod_capture_enoent")
			assert.Equal(t, "chmod", event.GetType(), "wrong event type")
			assert.Equal(t, -int64(syscall.ENOENT), event.Chmod.Retval, "wrong retval")
		}, "test_chmod_capture_enoent")
	})
	t.Run("open-enoent", func(t *testing.T) {
		test.WaitSignalFromRule(t, func() error {
			return runSyscallTesterFunc(context.Background(), t, syscallTester, "open-error", path)
		}, func(event *model.Event, rule *rules.Rule) {
			assertTriggeredRule(t, rule, "test_open_capture_enoent")
			assert.Equal(t, "open", event.GetType(), "wrong event type")
			assert.Equal(t, -int64(syscall.ENOENT), event.Open.Retval, "wrong retval")
		}, "test_open_capture_enoent")
	})

}

func TestCaptureAllSyscallErrorsDisabledByDefault(t *testing.T) {
	SkipIfNotAvailable(t)

	ruleDefs := []*rules.RuleDefinition{
		{
			ID:         "test_chmod_capture_enoent",
			Expression: `chmod.syscall.path == "{{.Root}}/does-not-exist" && chmod.retval == ENOENT`,
		},
		{
			ID:         "test_open_capture_enoent",
			Expression: `open.syscall.path == "{{.Root}}/does-not-exist" && open.retval == ENOENT`,
		},
	}

	// captureAllSyscallErrorsEnabled is intentionally left at its zero value (false)
	test, err := newTestModule(t, nil, ruleDefs)
	if err != nil {
		t.Fatal(err)
	}
	defer test.Close()

	syscallTester, err := loadSyscallTester(t, test, "syscall_tester")
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(test.Root(), "does-not-exist")

	t.Run("chmod-enoent-dropped", func(t *testing.T) {
		_ = test.GetSignal(t, func() error {
			return runSyscallTesterFunc(context.Background(), t, syscallTester, "chmod-error", path)
		}, func(_ *model.Event, rule *rules.Rule) {
			t.Errorf("unexpected event for chmod ENOENT (rule %q): the kernel should have dropped it", rule.ID)
		})
	})

	t.Run("open-enoent-dropped", func(t *testing.T) {
		_ = test.GetSignal(t, func() error {
			return runSyscallTesterFunc(context.Background(), t, syscallTester, "open-error", path)
		}, func(_ *model.Event, rule *rules.Rule) {
			t.Errorf("unexpected event for open ENOENT (rule %q): the kernel should have dropped it", rule.ID)
		})
	})
}
