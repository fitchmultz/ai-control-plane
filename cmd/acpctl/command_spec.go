// command_spec.go - Typed acpctl command specification and execution engine.
//
// Purpose:
//
//	Provide the single typed source of truth for acpctl command metadata,
//	parsing, help rendering, completion metadata, and execution dispatch.
//
// Responsibilities:
//   - Define reusable command, flag, argument, and backend types.
//   - Parse argv into typed command invocations with validated options.
//   - Render help and completion views from the same normalized command tree.
//   - Dispatch native, Make-backed, and bridge-backed commands consistently.
//
// Scope:
//   - CLI platform behavior only; business workflows stay in command handlers.
//
// Usage:
//   - Consumed by main.go, command specs, completion generation, and tests.
//
// Invariants/Assumptions:
//   - The compiled spec tree is the only command metadata source.
//   - Every leaf command resolves to exactly one execution backend.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
)

type commandBackendKind string

const (
	commandBackendNative commandBackendKind = "native"
	commandBackendMake   commandBackendKind = "make"
	commandBackendBridge commandBackendKind = "bridge"
)

type optionValueType string

const (
	optionValueBool   optionValueType = "bool"
	optionValueString optionValueType = "string"
	optionValueInt    optionValueType = "int"
	optionValueFloat  optionValueType = "float"
)

type completionValuesProvider func(repoRoot string) []string

type commandOptionSpec struct {
	Name         string
	Short        string
	ValueName    string
	Summary      string
	Type         optionValueType
	DefaultText  string
	Repeatable   bool
	Required     bool
	Suggestions  completionValuesProvider
	SuggestAsKey string
}

type commandArgumentSpec struct {
	Name         string
	Summary      string
	Required     bool
	Repeatable   bool
	Suggestions  completionValuesProvider
	SuggestAsKey string
}

type commandHelpSection struct {
	Title string
	Lines []string
}

type commandBackend struct {
	Kind               commandBackendKind
	NativeBind         func(commandBindContext, parsedCommandInput) (any, error)
	NativeRun          func(context.Context, commandRunContext, any) int
	MakeTarget         string
	BridgeRelativePath string
	BridgeArgs         []string
}

type commandSpec struct {
	Name              string
	Summary           string
	Description       string
	Examples          []string
	Hidden            bool
	Options           []commandOptionSpec
	Arguments         []commandArgumentSpec
	Sections          []commandHelpSection
	Children          []*commandSpec
	Backend           commandBackend
	AllowTrailingArgs bool
}

type commandRunContext struct {
	RepoRoot string
	Stdout   *os.File
	Stderr   *os.File
}

type commandBindContext struct {
	RepoRoot string
}

type parsedCommandInput struct {
	flags     map[string][]string
	arguments []string
	trailing  []string
}

func (p parsedCommandInput) Bool(name string) bool {
	return firstValue(p.flags[name]) == "true"
}

func (p parsedCommandInput) String(name string) string {
	return firstValue(p.flags[name])
}

func (p parsedCommandInput) Strings(name string) []string {
	values := p.flags[name]
	if len(values) == 0 {
		return nil
	}
	cloned := make([]string, len(values))
	copy(cloned, values)
	return cloned
}

func (p parsedCommandInput) Int(name string) (int, error) {
	value := firstValue(p.flags[name])
	if value == "" {
		return 0, nil
	}
	return strconv.Atoi(value)
}

func (p parsedCommandInput) Float(name string) (float64, error) {
	value := firstValue(p.flags[name])
	if value == "" {
		return 0, nil
	}
	return strconv.ParseFloat(value, 64)
}

func (p parsedCommandInput) Argument(index int) string {
	if index < 0 || index >= len(p.arguments) {
		return ""
	}
	return p.arguments[index]
}

func (p parsedCommandInput) Arguments() []string {
	values := make([]string, len(p.arguments))
	copy(values, p.arguments)
	return values
}

func (p parsedCommandInput) Trailing() []string {
	values := make([]string, len(p.trailing))
	copy(values, p.trailing)
	return values
}

type commandInvocation struct {
	Spec     *commandSpec
	Path     []*commandSpec
	Input    parsedCommandInput
	HelpOnly bool
}

