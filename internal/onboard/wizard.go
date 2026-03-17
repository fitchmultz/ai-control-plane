// Package onboard implements tool onboarding workflows.
//
// Purpose:
//
//	Run the guided onboarding wizard for supported local tools.
//
// Responsibilities:
//   - Prompt for tool, mode, host, port, TLS, budget, model, verification, and
//     tool-specific choices.
//   - Apply safe defaults at the prompt layer.
//   - Keep interactive I/O separate from workflow execution.
//
// Scope:
//   - Interactive prompt collection only.
//
// Usage:
//   - Called by Run when Options.Stdin is provided by the CLI layer.
//
// Invariants/Assumptions:
//   - Pressing Enter accepts defaults where defaults exist.
//   - Tool selection is required unless a valid tool was preselected.
package onboard

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/config"
)

var errWizardToolRequired = errors.New("tool selection is required; rerun in a terminal or pass a tool name like `acpctl onboard codex`")

type choice struct {
	Value   string
	Label   string
	Summary string
}

type wizard struct {
	in  *bufio.Reader
	out io.Writer
}

func defaultGatewayPort() string {
	return strconv.Itoa(config.DefaultLiteLLMPort)
}

func promptForWizardOptions(opts Options) (Options, error) {
	w := wizard{
		in:  bufio.NewReader(opts.Stdin),
		out: opts.Stdout,
	}

	fprintf(w.out, "ACP onboarding wizard\n\n")
	fprintf(w.out, "This wizard configures a supported tool to use the AI Control Plane gateway.\n")
	fprintf(w.out, "Press Enter to accept defaults shown in [brackets].\n")
	fprintf(w.out, "Gateway contract: local default is http://127.0.0.1:4000. For remote access, enter the routable host or domain. If that endpoint is fronted by TLS, answer yes for HTTPS/TLS and use port 443 or set GATEWAY_URL directly in your shell before launching your tool.\n\n")

	var err error
	if strings.TrimSpace(opts.Tool) == "" {
		opts.Tool, err = w.selectTool()
		if err != nil {
			return Options{}, err
		}
	} else if _, err := lookupToolSpec(opts.Tool); err != nil {
		return Options{}, err
	}

	spec, _ := lookupToolSpec(opts.Tool)

	if strings.TrimSpace(opts.Mode) == "" {
		opts.Mode, err = w.selectMode(spec)
		if err != nil {
			return Options{}, err
		}
	}
	if err := validateMode(spec, opts.Mode); err != nil {
		return Options{}, err
	}

	if opts.Host, err = w.askString("Gateway host", firstNonBlank(opts.Host, config.DefaultGatewayHost), validateNonBlank); err != nil {
		return Options{}, err
	}
	if opts.Port, err = w.askString("Gateway port", firstNonBlank(opts.Port, defaultGatewayPort()), validatePort); err != nil {
		return Options{}, err
	}
	tlsDefault := opts.UseTLS
	if !tlsDefault && opts.Host != config.DefaultGatewayHost {
		tlsDefault = true
	}
	if !tlsDefault && opts.Port == "443" {
		tlsDefault = true
	}
	if opts.UseTLS, err = w.askBool("Use HTTPS/TLS for the gateway URL?", tlsDefault); err != nil {
		return Options{}, err
	}

	if opts.Mode != "direct" {
		if opts.Budget, err = w.askString("Virtual key budget in USD", firstNonBlank(opts.Budget, DefaultBudget), validateBudget); err != nil {
			return Options{}, err
		}
	}
	if opts.Mode != "direct" {
		if opts.Model, err = w.askString("Model alias", firstNonBlank(opts.Model, spec.DefaultModel(opts.Mode)), validateNonBlank); err != nil {
			return Options{}, err
		}
	}
	if opts.Verify, err = w.askBool("Run verification after setup?", true); err != nil {
		return Options{}, err
	}
	if opts.Tool == "codex" && opts.Mode != "direct" {
		if opts.WriteConfig, err = w.askBool("Write ACP-managed ~/.codex/config.toml if the file is safe to manage?", true); err != nil {
			return Options{}, err
		}
	}
	if opts.Mode != "direct" {
		if opts.ShowKey, err = w.askBool("After the redacted summary, reveal the full generated key once?", false); err != nil {
			return Options{}, err
		}
	}

	fprintf(w.out, "\n")
	return opts, nil
}

