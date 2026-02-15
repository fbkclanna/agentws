package main

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/fbkclanna/agentws/internal/workspace"
	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose environment for common issues",
		RunE:  runDoctor,
	}
}

func runDoctor(cmd *cobra.Command, _ []string) error {
	ok := true

	// Check git.
	fmt.Print("Checking git... ")
	gitPath, err := exec.LookPath("git")
	if err != nil {
		fmt.Println("NOT FOUND")
		fmt.Println("  git is required. Install it from https://git-scm.com/")
		ok = false
	} else {
		fmt.Printf("found at %s\n", gitPath)
	}

	// Check git version.
	if err == nil {
		fmt.Print("Checking git version... ")
		out, verr := exec.Command("git", "version").Output()
		if verr != nil {
			fmt.Println("ERROR")
			ok = false
		} else {
			ver := strings.TrimSpace(string(out))
			fmt.Println(ver)
			if !checkGitFeature("sparse-checkout") {
				fmt.Println("  Warning: git sparse-checkout not supported (need git 2.25+)")
			}
		}
	}

	// Check SSH authentication.
	fmt.Print("Checking SSH auth (github.com)... ")
	if checkSSHAuth("github.com") {
		fmt.Println("OK")
	} else {
		fmt.Println("FAILED (SSH key may not be configured)")
		ok = false
	}

	// Check workspace manifest if in a workspace dir.
	root, _ := cmd.Flags().GetString("root")
	ctx, loadErr := workspace.Load(root)
	if loadErr == nil {
		fmt.Printf("Workspace: %s (%d repos)\n", ctx.Manifest.Name, len(ctx.Manifest.Repos))
		checkRepoURLs(ctx)
	} else {
		fmt.Println("No workspace found in current directory (skipping repo checks)")
	}

	if ok {
		fmt.Println("\nAll checks passed.")
		return nil
	}
	fmt.Println("\nSome checks failed. See above for details.")
	return fmt.Errorf("doctor checks failed")
}

func checkGitFeature(feature string) bool {
	err := exec.Command("git", feature, "--help").Run()
	return err == nil
}

// checkSSHAuth tests SSH connectivity to a host (e.g., github.com).
// GitHub returns exit code 1 even on success (with a welcome message), so
// we check stderr for "successfully authenticated".
func checkSSHAuth(host string) bool {
	out, _ := exec.Command("ssh", "-T", "-o", "StrictHostKeyChecking=accept-new", //nolint:gosec // host is derived from workspace manifest, not arbitrary user input
		"-o", "ConnectTimeout=5", "git@"+host).CombinedOutput()
	result := strings.ToLower(string(out))
	return strings.Contains(result, "successfully authenticated") ||
		strings.Contains(result, "you've successfully authenticated") ||
		strings.Contains(result, "hi ")
}

// checkRepoURLs tests connectivity to each repo URL listed in the manifest.
func checkRepoURLs(ctx *workspace.Context) {
	for _, r := range ctx.Manifest.Repos {
		fmt.Printf("  Checking %s (%s)... ", r.ID, r.URL)
		if checkGitLsRemote(r.URL) {
			fmt.Println("OK")
		} else {
			fmt.Println("FAILED (cannot access)")
		}
	}
}

// checkGitLsRemote verifies that a repo URL is reachable.
func checkGitLsRemote(url string) bool {
	cmd := exec.Command("git", "ls-remote", "--exit-code", "--quiet", url)
	return cmd.Run() == nil
}