type compiledCommandSpec struct {
	Root             *commandSpec
	VisibleRoots     []*commandSpec
	VisibleRootNames []string
	NodesByPath      map[string]*commandSpec
}

type commandLookupError struct {
	Kind string
	Path string
	Name string
}

func (e *commandLookupError) Error() string {
	switch e.Kind {
	case "root":
		return fmt.Sprintf("unknown root command: %s", e.Name)
	case "subcommand":
		return fmt.Sprintf("unknown subcommand %s for %s", e.Name, e.Path)
	default:
		return "invalid command lookup"
	}
}

var (
	commandSpecOnce sync.Once
	commandSpecData compiledCommandSpec
	commandSpecErr  error
)

func loadCommandSpec() (compiledCommandSpec, error) {
	commandSpecOnce.Do(func() {
		commandSpecData, commandSpecErr = compileCommandSpec(acpctlCommandSpec())
	})
	return commandSpecData, commandSpecErr
}

func commandStartupError() error {
	_, err := loadCommandSpec()
	return err
}

func rootCommandSpec(children ...*commandSpec) *commandSpec {
	return &commandSpec{
		Name:        "acpctl",
		Summary:     "Typed control-plane CLI for AI Control Plane operations.",
		Description: "Typed control-plane CLI for AI Control Plane operations.",
		Examples: []string{
			"acpctl ci should-run-runtime --quiet",
			"acpctl ci wait --timeout 120",
			"acpctl env get LITELLM_MASTER_KEY",
			"acpctl chargeback report --format all",
			"acpctl benchmark baseline --requests 20 --concurrency 2",
			"acpctl onboard codex --mode subscription --verify",
			"acpctl deploy up",
			"acpctl deploy readiness-evidence run",
		},
		Sections: []commandHelpSection{
			{
				Title: "Environment",
				Lines: []string{
					"ACPCTL_MAKE_BIN  Override make executable used by delegated commands (default: make)",
					"ACP_REPO_ROOT    Override repository root detection",
				},
			},
		},
		Children: children,
	}
}

func compileCommandSpec(root *commandSpec) (compiledCommandSpec, error) {
	if root == nil {
		return compiledCommandSpec{}, errors.New("command spec root is nil")
	}
	compiled := compiledCommandSpec{
		Root:        root,
		NodesByPath: make(map[string]*commandSpec),
	}
	if err := compileCommandNode(root, nil, compiled.NodesByPath, &compiled.VisibleRoots, &compiled.VisibleRootNames); err != nil {
		return compiledCommandSpec{}, err
	}
	return compiled, nil
}

func compileCommandNode(node *commandSpec, ancestors []*commandSpec, index map[string]*commandSpec, visibleRoots *[]*commandSpec, visibleRootNames *[]string) error {
	if strings.TrimSpace(node.Name) == "" {
		return errors.New("command spec name must not be empty")
	}
	path := append(append([]*commandSpec(nil), ancestors...), node)
	pathKey := commandPathKey(path[1:])
	if _, exists := index[pathKey]; exists {
		return fmt.Errorf("duplicate command path: %s", pathKey)
	}
	index[pathKey] = node
	if len(ancestors) == 1 && !node.Hidden {
		*visibleRoots = append(*visibleRoots, node)
		*visibleRootNames = append(*visibleRootNames, node.Name)
	}
	if len(node.Children) > 0 {
		seen := make(map[string]struct{}, len(node.Children))
		for _, child := range node.Children {
			if _, exists := seen[child.Name]; exists {
				return fmt.Errorf("duplicate child command under %s: %s", commandPathKey(path[1:]), child.Name)
			}
			seen[child.Name] = struct{}{}
			if err := compileCommandNode(child, path, index, visibleRoots, visibleRootNames); err != nil {
				return err
			}
		}
		if node.Backend.Kind != "" {
			return fmt.Errorf("group command cannot also own an execution backend: %s", commandPathKey(path[1:]))
		}
		return nil
	}
	switch node.Backend.Kind {
	case commandBackendNative:
		if node.Backend.NativeBind == nil || node.Backend.NativeRun == nil {
			return fmt.Errorf("native command missing binder or runner: %s", commandPathKey(path[1:]))
		}
	case commandBackendMake:
		if strings.TrimSpace(node.Backend.MakeTarget) == "" {
			return fmt.Errorf("make command missing target: %s", commandPathKey(path[1:]))
		}
	case commandBackendBridge:
		if strings.TrimSpace(node.Backend.BridgeRelativePath) == "" {
			return fmt.Errorf("bridge command missing script path: %s", commandPathKey(path[1:]))
		}
	default:
		return fmt.Errorf("leaf command missing backend: %s", commandPathKey(path[1:]))
	}
	return nil
}

