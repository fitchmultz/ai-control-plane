#!/usr/bin/env bash
# demo/scripts/lib/chargeback_render.sh - Report rendering library for chargeback reports
#
# Purpose: Handles all formatting and rendering of chargeback data into
#          Markdown, JSON, and CSV formats.
#
# Responsibilities:
#   - Parse raw database table output into structured formats
#   - Generate Markdown reports with tables and executive summaries
#   - Generate JSON reports adhering to stable schema
#   - Generate CSV reports for finance system integration
#   - Provide UI helper functions (success, warning, error, section_header)
#
# Non-scope:
#   - Does NOT interact with the database
#   - Does NOT perform spend analysis or variance calculation
#
# Invariants:
#   - Adheres to SCHEMA_VERSION for JSON output
#

if [[ -n "${ACP_CHARGEBACK_RENDER_LOADED:-}" ]]; then
    return 0
fi
readonly ACP_CHARGEBACK_RENDER_LOADED=1

# Print section header
section_header() {
    # shellcheck disable=SC2154
    if [[ "$OUTPUT_FORMAT" != "json" && "$OUTPUT_FORMAT" != "csv" ]]; then
        echo ""
        # shellcheck disable=SC2154
        echo -e "${COLOR_BOLD}$1${COLOR_RESET}"
    fi
}

# Print info line
info_line() {
    if [[ "$OUTPUT_FORMAT" != "json" && "$OUTPUT_FORMAT" != "csv" ]]; then
        echo "  $1"
    fi
}

# Print success
success() {
    if [[ "$OUTPUT_FORMAT" != "json" && "$OUTPUT_FORMAT" != "csv" ]]; then
        # shellcheck disable=SC2154
        echo -e "${COLOR_GREEN}$1${COLOR_RESET}"
    fi
}

# Print warning
warning() {
    if [[ "$OUTPUT_FORMAT" != "json" && "$OUTPUT_FORMAT" != "csv" ]]; then
        # shellcheck disable=SC2154
        echo -e "${COLOR_YELLOW}$1${COLOR_RESET}"
    fi
}

# Print error
error() {
    if [[ "$OUTPUT_FORMAT" != "json" && "$OUTPUT_FORMAT" != "csv" ]]; then
        # shellcheck disable=SC2154
        echo -e "${COLOR_RED}$1${COLOR_RESET}"
    else
        echo "Error: $1" >&2
    fi
}