func (w wizard) selectTool() (string, error) {
	choices := make([]choice, 0, len(onboardingToolOrder))
	for _, spec := range orderedToolSpecs() {
		choices = append(choices, choice{Value: spec.Name, Label: spec.Name, Summary: spec.Summary})
	}
	return w.selectRequired("Select a tool", choices, "")
}

func (w wizard) selectMode(spec toolSpec) (string, error) {
	if len(spec.Modes) == 1 {
		mode := spec.Modes[0]
		fprintf(w.out, "Mode: %s (%s)\n\n", mode.Name, mode.Summary)
		return mode.Name, nil
	}
	choices := make([]choice, 0, len(spec.Modes))
	for _, mode := range spec.Modes {
		choices = append(choices, choice{Value: mode.Name, Label: mode.Name, Summary: mode.Summary})
	}
	return w.selectRequired("Select a mode", choices, spec.DefaultMode)
}

func (w wizard) selectRequired(label string, choices []choice, defaultValue string) (string, error) {
	fprintf(w.out, "%s:\n", label)
	for index, item := range choices {
		suffix := ""
		if defaultValue != "" && item.Value == defaultValue {
			suffix = " [default]"
		}
		fprintf(w.out, "  %d) %s — %s%s\n", index+1, item.Label, item.Summary, suffix)
	}
	for {
		prompt := "> "
		if defaultValue != "" {
			prompt = fmt.Sprintf("> [%s] ", defaultValue)
		}
		fprintf(w.out, "%s", prompt)

		line, err := w.readLine()
		if err != nil {
			if errors.Is(err, io.EOF) && defaultValue != "" {
				return defaultValue, nil
			}
			if errors.Is(err, io.EOF) {
				return "", errWizardToolRequired
			}
			return "", err
		}

		answer := strings.TrimSpace(line)
		if answer == "" && defaultValue != "" {
			return defaultValue, nil
		}
		if idx, err := strconv.Atoi(answer); err == nil && idx >= 1 && idx <= len(choices) {
			return choices[idx-1].Value, nil
		}
		answer = strings.ToLower(answer)
		for _, item := range choices {
			if answer == item.Value {
				return item.Value, nil
			}
		}
		fprintf(w.out, "Please enter a listed number or name.\n")
	}
}

func (w wizard) askString(label string, defaultValue string, validate func(string) error) (string, error) {
	for {
		if defaultValue == "" {
			fprintf(w.out, "%s: ", label)
		} else {
			fprintf(w.out, "%s [%s]: ", label, defaultValue)
		}
		line, err := w.readLine()
		if err != nil {
			if errors.Is(err, io.EOF) {
				line = defaultValue
			} else {
				return "", err
			}
		}
		value := strings.TrimSpace(line)
		if value == "" {
			value = defaultValue
		}
		if err := validate(value); err != nil {
			fprintf(w.out, "%s\n", err.Error())
			continue
		}
		return value, nil
	}
}

func (w wizard) askBool(label string, defaultValue bool) (bool, error) {
	defaultText := "y/N"
	if defaultValue {
		defaultText = "Y/n"
	}
	for {
		fprintf(w.out, "%s [%s]: ", label, defaultText)
		line, err := w.readLine()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return defaultValue, nil
			}
			return false, err
		}
		switch strings.ToLower(strings.TrimSpace(line)) {
		case "":
			return defaultValue, nil
		case "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		default:
			fprintf(w.out, "Please answer yes or no.\n")
		}
	}
}

func (w wizard) readLine() (string, error) {
	line, err := w.in.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), err
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func validateNonBlank(value string) error {
	if strings.TrimSpace(value) == "" {
		return errors.New("value is required")
	}
	return nil
}

func validatePort(value string) error {
	port, err := strconv.Atoi(value)
	if err != nil || port < 1 || port > 65535 {
		return errors.New("port must be a number between 1 and 65535")
	}
	return nil
}

func validateBudget(value string) error {
	budget, err := strconv.ParseFloat(value, 64)
	if err != nil || budget <= 0 {
		return errors.New("budget must be a positive USD amount like 10.00")
	}
	return nil
}