func commandPathKey(path []*commandSpec) string {
	names := make([]string, 0, len(path))
	for _, node := range path {
		if node.Name == "acpctl" {
			continue
		}
		names = append(names, node.Name)
	}
	return strings.Join(names, " ")
}

func commandPathLabel(path []*commandSpec) string {
	if len(path) == 0 {
		return "acpctl"
	}
	names := []string{"acpctl"}
	for _, node := range path {
		if node.Name == "acpctl" {
			continue
		}
		names = append(names, node.Name)
	}
	return strings.Join(names, " ")
}

func findCommand(path []string) (*commandSpec, error) {
	spec, err := loadCommandSpec()
	if err != nil {
		return nil, err
	}
	current := spec.Root
	for index, part := range path {
		next := findChildCommand(current, part)
		if next == nil {
			if index == 0 {
				return nil, &commandLookupError{Kind: "root", Name: part}
			}
			return nil, &commandLookupError{Kind: "subcommand", Path: commandPathKey(pathToSpecs(path[:index])), Name: part}
		}
		current = next
	}
	return current, nil
}

func pathToSpecs(path []string) []*commandSpec {
	nodes := make([]*commandSpec, 0, len(path))
	for _, name := range path {
		nodes = append(nodes, &commandSpec{Name: name})
	}
	return nodes
}

func findChildCommand(node *commandSpec, name string) *commandSpec {
	for _, child := range node.Children {
		if child.Name == name {
			return child
		}
	}
	return nil
}

func parseInvocation(args []string) (commandInvocation, error) {
	spec, err := loadCommandSpec()
	if err != nil {
		return commandInvocation{}, err
	}
	path := []*commandSpec{spec.Root}
	current := spec.Root
	remaining := args
	for len(remaining) > 0 {
		token := remaining[0]
		if isHelpToken(token) {
			return commandInvocation{Spec: current, Path: path, HelpOnly: true}, nil
		}
		next := findChildCommand(current, token)
		if next == nil {
			break
		}
		current = next
		path = append(path, current)
		remaining = remaining[1:]
	}
	if current == spec.Root {
		if len(remaining) == 0 {
			return commandInvocation{Spec: spec.Root, Path: path, HelpOnly: true}, nil
		}
		return commandInvocation{}, &commandLookupError{Kind: "root", Name: remaining[0]}
	}
	if len(current.Children) > 0 {
		if len(remaining) == 0 {
			return commandInvocation{Spec: current, Path: path, HelpOnly: true}, nil
		}
		if isHelpToken(remaining[0]) {
			return commandInvocation{Spec: current, Path: path, HelpOnly: true}, nil
		}
		return commandInvocation{}, &commandLookupError{Kind: "subcommand", Path: commandPathKey(path[1:]), Name: remaining[0]}
	}
	input, helpOnly, err := parseLeafInput(current, remaining)
	if err != nil {
		return commandInvocation{}, err
	}
	return commandInvocation{
		Spec:     current,
		Path:     path,
		Input:    input,
		HelpOnly: helpOnly,
	}, nil
}

