// cmd_env.go - Strict environment-file access commands.
//
// Purpose:
//   - Provide typed, non-executing access to repository .env values.
//
// Responsibilities:
//   - Define the typed `env` command spec and option binding.
//   - Read specific keys from env files via the shared strict parser.
//   - Emit deterministic exit codes for missing keys and invalid files.
//
// Non-scope:
//   - Does not export shell snippets or evaluate shell syntax.
//   - Does not mutate env files.
//
// Invariants/Assumptions:
//   - `.env` content is treated as data only.
//   - Command output is the raw requested value on stdout.
//
// Scope:
//   - File-local implementation and interfaces only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/mitchfultz/ai-control-plane/internal/config"
	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	repopath "github.com/mitchfultz/ai-control-plane/internal/paths"
)

type envGetOptions struct {
	File string
	Key  string
}

func envCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "env",
		Summary:     "Strict .env access helpers",
		Description: "Typed .env access helpers that never execute shell code.",
		Examples: []string{
			"acpctl env get LITELLM_MASTER_KEY",
			"acpctl env get --file demo/.env DATABASE_URL",
		},
		Children: []*commandSpec{
			{
				Name:        "get",
				Summary:     "Read a single env key without shell execution",
				Description: "Read a single env key from an env file as data only.",
				Examples: []string{
					"acpctl env get LITELLM_MASTER_KEY",
					"acpctl env get --file demo/.env DATABASE_URL",
					"acpctl env get --file /etc/ai-control-plane/secrets.env LITELLM_MASTER_KEY",
				},
				Options: []commandOptionSpec{
					{
						Name:        "file",
						ValueName:   "PATH",
						Summary:     "Path to env file",
						Type:        optionValueString,
						DefaultText: "demo/.env",
					},
				},
				Arguments: []commandArgumentSpec{
					{Name: "key", Summary: "Env key to read", Required: true},
				},
				Sections: []commandHelpSection{
					{
						Title: "Notes",
						Lines: []string{
							"Env files are treated as data only and are never executed.",
							"Prefer this over sourcing env files or grepping secrets from them.",
						},
					},
				},
				Backend: commandBackend{
					Kind:       commandBackendNative,
					NativeBind: bindRepoParsed(bindEnvGetOptions),
					NativeRun:  runEnvGet,
				},
			},
		},
	}
}

func bindEnvGetOptions(bindCtx commandBindContext, input parsedCommandInput) (envGetOptions, error) {
	key := input.NormalizedArgument(0)
	if key == "" {
		return envGetOptions{}, fmt.Errorf("env key must not be empty")
	}

	repoRoot, err := requireCommandRepoRoot(bindCtx)
	if err != nil {
		return envGetOptions{}, err
	}

	defaultEnvPath := repopath.DemoEnvPath(repoRoot)
	envPath := input.NormalizedString("file")
	if envPath == "" {
		envPath = defaultEnvPath
	} else {
		envPath = resolveRepoInput(repoRoot, envPath)
	}
	return envGetOptions{
		File: envPath,
		Key:  key,
	}, nil
}

func runEnvGet(_ context.Context, runCtx commandRunContext, raw any) int {
	opts := raw.(envGetOptions)
	value, ok, err := config.NewEnvFile(opts.File).Lookup(opts.Key)
	if err != nil {
		fmt.Fprintf(runCtx.Stderr, "Error: failed to read env file: %v\n", err)
		return exitcodes.ACPExitPrereq
	}
	if !ok {
		fmt.Fprintf(runCtx.Stderr, "Error: %s not found in %s\n", opts.Key, opts.File)
		return exitcodes.ACPExitDomain
	}

	fmt.Fprintln(runCtx.Stdout, value)
	return exitcodes.ACPExitSuccess
}

func runEnvCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runCommandPath(ctx, []string{"env"}, args, stdout, stderr)
}

func runEnvGetCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runCommandPath(ctx, []string{"env", "get"}, args, stdout, stderr)
}
