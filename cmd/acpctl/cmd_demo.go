// cmd_demo.go - Delegated demo command tree.
//
// Purpose:
//   - Own the demo workflow command surface.
//
// Responsibilities:
//   - Define the Make-backed demo scenario, preset, and snapshot commands.
//
// Scope:
//   - Demo command metadata only.
//
// Usage:
//   - Registered by `command_registry.go` as the `demo` root command.
//
// Invariants/Assumptions:
//   - Demo orchestration remains Make-backed.
package main

func demoCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "demo",
		Summary:     "Demo scenario, preset, and snapshot operations",
		Description: "Demo scenario, preset, and snapshot operations.",
		Examples: []string{
			"acpctl demo scenario SCENARIO=1",
			"acpctl demo preset PRESET=executive-demo",
		},
		Children: []*commandSpec{
			makeLeafSpec("scenario", "Run a specific demo scenario", "demo-scenario"),
			makeLeafSpec("all", "Run all demo scenarios", "demo-all"),
			makeLeafSpec("preset", "Run a named demo preset", "demo-preset"),
			makeLeafSpec("snapshot", "Create a named demo snapshot", "demo-snapshot"),
			makeLeafSpec("restore", "Restore a named demo snapshot", "demo-restore"),
		},
	}
}
