package main

import (
	"errors"
	"fmt"
	"io/fs"
	"runtime"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/MustardSeedNetworks/niac-go/internal/attachpolicy"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

// resolveAttachmentPolicies is the daemon's approved set: the
// --attachment-policy flags merged with the policy file.
//
// A policy file the daemon cannot trust (anyone but root could have written
// it) or cannot parse approves nothing and is logged loudly; the daemon starts
// on its flags alone, which can only narrow what it approves. Flags that
// contradict the file are an operator mistake and stop the start, as two
// contradictory flags do.
func resolveAttachmentPolicies(flagValues []string) ([]attachpolicy.Policy, error) {
	return mergeAttachmentPolicies(attachpolicy.DefaultPath(), attachpolicy.OperatorTrust(), flagValues)
}

func mergeAttachmentPolicies(
	path string, trust attachpolicy.Trust, flagValues []string,
) ([]attachpolicy.Policy, error) {
	fromFlags, err := attachpolicy.ParseAll(flagValues)
	if err != nil {
		return nil, err
	}

	fromFile, err := attachpolicy.Load(path, trust)
	if errors.Is(err, fs.ErrPermission) {
		logging.Warningf(
			"Cannot read attachment policy file %s as this user (%v); "+
				"only --attachment-policy flags approve an interface.", path, err,
		)
		return fromFlags, nil
	}
	if err != nil {
		logging.Errorf(
			"Ignoring attachment policy file %s: %v. Fix it, then restart the daemon; "+
				"until then only --attachment-policy flags approve an interface.", path, err,
		)
		return fromFlags, nil
	}
	if len(fromFile) > 0 {
		logging.Infof("Loaded %d attachment policies from %s", len(fromFile), path)
	}

	merged := attachpolicy.Merge(fromFlags, fromFile)
	if err = attachpolicy.Validate(merged); err != nil {
		return nil, fmt.Errorf(
			"--attachment-policy flags and %s contradict each other: %w", path, err,
		)
	}
	return merged, nil
}

// policyEditor holds what the attachment-policy commands reach outside
// themselves for, so tests can use a temporary file and a stand-in for root.
type policyEditor struct {
	path       string
	trust      attachpolicy.Trust
	privileged func() bool
}

func addAttachmentPolicyCommand(root *cobra.Command) {
	root.AddCommand(newAttachmentPolicyCmd(policyEditor{
		path:       attachpolicy.DefaultPath(),
		trust:      attachpolicy.OperatorTrust(),
		privileged: isPrivileged,
	}))
}

func newAttachmentPolicyCmd(editor policyEditor) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "attachment-policy",
		Short: "Approve the host interfaces simulations may bind",
		Long: `Manage the root-owned policy file listing the physical attachments the daemon
approves: which host interface a simulation may bind, in which mode, on which
VLANs. A scenario that routes through a host interface fails preflight with
attachment_policy_denied until its interface is approved here or with a
--attachment-policy flag.

The file is attachment-policy.conf in the daemon's configuration directory:
/usr/local/niac on macOS, /etc/niac on Linux, %ProgramData%\NiAC on Windows.
It holds one policy per line in the --attachment-policy syntax
(` + attachpolicy.Syntax + `); blank lines and lines starting with # are ignored.
The daemon reads it at start, so a change needs a daemon restart, and any
running simulation stops with it. It merges with --attachment-policy flags:
the same policy in both counts once, a contradiction stops the daemon. A file
anyone but root (SYSTEM or Administrators on Windows) could have written is
ignored. There is no API to change it: an API client must not approve its own
NIC.`,
		Example: `  # Approve a directly connected tester on en16, then restart the daemon
  sudo niac attachment-policy add en16=direct

  # See what the file approves
  niac attachment-policy list`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	cmd.AddCommand(
		newPolicyListCmd(editor),
		newPolicyAddCmd(editor),
		newPolicyRemoveCmd(editor),
	)
	return cmd
}

func newPolicyListCmd(editor policyEditor) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List the policies in the policy file",
		Long: `List the policies in the policy file. Policies passed to the daemon as flags
are not in the file; 'GET /api/v1/attachment-policies' lists what the running
daemon actually approved. On Linux the package makes /etc/niac readable only
by root and the niac service account, so run it with sudo there.`,
		Example: `  niac attachment-policy list`,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			policies, err := attachpolicy.Load(editor.path, editor.trust)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if len(policies) == 0 {
				_, _ = fmt.Fprintf(out, "No attachment policies in %s.\n", editor.path)
				return nil
			}
			for _, policy := range policies {
				_, _ = fmt.Fprintln(out, attachpolicy.Format(policy))
			}
			return nil
		},
	}
}

