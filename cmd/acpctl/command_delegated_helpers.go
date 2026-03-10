// command_delegated_helpers.go - Shared delegated command spec helpers.
//
// Purpose:
//   - Keep Make-backed and bridge-backed leaf spec construction consistent
//     across delegated command domains.
//
// Responsibilities:
//   - Define reusable Make leaf constructors.
//   - Define reusable bridge leaf constructors.
//
// Scope:
//   - Shared delegated command metadata helpers only.
//
// Usage:
//   - Used by delegated command domain files in `cmd/acpctl`.
//
// Invariants/Assumptions:
//   - Delegated command specs remain thin metadata-only definitions.
package main

func makeLeafSpec(name string, summary string, target string) *commandSpec {
	return &commandSpec{
		Name:              name,
		Summary:           summary,
		Description:       summary + ".",
		AllowTrailingArgs: true,
		Backend: commandBackend{
			Kind:       commandBackendMake,
			MakeTarget: target,
		},
	}
}

func bridgeLeafSpec(name string, summary string, relativePath string) *commandSpec {
	return bridgeLeafSpecWithArgs(name, summary, relativePath)
}

func bridgeLeafSpecWithArgs(name string, summary string, relativePath string, bridgeArgs ...string) *commandSpec {
	return &commandSpec{
		Name:              name,
		Summary:           summary,
		Description:       summary + ".",
		AllowTrailingArgs: true,
		Backend: commandBackend{
			Kind:               commandBackendBridge,
			BridgeRelativePath: relativePath,
			BridgeArgs:         append([]string(nil), bridgeArgs...),
		},
	}
}
