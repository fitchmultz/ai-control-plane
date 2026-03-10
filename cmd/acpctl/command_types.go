// command_types.go - Typed acpctl command model and runtime context.
//
// Purpose:
//
//	Define the shared command metadata, parsed input helpers, and runtime
//	context used by the acpctl command platform.
//
// Responsibilities:
//   - Declare command, option, argument, backend, and invocation types.
//   - Provide typed accessors over parsed flag and argument values.
//   - Carry the runtime logger/output context used by native commands.
//
// Scope:
//   - Shared CLI platform types only.
//
// Usage:
//   - Consumed by the command compiler, parser, dispatcher, help renderer,
//     completions, and native command handlers.
//
// Invariants/Assumptions:
//   - The compiled command tree remains the only source of command metadata.
//   - Native command handlers receive a fully-populated runtime context.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/mitchfultz/ai-control-plane/internal/textutil"
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
	Logger   *slog.Logger
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

func (p parsedCommandInput) NormalizedString(name string) string {
	return textutil.Trim(p.String(name))
}

func (p parsedCommandInput) LowerString(name string) string {
	return textutil.LowerTrim(p.String(name))
}

func (p parsedCommandInput) Has(name string) bool {
	return len(p.flags[name]) > 0
}

func (p parsedCommandInput) StringDefault(name string, fallback string) string {
	return textutil.DefaultIfBlank(p.String(name), fallback)
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

func (p parsedCommandInput) IntDefault(name string, fallback int) (int, error) {
	if textutil.IsBlank(p.String(name)) {
		return fallback, nil
	}
	return p.Int(name)
}

func (p parsedCommandInput) Float(name string) (float64, error) {
	value := firstValue(p.flags[name])
	if value == "" {
		return 0, nil
	}
	return strconv.ParseFloat(value, 64)
}

func (p parsedCommandInput) FloatDefault(name string, fallback float64) (float64, error) {
	if textutil.IsBlank(p.String(name)) {
		return fallback, nil
	}
	return p.Float(name)
}

func (p parsedCommandInput) Argument(index int) string {
	if index < 0 || index >= len(p.arguments) {
		return ""
	}
	return p.arguments[index]
}

func (p parsedCommandInput) NormalizedArgument(index int) string {
	return textutil.Trim(p.Argument(index))
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

func firstValue(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
