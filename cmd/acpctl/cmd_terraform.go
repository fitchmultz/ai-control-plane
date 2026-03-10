// cmd_terraform.go - Delegated Terraform command tree.
//
// Purpose:
//   - Own the Terraform helper command surface.
//
// Responsibilities:
//   - Define Make-backed Terraform workflow commands.
//
// Scope:
//   - Terraform command metadata only.
//
// Usage:
//   - Registered by `command_registry.go` as the `terraform` root command.
//
// Invariants/Assumptions:
//   - Terraform orchestration remains Make-backed.
package main

func terraformCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "terraform",
		Summary:     "Terraform provisioning workflow helpers",
		Description: "Terraform provisioning workflow helpers.",
		Examples: []string{
			"acpctl terraform init",
			"acpctl terraform plan",
			"acpctl terraform apply",
		},
		Children: []*commandSpec{
			makeLeafSpec("init", "Initialize Terraform", "tf-init"),
			makeLeafSpec("plan", "Run Terraform plan", "tf-plan"),
			makeLeafSpec("apply", "Run Terraform apply", "tf-apply"),
			makeLeafSpec("destroy", "Run Terraform destroy", "tf-destroy"),
			makeLeafSpec("fmt", "Format Terraform files", "tf-fmt"),
			makeLeafSpec("validate", "Validate Terraform configuration", "tf-validate"),
		},
	}
}