func parseLeafInput(spec *commandSpec, args []string) (parsedCommandInput, bool, error) {
	input := parsedCommandInput{flags: make(map[string][]string)}
	optionByLong := make(map[string]commandOptionSpec, len(spec.Options))
	optionByShort := make(map[string]commandOptionSpec, len(spec.Options))
	for _, option := range spec.Options {
		optionByLong[option.Name] = option
		if option.Short != "" {
			optionByShort[option.Short] = option
		}
	}

	argumentIndex := 0
	for index := 0; index < len(args); index++ {
		token := args[index]
		if isHelpToken(token) {
			return parsedCommandInput{}, true, nil
		}
		if spec.AllowTrailingArgs && token == "--" {
			input.trailing = append(input.trailing, args[index+1:]...)
			break
		}
		if strings.HasPrefix(token, "--") {
			name, explicitValue, hasExplicitValue := strings.Cut(strings.TrimPrefix(token, "--"), "=")
			option, ok := optionByLong[name]
			if !ok {
				if spec.AllowTrailingArgs {
					input.trailing = append(input.trailing, args[index:]...)
					break
				}
				return parsedCommandInput{}, false, fmt.Errorf("unknown option: --%s", name)
			}
			value, consumed, err := parseOptionValue(option, token, explicitValue, hasExplicitValue, args, index)
			if err != nil {
				return parsedCommandInput{}, false, err
			}
			input.flags[option.Name] = append(input.flags[option.Name], value)
			index += consumed
			continue
		}
		if strings.HasPrefix(token, "-") && token != "-" {
			shortName, explicitValue, hasExplicitValue := strings.Cut(strings.TrimPrefix(token, "-"), "=")
			option, ok := optionByShort[shortName]
			if !ok {
				if spec.AllowTrailingArgs {
					input.trailing = append(input.trailing, args[index:]...)
					break
				}
				return parsedCommandInput{}, false, fmt.Errorf("unknown option: -%s", shortName)
			}
			value, consumed, err := parseOptionValue(option, token, explicitValue, hasExplicitValue, args, index)
			if err != nil {
				return parsedCommandInput{}, false, err
			}
			input.flags[option.Name] = append(input.flags[option.Name], value)
			index += consumed
			continue
		}
		if argumentIndex < len(spec.Arguments) {
			input.arguments = append(input.arguments, token)
			if !spec.Arguments[argumentIndex].Repeatable {
				argumentIndex++
			}
			continue
		}
		if spec.AllowTrailingArgs {
			input.trailing = append(input.trailing, args[index:]...)
			break
		}
		return parsedCommandInput{}, false, fmt.Errorf("unexpected argument: %s", token)
	}

	for _, option := range spec.Options {
		if option.Required && len(input.flags[option.Name]) == 0 {
			return parsedCommandInput{}, false, fmt.Errorf("missing required option: --%s", option.Name)
		}
		if !option.Repeatable && len(input.flags[option.Name]) > 1 {
			return parsedCommandInput{}, false, fmt.Errorf("option may only be provided once: --%s", option.Name)
		}
	}
	requiredArgs := 0
	for _, argument := range spec.Arguments {
		if argument.Required {
			requiredArgs++
		}
	}
	if len(input.arguments) < requiredArgs {
		missing := spec.Arguments[len(input.arguments)].Name
		return parsedCommandInput{}, false, fmt.Errorf("missing required argument: %s", missing)
	}
	return input, false, nil
}

func parseOptionValue(option commandOptionSpec, token string, explicitValue string, hasExplicitValue bool, args []string, index int) (string, int, error) {
	switch option.Type {
	case optionValueBool:
		if hasExplicitValue {
			value := strings.ToLower(strings.TrimSpace(explicitValue))
			switch value {
			case "true", "1", "yes", "on":
				return "true", 0, nil
			case "false", "0", "no", "off":
				return "false", 0, nil
			default:
				return "", 0, fmt.Errorf("invalid boolean value for %s: %s", token, explicitValue)
			}
		}
		return "true", 0, nil
	default:
		if hasExplicitValue {
			return explicitValue, 0, nil
		}
		if index+1 >= len(args) {
			return "", 0, fmt.Errorf("missing value for %s", token)
		}
		return args[index+1], 1, nil
	}
}

