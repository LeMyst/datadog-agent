// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package installer

import (
	"fmt"
	"math/rand/v2"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// launcherPreloadPath is the path verifySharedLib() exercises and the path
// written into /etc/ld.so.preload by the datadog-apm-inject service's
// instrument-start command.
var launcherPreloadPath = filepath.Join(injectOCIPath, "stable", "inject", "launcher.preload.so")

// crashyConstructorUnconditionalSrc is a tiny C source compiled into a
// shared library that crashes any program that loads it via any
// mechanism: the ELF constructor calls _exit(1) regardless of whether
// the lib was injected via LD_PRELOAD env var or /etc/ld.so.preload.
// This is the variant required to exercise the boot-time brick scenario
// — on the next boot, init(1) is launched by the kernel with no
// LD_PRELOAD env var, but ld.so still consults /etc/ld.so.preload, so
// only the unconditional crash reproduces the production failure mode.
// The guarded variant (crashyConstructorGuardedSrc in
// package_apm_inject_test.go) is used by in-process tests that cannot
// afford to kill their own shell chain; this one is only safe inside a
// test that goes through a real reboot before any process tries to use
// the lib.
const crashyConstructorUnconditionalSrc = `#include <unistd.h>
__attribute__((constructor)) static void crash(void) { _exit(1); }
`

// reboot reboots the host and waits for SSH to come back on a different boot
// id. The boot id is the source of truth: the reboot command's exit status is
// unreliable because sshd can be torn down before the SSH session returns.
func (s *packageApmInjectSuite) reboot() {
	s.T().Helper()
	host := s.Env().RemoteHost

	bootIDBefore := strings.TrimSpace(host.MustExecute("cat /proc/sys/kernel/random/boot_id"))
	s.T().Logf("rebooting host (boot id before: %s)", bootIDBefore)

	// `--no-block` queues the reboot job and returns immediately. We do not
	// require this call to succeed: on some images sshd is killed before the
	// SSH session returns, which surfaces as a transport / dial error here.
	// Whether the command returned cleanly or dropped the connection, the
	// boot-id check below is what tells us the host actually rebooted.
	if _, err := host.Execute("sudo systemctl --no-block reboot"); err != nil {
		s.T().Logf("reboot command returned an error (likely sshd torn down before reply): %v", err)
	}

	// Give sshd time to actually go down so we don't reconnect to the old
	// boot. 30s is conservative; on fast images this is wasted, on slow
	// images (cloud-init, snapshot restores) 10s is not always enough.
	time.Sleep(30 * time.Second)

	require.Eventually(s.T(), func() bool {
		if err := host.Reconnect(); err != nil {
			s.T().Logf("reconnect failed, will retry: %v", err)
			return false
		}
		out, err := host.Execute("cat /proc/sys/kernel/random/boot_id")
		if err != nil {
			s.T().Logf("boot id read failed after reconnect, will retry: %v", err)
			return false
		}
		bootIDAfter := strings.TrimSpace(out)
		if bootIDAfter == bootIDBefore {
			s.T().Logf("boot id unchanged (%s), host still on old boot", bootIDAfter)
			return false
		}
		s.T().Logf("host back up (boot id after: %s)", bootIDAfter)
		return true
	}, 10*time.Minute, 15*time.Second, "host did not reboot within timeout")
}

// requireSystemd skips the test if systemd is not PID 1. The
// datadog-apm-inject.service is only installed on systemd hosts.
func (s *packageApmInjectSuite) requireSystemd() {
	s.T().Helper()
	if _, err := s.Env().RemoteHost.Execute("test \"$(cat /proc/1/comm 2>/dev/null)\" = systemd"); err != nil {
		s.T().Skip("systemd is not running as PID 1 on this host")
	}
}

// TestSystemdServiceReboot verifies the boot-time instrumentation contract on
// a healthy host: after a reboot, the systemd service runs ExecStart, restores
// /etc/ld.so.preload, and tracer injection still produces traces end-to-end.
//
// Without the service, /etc/ld.so.preload would only be written at install
// time and rebooting after a clean shutdown (which clears it via ExecStop)
// would leave the host uninstrumented until the next install action.
func (s *packageApmInjectSuite) TestSystemdServiceReboot() {
	s.requireSystemd()

	s.RunInstallScript("DD_APM_INSTRUMENTATION_ENABLED=host", "DD_APM_INSTRUMENTATION_LIBRARIES=python")
	defer s.Purge()

	s.host.WaitForUnitActive(s.T(), "datadog-apm-inject.service", "datadog-agent.service", "datadog-agent-trace.service")
	s.assertLDPreloadInstrumented(injectOCIPath)

	s.reboot()

	// After reboot, ExecStop ran during shutdown (clearing ld.so.preload) and
	// ExecStart ran on boot (re-adding the entry). The service unit must be
	// active and the file must contain the injector path again.
	s.host.WaitForUnitActive(s.T(), "datadog-apm-inject.service", "datadog-agent.service", "datadog-agent-trace.service")

	state := s.host.State()
	state.AssertFileExists("/etc/systemd/system/datadog-apm-inject.service", 0644, "root", "root")
	state.AssertUnitsEnabled("datadog-apm-inject.service")
	state.AssertUnitsActive("datadog-apm-inject.service")
	s.assertLDPreloadInstrumented(injectOCIPath)
	s.assertSocketPath()

	// End-to-end check: the tracer is injected into a freshly-spawned process
	// and the resulting trace lands in fakeintake.
	s.host.StartExamplePythonApp()
	defer s.host.StopExamplePythonApp()
	traceID := rand.Uint64()
	s.host.CallExamplePythonApp(strconv.FormatUint(traceID, 10))
	s.assertTraceReceived(traceID)
}

// TestSystemdServiceRebootBrokenInjector verifies the safety property the
// service exists to provide: if the injector library on disk is replaced by a
// .so that crashes any program that LD_PRELOADs it, a reboot still leaves the
// host usable.
//
// The contract is: ExecStart and ExecStop invoke the static datadog-installer
// binary directly (no /bin/sh wrapper, no dynamic-linker dependency). On the
// reboot's shutdown, ExecStop runs `installer apm instrument-stop host`
// without going through ld.so, so it clears /etc/ld.so.preload regardless of
// the broken .so on disk. On the next boot, /etc/ld.so.preload is empty, so
// the kernel never tries to LD_PRELOAD the crashy lib for init(1) or any
// other process — the system comes up normally. Then ExecStart runs,
// verifySharedLib forks `echo 1` with `LD_PRELOAD=launcher.preload.so`, the
// .so's constructor calls _exit(1), echo exits non-zero, verifySharedLib
// returns an error, instrument-start exits non-zero, the unit lands in
// failed state, and /etc/ld.so.preload is left empty.
//
// This is the failure mode the unit file's no-shell design protects against.
// If ExecStop went through /bin/sh (dynamically linked), the crashy LD_PRELOAD
// would kill sh before it could exec the installer — /etc/ld.so.preload would
// never get cleared, and on the next boot init(1) itself would die when
// ld.so tried to LD_PRELOAD the broken .so. Kernel panic, host bricked.
func (s *packageApmInjectSuite) TestSystemdServiceRebootBrokenInjector() {
	s.requireSystemd()

	s.RunInstallScript("DD_APM_INSTRUMENTATION_ENABLED=host", "DD_APM_INSTRUMENTATION_LIBRARIES=python")
	defer s.Purge()

	host := s.Env().RemoteHost
	s.host.WaitForUnitActive(s.T(), "datadog-apm-inject.service")
	s.assertLDPreloadInstrumented(injectOCIPath)

	// Replace the launcher with a real, loadable, crashy .so. We move the
	// original aside (rename, no truncation) so the build step writes to a
	// fresh inode and cannot SIGBUS any process that still has the original
	// mmapped. Restoring the original in a defer keeps the host sane for
	// Purge() and any retries.
	host.MustExecute(fmt.Sprintf("sudo mv %[1]s %[1]s.bak", launcherPreloadPath))
	s.buildCrashyInjectorSO(launcherPreloadPath, crashyConstructorUnconditionalSrc)
	defer host.Execute(fmt.Sprintf("sudo mv -f %[1]s.bak %[1]s 2>/dev/null || true", launcherPreloadPath)) //nolint:errcheck

	s.reboot()

	// Reaching here means SSH came back: the host booted past init(1) with a
	// crashy .so on disk. That's only possible because ExecStop on the
	// previous shutdown cleared /etc/ld.so.preload despite the broken lib —
	// the property the static, sh-less unit file provides.
	out, err := host.Execute("uname -a && id && /bin/true")
	require.NoError(s.T(), err, "host is not usable after reboot with broken injector")
	require.NotEmpty(s.T(), out)

	// The new boot's ExecStart fired verifySharedLib, the LD_PRELOAD=lib echo
	// subprocess exited 1 (constructor _exit), the command exited non-zero,
	// the unit is failed.
	assert.Eventually(s.T(), func() bool {
		_, err := host.Execute("systemctl is-failed --quiet datadog-apm-inject.service")
		return err == nil
	}, 90*time.Second, 2*time.Second,
		"datadog-apm-inject.service did not enter failed state after reboot. status:\n%s\nlogs:\n%s",
		host.MustExecute("systemctl status datadog-apm-inject.service --no-pager || true"),
		host.MustExecute("sudo journalctl -xeu datadog-apm-inject.service --no-pager || true"),
	)

	// /etc/ld.so.preload must remain empty: ExecStop cleared it on the way
	// down, and the failed ExecStart never wrote it back. This is the core
	// guarantee — a broken injector on disk does not brick subsequent boots.
	s.assertLDPreloadNotInstrumented()

	// The agent itself is unaffected — it does not depend on the inject
	// service for its own operation.
	s.host.WaitForUnitActive(s.T(), "datadog-agent.service", "datadog-agent-trace.service")
}
