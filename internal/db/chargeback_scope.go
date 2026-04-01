// Package db provides typed PostgreSQL runtime, readonly, and admin services.
//
// Purpose:
//   - Centralize tenant-safe chargeback query scoping for typed database readers.
//
// Responsibilities:
//   - Hold the alias-namespace filters applied to chargeback queries.
//   - Render deterministic SQL WHERE fragments for spend-log and key-budget
//     joins.
//
// Scope:
//   - Chargeback query scoping only.
//
// Usage:
//   - Construct via chargeback composition and pass to NewChargebackReader.
//
// Invariants/Assumptions:
//   - Scope matches tracked tenant key namespace prefixes.
//   - Empty scope means repository-wide chargeback queries.
package db

import (
	"fmt"
	"sort"
	"strings"
)

// ChargebackQueryScope narrows chargeback queries to one or more key namespaces.
type ChargebackQueryScope struct {
	NamespacePrefixes []string
}

func (s *ChargebackQueryScope) spendLogAliasFilter(aliasColumn string) string {
	return scopedAliasWhere(aliasColumn, s)
}

func (s *ChargebackQueryScope) verificationAliasFilter(aliasColumn string) string {
	return scopedAliasWhere(aliasColumn, s)
}

func scopedAliasWhere(aliasColumn string, scope *ChargebackQueryScope) string {
	if scope == nil {
		return ""
	}
	prefixes := dedupeChargebackPrefixes(scope.NamespacePrefixes)
	if len(prefixes) == 0 {
		return ""
	}
	clauses := make([]string, 0, len(prefixes))
	for _, prefix := range prefixes {
		clauses = append(clauses, fmt.Sprintf("%s LIKE '%s--%%'", aliasColumn, sqlLiteral(prefix)))
	}
	return " AND (" + strings.Join(clauses, " OR ") + ")"
}

func dedupeChargebackPrefixes(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	sort.Strings(result)
	return result
}