func newPolicyAddCmd(editor policyEditor) *cobra.Command {
	return &cobra.Command{
		Use:   "add <POLICY>",
		Short: "Approve an interface in the policy file",
		Long: `Add one policy to the policy file, creating the file if needed. Run as root
(an elevated shell on Windows). Adding a policy the file already holds changes
nothing.`,
		Example: `  sudo niac attachment-policy add en16=direct
  sudo niac attachment-policy add eth0=access:200
  sudo niac attachment-policy add eth0=trunk:200,201,202`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return editor.run(cmd, args[0], func(lines []string, entry string) ([]string, string, error) {
				if slices.ContainsFunc(lines, func(line string) bool { return sameEntry(line, entry) }) {
					return lines, fmt.Sprintf("%s is already in %s.", entry, editor.path), nil
				}
				return append(lines, entry), fmt.Sprintf("Added %s to %s.", entry, editor.path), nil
			})
		},
	}
}

func newPolicyRemoveCmd(editor policyEditor) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <POLICY>",
		Short: "Withdraw an approval from the policy file",
		Long: `Remove one policy from the policy file. Run as root (an elevated shell on
Windows). The argument must match a line of the file exactly as 'list' prints it;
a policy the daemon was given as a flag is not in the file and is removed by
dropping the flag.`,
		Example: `  sudo niac attachment-policy remove en16=direct`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return editor.run(cmd, args[0], func(lines []string, entry string) ([]string, string, error) {
				kept := slices.DeleteFunc(slices.Clone(lines), func(line string) bool {
					return sameEntry(line, entry)
				})
				if len(kept) == len(lines) {
					return nil, "", fmt.Errorf("%s is not in %s", entry, editor.path)
				}
				return kept, fmt.Sprintf("Removed %s from %s.", entry, editor.path), nil
			})
		},
	}
}

// run validates the argument, requires privilege, applies change to the file
// and tells the operator what is left to do. change returns the new lines and
// the message to print; returning the lines it was given writes nothing new
// but still reports.
func (e policyEditor) run(
	cmd *cobra.Command,
	value string,
	change func(lines []string, entry string) ([]string, string, error),
) error {
	policy, err := attachpolicy.Parse(value)
	if err != nil {
		return withExitCode(exitUsage, err)
	}
	if !e.privileged() {
		return fmt.Errorf(
			"%s changes a root-owned file and must run as %s: %s",
			cmd.CommandPath(), privilegedAccount(), rerunCommand(cmd, value),
		)
	}

	entry := attachpolicy.Format(policy)
	var message string
	changed := false
	if err = attachpolicy.Edit(e.path, e.trust, func(lines []string) ([]string, error) {
		updated, note, changeErr := change(lines, entry)
		if changeErr != nil {
			return nil, changeErr
		}
		message = note
		changed = !slices.Equal(updated, lines)
		return updated, nil
	}); err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	_, _ = fmt.Fprintln(out, message)
	if changed {
		_, _ = fmt.Fprintf(out, "The daemon reads the file at start. Restart it to apply: %s\n", restartCommand())
	}
	return nil
}

// sameEntry reports whether a file line holds the policy entry names.
func sameEntry(line, entry string) bool {
	policy, err := attachpolicy.Parse(strings.TrimSpace(line))
	return err == nil && attachpolicy.Format(policy) == entry
}

func privilegedAccount() string {
	if runtime.GOOS == "windows" {
		return "an Administrator (an elevated shell)"
	}
	return "root"
}

// rerunCommand is the same command line, ready to paste with the privilege it
// lacked.
func rerunCommand(cmd *cobra.Command, value string) string {
	line := cmd.CommandPath() + " " + value
	if runtime.GOOS == "windows" {
		return "run `" + line + "` from an elevated PowerShell"
	}
	return "sudo " + line
}

// restartCommand restarts the daemon the way the platform's package runs it.
func restartCommand() string {
	switch runtime.GOOS {
	case "windows":
		return "Restart-Service NiACSimulator"
	case "darwin":
		return "sudo launchctl unload /Library/LaunchDaemons/com.niac.plist && " +
			"sudo launchctl load -w /Library/LaunchDaemons/com.niac.plist"
	default:
		return "sudo systemctl restart niac (or restart the foreground process)"
	}
}
