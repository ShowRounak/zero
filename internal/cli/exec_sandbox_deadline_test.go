package cli

import (
	"testing"
	"time"

	"github.com/Gitlawb/zero/internal/config"
	"github.com/Gitlawb/zero/internal/sandbox"
)

// TestExecSandboxPolicyDefaultUnchanged is the load-bearing guard for real users:
// WITHOUT --sandbox off / ZERO_SANDBOX, the exec policy MUST be the default
// enforcing one (workspace-confined, network denied). If this ever changes, the
// container-trusted off-switch has leaked into the default.
func TestExecSandboxPolicyDefaultUnchanged(t *testing.T) {
	policy := execSandboxPolicy(config.SandboxConfig{}, false)
	if policy.Mode != sandbox.ModeEnforce {
		t.Errorf("Mode = %q, want %q (default must not change)", policy.Mode, sandbox.ModeEnforce)
	}
	if policy.Network != sandbox.NetworkDeny {
		t.Errorf("Network = %q, want %q (default must not change)", policy.Network, sandbox.NetworkDeny)
	}
	if !policy.EnforceWorkspace {
		t.Error("EnforceWorkspace = false, want true (default must not change)")
	}
}

// TestExecSandboxPolicyOff: with the off-switch, all three confinement gates drop
// so writes-anywhere and network both work inside an isolated container.
func TestExecSandboxPolicyOff(t *testing.T) {
	policy := execSandboxPolicy(config.SandboxConfig{}, true)
	if policy.Mode != sandbox.ModeDisabled {
		t.Errorf("Mode = %q, want %q", policy.Mode, sandbox.ModeDisabled)
	}
	if policy.Network != sandbox.NetworkAllow {
		t.Errorf("Network = %q, want %q", policy.Network, sandbox.NetworkAllow)
	}
	if policy.EnforceWorkspace {
		t.Error("EnforceWorkspace = true, want false")
	}
}

func TestResolveSandboxOff(t *testing.T) {
	cases := []struct {
		flag, env string
		want      bool
	}{
		{"", "", false},
		{"off", "", true},
		{"disabled", "", true},
		{"none", "", true},
		{"0", "", true},
		{"OFF", "", true}, // case-insensitive
		{"enforce", "", false},
		{"", "off", true}, // env fallback
		{"", "disabled", true},
		{"", "1", false},          // unrecognized env value
		{"enforce", "off", false}, // flag wins over env
		{"off", "enforce", true},  // flag wins over env
	}
	for _, c := range cases {
		t.Setenv("ZERO_SANDBOX", c.env)
		if got := resolveSandboxOff(c.flag); got != c.want {
			t.Errorf("resolveSandboxOff(flag=%q, env=%q) = %v, want %v", c.flag, c.env, got, c.want)
		}
	}
}

func TestResolveDeadline(t *testing.T) {
	cases := []struct {
		flag, env string
		want      time.Duration
	}{
		{"", "", 0},
		{"900s", "", 15 * time.Minute},
		{"15m", "", 15 * time.Minute},
		{"", "300s", 5 * time.Minute}, // env fallback
		{"60s", "300s", time.Minute},  // flag wins over env
		{"bogus", "", 0},              // unparseable -> no deadline
		{"-5s", "", 0},                // non-positive -> no deadline
		{"0s", "", 0},                 // zero -> no deadline
	}
	for _, c := range cases {
		t.Setenv("ZERO_DEADLINE", c.env)
		if got := resolveDeadline(c.flag); got != c.want {
			t.Errorf("resolveDeadline(flag=%q, env=%q) = %v, want %v", c.flag, c.env, got, c.want)
		}
	}
}

// TestParseExecArgsSandboxAndDeadline: both --flag value and --flag=value spellings
// thread through to the options.
func TestParseExecArgsSandboxAndDeadline(t *testing.T) {
	opts, _, err := parseExecArgs([]string{"--sandbox", "off", "--deadline", "900s", "do the task"})
	if err != nil {
		t.Fatalf("parseExecArgs: %v", err)
	}
	if opts.sandboxMode != "off" {
		t.Errorf("sandboxMode = %q, want off", opts.sandboxMode)
	}
	if opts.deadlineRaw != "900s" {
		t.Errorf("deadlineRaw = %q, want 900s", opts.deadlineRaw)
	}

	opts2, _, err := parseExecArgs([]string{"--sandbox=disabled", "--deadline=15m", "x"})
	if err != nil {
		t.Fatalf("parseExecArgs inline: %v", err)
	}
	if opts2.sandboxMode != "disabled" || opts2.deadlineRaw != "15m" {
		t.Errorf("inline parse: sandboxMode=%q deadlineRaw=%q, want disabled/15m", opts2.sandboxMode, opts2.deadlineRaw)
	}
}