func executeInvocation(ctx context.Context, invocation commandInvocation, stdout *os.File, stderr *os.File) int {
	if invocation.HelpOnly {
		printCommandHelp(stdout, invocation.Path)
		if len(invocation.Path) == 1 || len(invocation.Spec.Children) > 0 {
			return exitcodes.ACPExitSuccess
		}
		return exitcodes.ACPExitSuccess
	}
	runCtx := commandRunContext{
		RepoRoot: detectRepoRootWithContext(ctx),
		Stdout:   stdout,
		Stderr:   stderr,
	}
	switch invocation.Spec.Backend.Kind {
	case commandBackendNative:
		opts, err := invocation.Spec.Backend.NativeBind(commandBindContext{RepoRoot: runCtx.RepoRoot}, invocation.Input)
		if err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			printCommandHelp(stderr, invocation.Path)
			return exitcodes.ACPExitUsage
		}
		return invocation.Spec.Backend.NativeRun(ctx, runCtx, opts)
	case commandBackendMake:
		return runMakeTarget(ctx, invocation.Spec.Backend.MakeTarget, invocation.Input.Trailing(), stdout, stderr)
	case commandBackendBridge:
		return runBridgeScript(
			ctx,
			invocation.Spec.Backend.BridgeRelativePath,
			invocation.Spec.Name,
			invocation.Spec.Backend.BridgeArgs,
			invocation.Input.Trailing(),
			stdout,
			stderr,
		)
	default:
		fmt.Fprintln(stderr, "Error: invalid command backend")
		return exitcodes.ACPExitRuntime
	}
}

func renderUsage(path []*commandSpec) string {
	current := path[len(path)-1]
	usage := commandPathLabel(path[1:])
	if len(current.Children) > 0 {
		return usage + " <subcommand>"
	}
	if len(current.Options) > 0 {
		usage += " [options]"
	}
	for _, argument := range current.Arguments {
		if argument.Required {
			usage += " <" + argument.Name + ">"
		} else {
			usage += " [" + argument.Name + "]"
		}
		if argument.Repeatable {
			usage += "..."
		}
	}
	if current.AllowTrailingArgs {
		usage += " [args...]"
	}
	return usage
}

func printCommandHelp(out *os.File, path []*commandSpec) {
	current := path[len(path)-1]
	fmt.Fprintf(out, "Usage: %s\n\n", renderUsage(path))
	description := strings.TrimSpace(current.Description)
	if description == "" {
		description = current.Summary
	}
	if description != "" {
		fmt.Fprintln(out, description)
		fmt.Fprintln(out)
	}
	if len(current.Children) > 0 {
		fmt.Fprintln(out, "Commands:")
		for _, child := range current.Children {
			if child.Hidden {
				continue
			}
			fmt.Fprintf(out, "  %-22s %s\n", child.Name, child.Summary)
		}
	} else {
		if len(current.Arguments) > 0 {
			fmt.Fprintln(out, "Arguments:")
			for _, argument := range current.Arguments {
				fmt.Fprintf(out, "  %-22s %s\n", argument.Name, argument.Summary)
			}
			fmt.Fprintln(out)
		}
		if len(current.Options) > 0 {
			fmt.Fprintln(out, "Options:")
			for _, option := range current.Options {
				labels := []string{"--" + option.Name}
				if option.Short != "" {
					labels = append(labels, "-"+option.Short)
				}
				label := strings.Join(labels, ", ")
				if option.Type != optionValueBool {
					valueName := option.ValueName
					if valueName == "" {
						valueName = strings.ToUpper(option.Name)
					}
					label += " " + valueName
				}
				summary := option.Summary
				if option.DefaultText != "" {
					summary += " (default: " + option.DefaultText + ")"
				}
				fmt.Fprintf(out, "  %-22s %s\n", label, summary)
			}
			fmt.Fprintln(out)
		}
	}
	for _, section := range current.Sections {
		fmt.Fprintf(out, "%s:\n", section.Title)
		for _, line := range section.Lines {
			fmt.Fprintf(out, "  %s\n", line)
		}
		fmt.Fprintln(out)
	}
	if len(current.Examples) > 0 {
		fmt.Fprintln(out, "Examples:")
		for _, example := range current.Examples {
			fmt.Fprintf(out, "  %s\n", example)
		}
		fmt.Fprintln(out)
	}
	fmt.Fprintln(out, "Exit codes:")
	fmt.Fprintln(out, "  0   Success")
	fmt.Fprintln(out, "  1   Domain non-success")
	fmt.Fprintln(out, "  2   Prerequisites not ready")
	fmt.Fprintln(out, "  3   Runtime/internal error")
	fmt.Fprintln(out, "  64  Usage error")
}