# Parse cost center data into arrays for JSON/CSV output
parse_cost_center_data() {
    local data="$1"
    local json_array="[]"

    while IFS='|' read -r cc team requests tokens spend percent; do
        # Skip header and empty lines
        [[ -z "$cc" || "$cc" == " cost_center" || "$cc" == "cost_center" ]] && continue
        [[ "$cc" =~ ^-+$ ]] && continue
        [[ "$cc" =~ ^\( ]] && continue

        # Trim whitespace
        cc=$(echo "$cc" | xargs)
        team=$(echo "$team" | xargs)
        requests=$(echo "$requests" | xargs)
        tokens=$(echo "$tokens" | xargs)
        spend=$(echo "$spend" | xargs)
        percent=$(echo "$percent" | xargs)

        # Add to JSON array
        local entry="{\"cost_center\":\"${cc}\",\"team\":\"${team}\",\"request_count\":${requests:-0},\"token_count\":${tokens:-0},\"spend_amount\":${spend:-0},\"percent_of_total\":${percent:-0}}"
        if [[ "$json_array" == "[]" ]]; then
            json_array="[$entry]"
        else
            json_array="${json_array%]},$entry]"
        fi
    done <<<"$data"

    echo "$json_array"
}

# Parse model spend data into JSON
parse_model_data() {
    local data="$1"
    local json_array="[]"

    while IFS='|' read -r model requests tokens spend; do
        # Skip header and empty lines
        [[ -z "$model" || "$model" == " model_id" || "$model" == "model_id" ]] && continue
        [[ "$model" =~ ^-+$ ]] && continue
        [[ "$model" =~ ^\( ]] && continue

        # Trim whitespace
        model=$(echo "$model" | xargs)
        requests=$(echo "$requests" | xargs)
        tokens=$(echo "$tokens" | xargs)
        spend=$(echo "$spend" | xargs)

        local entry="{\"model\":\"${model}\",\"request_count\":${requests:-0},\"token_count\":${tokens:-0},\"spend_amount\":${spend:-0}}"
        if [[ "$json_array" == "[]" ]]; then
            json_array="[$entry]"
        else
            json_array="${json_array%]},$entry]"
        fi
    done <<<"$data"

    echo "$json_array"
}

# Output CSV format
output_csv() {
    local report_month="$1"
    local cost_center_data="$2"

    # CSV Header
    echo "CostCenter,Team,SpendAmount,RequestCount,TokenCount,PercentOfTotal,ReportMonth"

    # Data rows
    while IFS='|' read -r cc team requests tokens spend percent; do
        # Skip header and empty lines
        [[ -z "$cc" || "$cc" == " cost_center" || "$cc" == "cost_center" ]] && continue
        [[ "$cc" =~ ^-+$ ]] && continue
        [[ "$cc" =~ ^\( ]] && continue

        # Trim whitespace
        cc=$(echo "$cc" | xargs)
        team=$(echo "$team" | xargs)
        requests=$(echo "$requests" | xargs)
        tokens=$(echo "$tokens" | xargs)
        spend=$(echo "$spend" | xargs)
        percent=$(echo "$percent" | xargs)

        echo "${cc},${team},${spend},${requests},${tokens},${percent},${report_month}"
    done <<<"$cost_center_data"
}

# Parse individual forecast value from comma-separated string
parse_forecast_value() {
    local values="$1"
    local index="$2"

    if [[ "$values" == "N/A,N/A,N/A" ]]; then
        echo "null"
        return
    fi

    local value
    value=$(echo "$values" | cut -d',' -f"$index")
    if [[ -z "$value" || "$value" == "N/A" ]]; then
        echo "null"
    else
        echo "$value"
    fi
}

# Output JSON format
output_json() {
    local report_month="$1"
    local month_start="$2"
    local month_end="$3"
    local total_spend="$4"
    local total_requests="$5"
    local total_tokens="$6"
    local cost_center_data="$7"
    local model_data="$8"
    local unattributed_data="$9"
    local variance="${10}"
    local prev_month_spend="${11}"
    local anomalies="${12}"
    local forecast_values="${13:-N/A,N/A,N/A}"
    local daily_burn="${14:-0}"
    local days_remaining="${15:-N/A}"
    local exhaustion_date="${16:-N/A}"
    local total_budget="${17:-0}"
    local budget_risk="${18:-{}}"

    local timestamp
    # shellcheck disable=SC2154
    timestamp=$(get_timestamp)

    # Parse data into JSON arrays
    local cost_center_json
    cost_center_json=$(parse_cost_center_data "$cost_center_data")

    local model_json
    model_json=$(parse_model_data "$model_data")

    # Calculate attribution coverage
    local unattributed_spend=0
    local coverage_percent=100

    # Extract unattributed amount from data
    while IFS='|' read -r cc team requests tokens spend percent; do
        cc=$(echo "$cc" | xargs 2>/dev/null || echo "")
        if [[ "$cc" == "unknown-cc" ]]; then
            unattributed_spend=$(echo "$spend" | xargs)
            break
        fi
    done <<<"$cost_center_data"

    if [[ -n "$total_spend" && "$total_spend" != "0" ]]; then
        coverage_percent=$(echo "scale=2; (($total_spend - $unattributed_spend) / $total_spend) * 100" | bc 2>/dev/null || echo "100")
    fi

    # Build forecast section
    local forecast_enabled_bool="true"
    if [[ "${FORECAST_ENABLED:-true}" != "true" ]]; then
        forecast_enabled_bool="false"
    fi

    local days_remaining_json="null"
    if [[ "$days_remaining" != "N/A" && "$days_remaining" =~ ^[0-9]+$ ]]; then
        days_remaining_json="$days_remaining"
    fi

    local exhaustion_date_json="\"N/A\""
    if [[ "$exhaustion_date" != "N/A" && "$exhaustion_date" != "" ]]; then
        exhaustion_date_json="\"$exhaustion_date\""
    fi

    # Build JSON output
    cat <<EOF
{
  "schema_version": "${SCHEMA_VERSION}",
  "report_metadata": {
    "generated_at": "${timestamp}",
    "report_month": "${report_month}",
    "period_start": "${month_start}",
    "period_end": "${month_end}"
  },
  "executive_summary": {
    "total_spend": ${total_spend:-0},
    "total_requests": ${total_requests:-0},
    "total_tokens": ${total_tokens:-0},
    "attribution_coverage_percent": ${coverage_percent},
    "unattributed_spend": ${unattributed_spend:-0}
  },
  "allocations_by_cost_center": ${cost_center_json},
  "allocations_by_model": ${model_json},
  "variance_analysis": {
    "previous_month_spend": ${prev_month_spend:-0},
    "variance_percent": $(if [[ "${variance:-0}" == "N/A" ]]; then echo '"N/A"'; else echo "${variance:-0}"; fi),
    "variance_threshold": ${VARIANCE_THRESHOLD},
    "variance_threshold_exceeded": $(if variance_exceeds_threshold "$variance" "$VARIANCE_THRESHOLD"; then echo "true"; else echo "false"; fi)
  },
  "anomalies": ${anomalies},
  "forecast": {
    "enabled": ${forecast_enabled_bool},
    "methodology": "linear_regression",
    "confidence_note": "Estimates based on historical trends; actual spend may vary +/- 20%",
    "predictions": {
      "month_1": $(parse_forecast_value "$forecast_values" 1),
      "month_2": $(parse_forecast_value "$forecast_values" 2),
      "month_3": $(parse_forecast_value "$forecast_values" 3)
    },
    "burn_rate": {
      "daily_average": ${daily_burn:-0},
      "days_until_exhaustion": ${days_remaining_json},
      "exhaustion_date": ${exhaustion_date_json}
    },
    "budget_analysis": {
      "total_budget": ${total_budget:-0},
      "risk_assessment": ${budget_risk:-{}}
    }
  },
  "configuration": {
    "variance_threshold_percent": ${VARIANCE_THRESHOLD},
    "anomaly_threshold_percent": ${ANOMALY_THRESHOLD}
  }
}
EOF
}

# Output Markdown format
output_markdown() {
    local report_month="$1"
    local month_start="$2"
    local month_end="$3"
    local total_spend="$4"
    local total_requests="$5"
    local total_tokens="$6"
    local cost_center_data="$7"
    local model_data="$8"
    local unattributed_data="$9"
    local top_principals="${10}"
    local variance="${11}"
    local prev_month_spend="${12}"
    local anomalies="${13}"
    local forecast_values="${14:-N/A,N/A,N/A}"
    local daily_burn="${15:-0}"
    local days_remaining="${16:-N/A}"
    local exhaustion_date="${17:-N/A}"
    local total_budget="${18:-0}"
    local budget_risk="${19:-{}}"

    local timestamp
    timestamp=$(get_timestamp)

    # Calculate unattributed amount
    local unattributed_spend=0
    local coverage_percent=100

    while IFS='|' read -r cc team requests tokens spend percent; do
        cc=$(echo "$cc" | xargs 2>/dev/null || echo "")
        if [[ "$cc" == "unknown-cc" ]]; then
            unattributed_spend=$(echo "$spend" | xargs)
            break
        fi
    done <<<"$cost_center_data"

    if [[ -n "$total_spend" && "$total_spend" != "0" ]]; then
        coverage_percent=$(echo "scale=2; (($total_spend - $unattributed_spend) / $total_spend) * 100" | bc 2>/dev/null || echo "100")
    fi

    # Determine variance status
    local variance_status="✓ Within Threshold"
    if variance_exceeds_threshold "$variance" "$VARIANCE_THRESHOLD"; then
        variance_status="⚠ THRESHOLD EXCEEDED"
    fi

    echo "# Financial Chargeback Report"
    echo ""
    echo "**Reporting Period:** ${month_start} to ${month_end}  "
    echo "**Generated:** ${timestamp}  "
    echo "**Report Type:** Chargeback Allocation"
    echo ""
    echo "---"
    echo ""
    echo "## Executive Summary"
    echo ""
    echo "| Metric | Value |"
    echo "|--------|-------|"
    printf "| **Total AI Spend** | \$%.2f |\n" "${total_spend:-0}"
    printf "| **Total Requests** | %'d |\n" "${total_requests:-0}"
    printf "| **Total Tokens** | %'d |\n" "${total_tokens:-0}"
    printf "| **Attribution Coverage** | %.1f%% |\n" "$coverage_percent"
    printf "| **Unattributed Usage** | \$%.2f (%.1f%%) |\n" "${unattributed_spend:-0}" "$((100 - $(echo "$coverage_percent" | cut -d. -f1)))"
    echo ""

    echo "### Month-over-Month Variance"
    echo ""
    echo "| Metric | Value |"
    echo "|--------|-------|"
    printf "| **Previous Month Spend** | \$%.2f |\n" "${prev_month_spend:-0}"
    if [[ "$variance" == "N/A" ]]; then
        echo "| **Variance** | N/A (no previous data) |"
    else
        printf "| **Variance** | %+.1f%% %s |\n" "${variance}" "$variance_status"
    fi
    echo ""

    echo "---"
    echo ""
    echo "## Allocation by Cost Center"
    echo ""
    echo "| Cost Center | Team | Requests | Tokens | Spend | % of Total |"
    echo "|-------------|------|----------|--------|-------|------------|"

    while IFS='|' read -r cc team requests tokens spend percent; do
        [[ -z "$cc" || "$cc" == " cost_center" || "$cc" == "cost_center" ]] && continue
        [[ "$cc" =~ ^-+$ ]] && continue
        [[ "$cc" =~ ^\( ]] && continue

        cc=$(echo "$cc" | xargs)
        team=$(echo "$team" | xargs)
        requests=$(echo "$requests" | xargs)
        tokens=$(echo "$tokens" | xargs)
        spend=$(echo "$spend" | xargs)
        percent=$(echo "$percent" | xargs)

        printf "| %s | %s | %'d | %'d | \$%.2f | %.1f%% |\n" "$cc" "$team" "${requests:-0}" "${tokens:-0}" "${spend:-0}" "${percent:-0}"
    done <<<"$cost_center_data"

    echo ""
    echo "---"
    echo ""
    echo "## Top Principals by Spend"
    echo ""
    echo "| Principal | Team | Cost Center | Requests | Spend |"
    echo "|-----------|------|-------------|----------|-------|"

    while IFS='|' read -r alias spend requests team cc; do
        [[ -z "$alias" || "$alias" == " key_alias" || "$alias" == "key_alias" ]] && continue
        [[ "$alias" =~ ^-+$ ]] && continue
        [[ "$alias" =~ ^\( ]] && continue

        alias=$(echo "$alias" | xargs)
        spend=$(echo "$spend" | xargs)
        requests=$(echo "$requests" | xargs)
        team=$(echo "$team" | xargs)
        cc=$(echo "$cc" | xargs)

        printf "| %s | %s | %s | %'d | \$%.2f |\n" "$alias" "$team" "$cc" "${requests:-0}" "${spend:-0}"
    done <<<"$top_principals"

    echo ""
    echo "---"
    echo ""
    echo "## Spend by Model"
    echo ""
    echo "| Model | Requests | Tokens | Spend |"
    echo "|-------|----------|--------|-------|"

    while IFS='|' read -r model requests tokens spend; do
        [[ -z "$model" || "$model" == " model_id" || "$model" == "model_id" ]] && continue
        [[ "$model" =~ ^-+$ ]] && continue
        [[ "$model" =~ ^\( ]] && continue

        model=$(echo "$model" | xargs)
        requests=$(echo "$requests" | xargs)
        tokens=$(echo "$tokens" | xargs)
        spend=$(echo "$spend" | xargs)

        printf "| %s | %'d | %'d | \$%.2f |\n" "$model" "${requests:-0}" "${tokens:-0}" "${spend:-0}"
    done <<<"$model_data"

    # Anomalies section
    if [[ "$anomalies" != "[]" && "$anomalies" != "" ]]; then
        echo ""
        echo "---"
        echo ""
        echo "## Anomalies Detected"
        echo ""
        echo "| Cost Center | Type | Current Spend | Previous Spend | Spike % |"
        echo "|-------------|------|---------------|----------------|---------|"

        # Parse anomalies JSON (simple parsing for display)
        echo "$anomalies" | tr '}' '\n' | while read -r line; do
            [[ -z "$line" || "$line" == "[" || "$line" == "]" ]] && continue

            local cc type current previous spike
            cc=$(echo "$line" | grep -o '"cost_center":"[^"]*"' | cut -d'"' -f4)
            type=$(echo "$line" | grep -o '"type":"[^"]*"' | cut -d'"' -f4)
            current=$(echo "$line" | grep -o '"current_spend":[0-9.]*' | cut -d: -f2)
            previous=$(echo "$line" | grep -o '"previous_spend":[0-9.]*' | cut -d: -f2)
            spike=$(echo "$line" | grep -o '"spike_percent":[0-9.-]*' | cut -d: -f2)

            if [[ -n "$cc" ]]; then
                printf "| %s | %s | \$%.2f | \$%.2f | %+.1f%% |\n" "$cc" "$type" "${current:-0}" "${previous:-0}" "${spike:-0}"
            fi
        done
    fi

    # Forecast section
    if [[ "${FORECAST_ENABLED:-true}" == "true" && "$forecast_values" != "N/A,N/A,N/A" ]]; then
        local forecast_m1 forecast_m2 forecast_m3
        forecast_m1=$(echo "$forecast_values" | cut -d',' -f1)
        forecast_m2=$(echo "$forecast_values" | cut -d',' -f2)
        forecast_m3=$(echo "$forecast_values" | cut -d',' -f3)

        echo ""
        echo "---"
        echo ""
        echo "## Spend Forecast"
        echo ""
        echo "_Predictions based on linear regression of historical spend trends._"
        echo "_Confidence: Estimates may vary +/- 20% from actual spend._"
        echo ""

        echo "### 3-Month Projection"
        echo ""
        echo "| Period | Predicted Spend |"
        echo "|--------|-----------------|"
        printf "| Month +1 | \$%.2f |\n" "${forecast_m1:-0}"
        printf "| Month +2 | \$%.2f |\n" "${forecast_m2:-0}"
        printf "| Month +3 | \$%.2f |\n" "${forecast_m3:-0}"

        local forecast_total
        if [[ "$forecast_m1" != "N/A" && "$forecast_m2" != "N/A" && "$forecast_m3" != "N/A" ]]; then
            forecast_total=$(echo "$forecast_m1 + $forecast_m2 + $forecast_m3" | bc)
            printf "| **3-Mo Total** | **\$%.2f** |\n" "${forecast_total:-0}"
        else
            echo "| **3-Mo Total** | N/A |"
        fi
        echo ""

        echo "### Burn Rate Analysis"
        echo ""
        echo "| Metric | Value |"
        echo "|--------|-------|"
        printf "| **Daily Average** | \$%.2f |\n" "${daily_burn:-0}"

        if [[ "$days_remaining" != "N/A" ]]; then
            printf "| **Days Until Budget Exhaustion** | %s |\n" "$days_remaining"
            printf "| **Projected Exhaustion Date** | %s |\n" "$exhaustion_date"
        else
            echo "| **Days Until Budget Exhaustion** | N/A (no budget set) |"
        fi

        if [[ -n "$total_budget" && "$total_budget" != "0" && "$total_budget" != "0.00" ]]; then
            printf "| **Total Budget** | \$%.2f |\n" "$total_budget"
        fi
        echo ""

        # Budget risk alert
        local risk_level
        risk_level=$(echo "$budget_risk" | grep -o '"risk_level":"[^"]*"' | cut -d'"' -f4)
        if [[ "$risk_level" == "high" ]]; then
            echo "### ⚠️ Budget Alert"
            echo ""
            echo "**Risk Level: HIGH** - Forecasted spend is projected to exceed the budget alert threshold."
            echo ""
            echo "Recommended actions:"
            echo "- Review active API key budgets"
            echo "- Consider implementing usage limits"
            echo "- Notify cost center owners of projected overage"
            echo ""
        elif [[ "$risk_level" == "medium" ]]; then
            echo "### 💡 Budget Notice"
            echo ""
            echo "**Risk Level: MEDIUM** - Forecasted spend is approaching budget thresholds."
            echo ""
        fi
    fi

    echo ""
    echo "---"
    echo ""
    echo "*Report generated by AI Control Plane - Chargeback Reporting*"
    # shellcheck disable=SC2154
    echo "*Schema Version: ${SCHEMA_VERSION}*"
}
