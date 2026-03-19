// cmd_policy.go - Custom policy engine command surface.
//
// Purpose:
//   - Own the typed ACP-native custom policy evaluation workflow.
//
// Responsibilities:
//   - Define the `policy eval` command tree.
//   - Read request/response policy input from file or stdin.
//   - Delegate rule evaluation and artifact generation to internal/policyengine.
//
// Scope:
//   - Policy-evaluation command bindings and operator output only.
//
// Usage:
//   - `acpctl policy eval --file examples/policy-engine/request_response_eval.sample.json`
//   - `cat sample.json | acpctl policy eval`
//
// Invariants/Assumptions:
//   - Evaluation is a local host-first workflow, not a live inline proxy mode.
package main

import (
	"context"
	"os"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/output"
	repopath "github.com/mitchfultz/ai-control-plane/internal/paths"
	"github.com/mitchfultz/ai-control-plane/internal/policyengine"
)

type policyEvalOptions struct {
	RepoRoot   string
	RulesPath  string
	OutputRoot string
	InputPath  string
}

func policyCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "policy",
		Summary:     "ACP-native custom policy evaluation workflows",
		Description: "ACP-native custom policy evaluation workflows.",
		Examples: []string{
			"acpctl policy eval --file examples/policy-engine/request_response_eval.sample.json",
			"cat request_response_eval.sample.json | acpctl policy eval",
		},
		Children: []*commandSpec{
			newNativeCommandSpec(nativeCommandConfig{
				Name:        "eval",
				Summary:     "Evaluate local request/response records against custom ACP guardrails",
				Description: "Evaluate local request/response records against custom ACP guardrails.",
				Options: []commandOptionSpec{
					{Name: "file", ValueName: "PATH", Summary: "Read input from a JSON file instead of stdin", Type: optionValueString},
					{Name: "rules-file", ValueName: "PATH", Summary: "Policy rule file to evaluate with", Type: optionValueString, DefaultText: policyengine.DefaultRulesPath},
					{Name: "output-dir", ValueName: "DIR", Summary: "Output root for evaluation runs", Type: optionValueString, DefaultText: "demo/logs/evidence/policy-eval"},
				},
				Bind: bindRepoParsed(bindPolicyEvalOptions),
				Run:  runPolicyEvalTyped,
			}),
		},
	}
}

func bindPolicyEvalOptions(bindCtx commandBindContext, input parsedCommandInput) (policyEvalOptions, error) {
	repoRoot, err := requireCommandRepoRoot(bindCtx)
	if err != nil {
		return policyEvalOptions{}, err
	}
	options := policyEvalOptions{
		RepoRoot:   repoRoot,
		RulesPath:  repopath.ResolveRepoPath(repoRoot, policyengine.DefaultRulesPath),
		OutputRoot: repopath.DemoLogsPath(repoRoot, "evidence", policyengine.DefaultOutputSubdir),
	}
	if input.Has("file") {
		options.InputPath = resolveRepoInput(repoRoot, input.NormalizedString("file"))
	}
	if input.Has("rules-file") {
		options.RulesPath = resolveRepoInput(repoRoot, input.NormalizedString("rules-file"))
	}
	if input.Has("output-dir") {
		options.OutputRoot = resolveRepoInput(repoRoot, input.NormalizedString("output-dir"))
	}
	return options, nil
}

func runPolicyEvalTyped(ctx context.Context, runCtx commandRunContext, raw any) int {
	out := output.New()
	opts := raw.(policyEvalOptions)
	payload, inputLabel, inputCode, err := loadJSONPayload(opts.InputPath)
	if err != nil {
		return failCommand(runCtx.Stderr, out, inputCode, err, "policy eval input error")
	}

	printCommandSection(runCtx.Stdout, out, "Evaluating custom policy rules")
	result, err := policyengine.Evaluate(ctx, policyengine.Options{
		RepoRoot:     opts.RepoRoot,
		RulesPath:    opts.RulesPath,
		OutputRoot:   opts.OutputRoot,
		InputPath:    inputLabel,
		InputPayload: payload,
	})
	if err != nil {
		return failCommand(runCtx.Stderr, out, exitcodes.ACPExitRuntime, err, "policy eval failed")
	}

	printCommandSuccess(runCtx.Stdout, out, "Custom policy evaluation complete")
	printCommandDetail(runCtx.Stdout, "Run directory", result.Summary.RunDirectory)
	printCommandDetail(runCtx.Stdout, "Rules file", result.Summary.RulesPath)
	printCommandDetail(runCtx.Stdout, "Input", inputLabel)
	printCommandDetail(runCtx.Stdout, "Records", result.Summary.RecordCount)
	printCommandDetail(runCtx.Stdout, "Decisions", result.Summary.DecisionCount)
	printCommandDetail(runCtx.Stdout, "Evaluated", result.Summary.EvaluatedPath)
	printCommandDetail(runCtx.Stdout, "Normalized", result.Summary.NormalizedPath)
	printCommandDetail(runCtx.Stdout, "Summary", result.Summary.RunDirectory+string(os.PathSeparator)+policyengine.SummaryMarkdownName)
	printCommandDetail(runCtx.Stdout, "Issues", result.Summary.ValidationIssueCount)
	if len(result.Issues) > 0 {
		return failValidation(runCtx.Stderr, out, result.Issues, "Custom policy evaluation found validation issues")
	}
	return exitcodes.ACPExitSuccess
}

func runPolicyCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runCommandGroupPath(ctx, []string{"policy"}, args, stdout, stderr)
}