type commandRegistry struct {
	RootCommands     []commandDescriptor
	GroupSubcommands map[string][]commandDescriptor
}

type commandDescriptor struct {
	Name        string
	Description string
}

func buildCommandRegistry() commandRegistry {
	spec, err := loadCommandSpec()
	if err != nil {
		return commandRegistry{}
	}
	registry := commandRegistry{
		RootCommands:     make([]commandDescriptor, 0, len(spec.VisibleRoots)+1),
		GroupSubcommands: make(map[string][]commandDescriptor, len(spec.VisibleRoots)),
	}
	for _, root := range spec.VisibleRoots {
		registry.RootCommands = append(registry.RootCommands, commandDescriptor{Name: root.Name, Description: root.Summary})
		if len(root.Children) == 0 {
			continue
		}
		subcommands := make([]commandDescriptor, 0, len(root.Children))
		for _, child := range root.Children {
			if child.Hidden {
				continue
			}
			subcommands = append(subcommands, commandDescriptor{Name: child.Name, Description: child.Summary})
		}
		if len(subcommands) > 0 {
			registry.GroupSubcommands[root.Name] = subcommands
		}
	}
	registry.RootCommands = append(registry.RootCommands, commandDescriptor{Name: "help", Description: "Show this help message"})
	return registry
}

type commandCompletionCatalog struct {
	RootCommands     []string
	GroupSubcommands map[string][]string
}

func buildCompletionCatalog() commandCompletionCatalog {
	registry := buildCommandRegistry()
	catalog := commandCompletionCatalog{
		RootCommands:     make([]string, 0, len(registry.RootCommands)),
		GroupSubcommands: make(map[string][]string, len(registry.GroupSubcommands)),
	}
	for _, root := range registry.RootCommands {
		catalog.RootCommands = append(catalog.RootCommands, root.Name)
	}
	for name, subcommands := range registry.GroupSubcommands {
		values := make([]string, 0, len(subcommands))
		for _, subcommand := range subcommands {
			values = append(values, subcommand.Name)
		}
		catalog.GroupSubcommands[name] = values
	}
	return catalog
}

func resolveSuggestions(words []string, prefix string, repoRoot string) []string {
	spec, err := loadCommandSpec()
	if err != nil {
		return nil
	}
	if len(words) == 0 {
		return append([]string(nil), spec.VisibleRootNames...)
	}
	current := spec.Root
	for _, word := range words {
		next := findChildCommand(current, word)
		if next == nil {
			break
		}
		current = next
	}
	if strings.Contains(prefix, "=") {
		key, value, _ := strings.Cut(prefix, "=")
		values := current.suggestValues(repoRoot, key)
		if len(values) == 0 {
			return nil
		}
		suggestions := make([]string, 0, len(values))
		for _, candidate := range values {
			if strings.HasPrefix(candidate, value) {
				suggestions = append(suggestions, key+"="+candidate)
			}
		}
		return dedupeAndSort(suggestions)
	}
	if current != spec.Root && len(current.Children) > 0 {
		names := make([]string, 0, len(current.Children))
		for _, child := range current.Children {
			if child.Hidden {
				continue
			}
			names = append(names, child.Name)
		}
		return dedupeAndSort(names)
	}
	return append([]string(nil), spec.VisibleRootNames...)
}

func (spec *commandSpec) suggestValues(repoRoot string, key string) []string {
	values := make([]string, 0)
	for _, argument := range spec.Arguments {
		if argument.SuggestAsKey == key && argument.Suggestions != nil {
			values = append(values, argument.Suggestions(repoRoot)...)
		}
	}
	for _, option := range spec.Options {
		if option.SuggestAsKey == key && option.Suggestions != nil {
			values = append(values, option.Suggestions(repoRoot)...)
		}
	}
	return dedupeAndSort(values)
}

func dedupeAndSort(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	deduped := make([]string, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		deduped = append(deduped, value)
	}
	sort.Strings(deduped)
	return deduped
}

func firstValue(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
