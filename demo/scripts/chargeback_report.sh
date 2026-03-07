#!/usr/bin/env bash
set -euo pipefail

# Canonical source: demo/scripts/chargeback_report.sh
# Helm copy: deploy/helm/ai-control-plane/files/chargeback_report.sh
# Synchronization: Run 'make generate' to update the Helm copy.

# Chargeback Report Generation Script for AI Control Plane
#
# Purpose: Generates monthly chargeback reports with cost center attribution,
#          variance detection, and finance system integration.
#
# Responsibilities:
#   - High-level orchestration of report generation
#   - CLI argument parsing and help menu
#   - Prerequisite validation
#   - Notification triggering
#
# Non-scope:
#   - Database querying (handled by lib/chargeback_db.sh)
#   - Spend analysis (handled by lib/chargeback_analysis.sh)
#   - Report rendering (handled by lib/chargeback_render.sh)
#   - IO and boundaries (handled by lib/chargeback_io.sh)
#
# Invariants:
#   - Exit code 1 indicates variance threshold exceeded or high anomalies
#

#-------------------------------------------------------------------------------
# Configuration
#-------------------------------------------------------------------------------

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Save SCRIPT_DIR before sourcing libraries that may overwrite it
CHARGEBACK_SCRIPT_DIR="$SCRIPT_DIR"

# Source common.sh for color definitions and helpers
# shellcheck source=../../local/scripts/demo_scenarios/lib/common.sh
source "$PROJECT_ROOT/local/scripts/demo_scenarios/lib/common.sh"

# Source docker_sql library for differentiated error handling
# shellcheck source=lib/docker_sql.sh
source "$CHARGEBACK_SCRIPT_DIR/lib/docker_sql.sh"

# Source notification_dispatcher for webhook alerts
# shellcheck source=lib/notification_dispatcher.sh
source "$CHARGEBACK_SCRIPT_DIR/lib/notification_dispatcher.sh"

# Source chargeback libraries
# shellcheck source=lib/chargeback_db.sh
source "$CHARGEBACK_SCRIPT_DIR/lib/chargeback_db.sh"
# shellcheck source=lib/chargeback_analysis.sh
source "$CHARGEBACK_SCRIPT_DIR/lib/chargeback_analysis.sh"
# shellcheck source=lib/chargeback_render.sh
source "$CHARGEBACK_SCRIPT_DIR/lib/chargeback_render.sh"
# shellcheck source=lib/chargeback_io.sh
source "$CHARGEBACK_SCRIPT_DIR/lib/chargeback_io.sh"

# Restore SCRIPT_DIR
SCRIPT_DIR="$CHARGEBACK_SCRIPT_DIR"

# Default values
DB_NAME="${DB_NAME:-litellm}"
DB_USER="${DB_USER:-litellm}"
VERBOSE="${VERBOSE:-0}"
OUTPUT_FORMAT="markdown"
ARCHIVE_DIR="${ARCHIVE_DIR:-demo/backups/chargeback}"
VARIANCE_THRESHOLD="${VARIANCE_THRESHOLD:-15}"
ANOMALY_THRESHOLD="${ANOMALY_THRESHOLD:-200}"
NOTIFY="${NOTIFY:-false}"
SCHEMA_VERSION="1.0.0"

# Forecasting configuration
FORECAST_ENABLED="${FORECAST_ENABLED:-true}"
BUDGET_ALERT_THRESHOLD="${BUDGET_ALERT_THRESHOLD:-80}"

# Target month
TARGET_MONTH="${TARGET_MONTH:-}"

# Testability hooks
ACP_CHARGEBACK_SKIP_PREREQ="${ACP_CHARGEBACK_SKIP_PREREQ:-0}"

# Get docker compose command
DOCKER_COMPOSE="$(get_docker_compose)" || die "Neither 'docker compose' (V2) nor 'docker-compose' (V1) is available."

#-------------------------------------------------------------------------------
# Arguments
#-------------------------------------------------------------------------------

while [[ $# -gt 0 ]]; do
    case $1 in
    --month | -m)
        TARGET_MONTH="$2"
        shift 2
        ;;
    --format | -f)
        OUTPUT_FORMAT="$2"
        shift 2
        ;;
    --archive-dir | -a)
        ARCHIVE_DIR="$2"
        shift 2
        ;;
    --variance-threshold | -v)
        VARIANCE_THRESHOLD="$2"
        shift 2
        ;;
    --anomaly-threshold)
        ANOMALY_THRESHOLD="$2"
        shift 2
        ;;
    --forecast)
        FORECAST_ENABLED="true"
        shift
        ;;
    --no-forecast)
        FORECAST_ENABLED="false"
        shift
        ;;
    --budget-alert-threshold)
        BUDGET_ALERT_THRESHOLD="$2"
        shift 2
        ;;
    --notify | -n)
        NOTIFY="true"
        shift
        ;;
    --verbose)
        VERBOSE=1
        shift
        ;;
    --help | -h)
        cat <<'EOF'
Usage: chargeback_report.sh [OPTIONS]

Generate monthly chargeback reports with cost center attribution,
variance analysis, and spend forecasting for AI Control Plane usage.

Options:
  --month, -m YYYY-MM          Target month (default: previous calendar month)
  --format, -f FORMAT          Output format: markdown, json, csv, or all
                               (default: markdown)
  --archive-dir, -a DIR        Directory to archive reports
                               (default: demo/backups/chargeback)
  --variance-threshold, -v     Percentage threshold for variance warning
                               (default: 15)
  --anomaly-threshold          Percentage threshold for anomaly detection
                               (default: 200)
  --forecast                   Enable spend forecasting (default: enabled)
  --no-forecast                Disable spend forecasting
  --budget-alert-threshold     Budget % threshold for alerts (default: 80)
  --notify, -n                 Send webhook notifications to cost center owners
  --verbose                    Enable detailed output
  --help, -h                   Show this help message

Environment variables:
  DB_NAME                      Database name (default: litellm)
  DB_USER                      Database user (default: litellm)
  TARGET_MONTH                 Target month override (YYYY-MM format)
  ARCHIVE_DIR                  Archive directory override
  VARIANCE_THRESHOLD           Variance threshold override
  ANOMALY_THRESHOLD            Anomaly threshold override
  FORECAST_ENABLED             Enable/disable forecasting (true/false)
  BUDGET_ALERT_THRESHOLD       Budget alert threshold percent
  NOTIFY                       Enable notifications (true/false)
  SLACK_WEBHOOK_URL            Slack webhook for notifications
  GENERIC_WEBHOOK_URL          Generic webhook for finance system integration

Exit codes:
  0   - Success
  1   - Domain failure (variance threshold exceeded, high anomalies)
  2   - Prerequisites not ready (Docker/curl not installed)
  3   - Runtime error (SQL query failed)
  64  - Usage error (unknown options, invalid month format)
EOF
        exit 0
        ;;
    *)
        echo "Unknown option: $1"
        exit "$ACP_EXIT_USAGE"
        ;;
    esac
done

#-------------------------------------------------------------------------------
# Logic
#-------------------------------------------------------------------------------

check_prerequisites() {
    verbose_log "Checking prerequisites..."
    [[ "$ACP_CHARGEBACK_SKIP_PREREQ" -eq 1 ]] && return 0

    command -v docker >/dev/null 2>&1 || {
        error "Docker not installed"
        return 2
    }

    cd "$PROJECT_ROOT"
    local db_container
    db_container=$($DOCKER_COMPOSE -f demo/docker-compose.yml ps -q postgres 2>/dev/null | head -n 1)
    [[ -z "$db_container" ]] && {
        error "PostgreSQL container not found"
        return 2
    }
    docker exec "$db_container" pg_isready -U "$DB_USER" -d "$DB_NAME" >/dev/null 2>&1 || {
        error "DB not ready"
        return 2
    }

    init_notification_dispatcher >/dev/null 2>&1 || true
    return 0
}

send_notifications() {
    local report_month="$1" total_spend="$2" variance="$3" anomalies="$4"
    [[ "$NOTIFY" != "true" ]] && return 0
    notifications_enabled || return 0

    local variance_status_color="good"
    variance_exceeds_threshold "$variance" "$VARIANCE_THRESHOLD" && variance_status_color="danger"

    verbose_log "Sending notifications..."
    local payload
    payload=$(
        CHARGEBACK_PAYLOAD_EVENT="chargeback_report_generated" \
            CHARGEBACK_REPORT_MONTH="${report_month}" \
            CHARGEBACK_TOTAL_SPEND="${total_spend:-0}" \
            CHARGEBACK_VARIANCE="${variance}" \
            CHARGEBACK_ANOMALIES_JSON="${anomalies}" \
            CHARGEBACK_PAYLOAD_TIMESTAMP="$(get_timestamp)" \
            "${PROJECT_ROOT}/scripts/acpctl.sh" chargeback payload --target generic
    )
    [[ -n "${GENERIC_WEBHOOK_URL:-}" ]] && send_generic_webhook_alert "$GENERIC_WEBHOOK_URL" "$payload" || true

    if [[ -n "${SLACK_WEBHOOK_URL:-}" ]]; then
        local slack_payload
        slack_payload=$(
            CHARGEBACK_REPORT_MONTH="${report_month}" \
                CHARGEBACK_TOTAL_SPEND="${total_spend:-0}" \
                CHARGEBACK_VARIANCE="${variance}" \
                CHARGEBACK_SLACK_COLOR="${variance_status_color}" \
                CHARGEBACK_SLACK_EPOCH="$(date +%s)" \
                "${PROJECT_ROOT}/scripts/acpctl.sh" chargeback payload --target slack
        )
        send_slack_alert "$SLACK_WEBHOOK_URL" "$slack_payload" || true
    fi
}

# Main
if [[ -n "$TARGET_MONTH" ]]; then
    [[ "$TARGET_MONTH" =~ ^[0-9]{4}-[0-9]{2}$ ]] || die_usage "Invalid month: $TARGET_MONTH"
else
    TARGET_MONTH=$(date -d "last month" +%Y-%m 2>/dev/null || date -v-1m +%Y-%m)
fi

calculate_month_boundaries "$TARGET_MONTH"
check_prerequisites || exit "$?"

verbose_log "Collecting data for ${TARGET_MONTH}..."
TOTAL_SPEND=$(get_total_spend "$MONTH_START" "$MONTH_END") || TOTAL_SPEND="0"
METRICS=$(total_monthly_metrics "$MONTH_START" "$MONTH_END") || METRICS="0|0"
TOTAL_REQUESTS=$(echo "$METRICS" | cut -d'|' -f1)
TOTAL_TOKENS=$(echo "$METRICS" | cut -d'|' -f2)

COST_CENTER_DATA=$(get_cost_center_spend "$MONTH_START" "$MONTH_END") || COST_CENTER_DATA=""
MODEL_DATA=$(get_model_spend "$MONTH_START" "$MONTH_END") || MODEL_DATA=""
UNATTRIBUTED_DATA=$(get_unattributed_usage "$MONTH_START" "$MONTH_END") || UNATTRIBUTED_DATA=""
TOP_PRINCIPALS=$(get_top_principals "$MONTH_START" "$MONTH_END" 20) || TOP_PRINCIPALS=""
COST_CENTER_JSON="[]"
MODEL_JSON="[]"
if [[ "$OUTPUT_FORMAT" == "json" || "$OUTPUT_FORMAT" == "csv" || "$OUTPUT_FORMAT" == "all" ]]; then
    COST_CENTER_JSON=$(get_cost_center_spend_json "$MONTH_START" "$MONTH_END") || COST_CENTER_JSON="[]"
    MODEL_JSON=$(get_model_spend_json "$MONTH_START" "$MONTH_END") || MODEL_JSON="[]"
fi

PREV_MONTH_SPEND=$(get_total_spend "$PREV_MONTH_START" "$PREV_MONTH_END") || PREV_MONTH_SPEND="0"
VARIANCE=$(calculate_variance "$TOTAL_SPEND" "$PREV_MONTH_SPEND")
ANOMALIES=$(detect_anomalies "$COST_CENTER_DATA" "$ANOMALY_THRESHOLD" "$TOTAL_SPEND" "$PREV_MONTH_START" "$PREV_MONTH_END")

# Forecasting (if enabled)
FORECAST_VALUES="N/A,N/A,N/A"
DAILY_BURN="0"
DAYS_REMAINING="N/A"
EXHAUSTION_DATE="N/A"
TOTAL_BUDGET="0"
BUDGET_RISK="{}"
BUDGET_RISK_LEVEL="unknown"
BUDGET_RISK_PERCENT="N/A"
BUDGET_RISK_THRESHOLD_EXCEEDED="false"

if [[ "$FORECAST_ENABLED" == "true" ]]; then
    verbose_log "Calculating spend forecast..."

    # Get historical spend data (6 months)
    HISTORICAL_SPEND=$(get_historical_spend 6 "$MONTH_START") || HISTORICAL_SPEND=""

    # Get budget totals
    TOTAL_BUDGET=$(get_budget_totals) || TOTAL_BUDGET="0"

    # Generate forecast
    FORECAST_VALUES=$(forecast_spend "$HISTORICAL_SPEND")

    # Calculate burn rate
    BURN_RATE_DATA=$(calculate_burn_rate "$TOTAL_SPEND" "$MONTH_START" "$TOTAL_BUDGET")
    DAILY_BURN=$(echo "$BURN_RATE_DATA" | cut -d'|' -f1)
    DAYS_REMAINING=$(echo "$BURN_RATE_DATA" | cut -d'|' -f2)
    EXHAUSTION_DATE=$(echo "$BURN_RATE_DATA" | cut -d'|' -f3)

    # Check budget risk (using 3-month forecast total)
    FORECAST_M1=$(echo "$FORECAST_VALUES" | cut -d',' -f1)
    FORECAST_M2=$(echo "$FORECAST_VALUES" | cut -d',' -f2)
    FORECAST_M3=$(echo "$FORECAST_VALUES" | cut -d',' -f3)
    FORECAST_3MO_TOTAL=$(echo "$FORECAST_M1 + $FORECAST_M2 + $FORECAST_M3" | bc)

    BUDGET_RISK=$(check_budget_risk "$FORECAST_3MO_TOTAL" "$TOTAL_BUDGET" "$BUDGET_ALERT_THRESHOLD")
    IFS='|' read -r BUDGET_RISK_LEVEL BUDGET_RISK_PERCENT BUDGET_RISK_THRESHOLD_EXCEEDED <<<"$BUDGET_RISK"
fi

EXIT_CODE=0
case "$OUTPUT_FORMAT" in
csv) output_csv "$TARGET_MONTH" "$COST_CENTER_JSON" ;;
json)
    output_json "$TARGET_MONTH" "$MONTH_START" "$MONTH_END" \
        "$TOTAL_SPEND" "$TOTAL_REQUESTS" "$TOTAL_TOKENS" \
        "$COST_CENTER_JSON" "$MODEL_JSON" \
        "$VARIANCE" "$PREV_MONTH_SPEND" "$ANOMALIES" \
        "$FORECAST_VALUES" "$DAILY_BURN" "$DAYS_REMAINING" "$EXHAUSTION_DATE" "$TOTAL_BUDGET" \
        "$BUDGET_RISK_LEVEL" "$BUDGET_RISK_PERCENT" "$BUDGET_RISK_THRESHOLD_EXCEEDED"
    ;;
markdown)
    output_markdown "$TARGET_MONTH" "$MONTH_START" "$MONTH_END" \
        "$TOTAL_SPEND" "$TOTAL_REQUESTS" "$TOTAL_TOKENS" \
        "$COST_CENTER_DATA" "$MODEL_DATA" "$UNATTRIBUTED_DATA" \
        "$TOP_PRINCIPALS" "$VARIANCE" "$PREV_MONTH_SPEND" "$ANOMALIES" \
        "$FORECAST_VALUES" "$DAILY_BURN" "$DAYS_REMAINING" "$EXHAUSTION_DATE" "$TOTAL_BUDGET" "$BUDGET_RISK"
    ;;
all)
    MD=$(output_markdown "$TARGET_MONTH" "$MONTH_START" "$MONTH_END" "$TOTAL_SPEND" "$TOTAL_REQUESTS" "$TOTAL_TOKENS" "$COST_CENTER_DATA" "$MODEL_DATA" "$UNATTRIBUTED_DATA" "$TOP_PRINCIPALS" "$VARIANCE" "$PREV_MONTH_SPEND" "$ANOMALIES" "$FORECAST_VALUES" "$DAILY_BURN" "$DAYS_REMAINING" "$EXHAUSTION_DATE" "$TOTAL_BUDGET" "$BUDGET_RISK")
    archive_report "$TARGET_MONTH" "md" "$MD" >/dev/null
    JSON=$(output_json "$TARGET_MONTH" "$MONTH_START" "$MONTH_END" "$TOTAL_SPEND" "$TOTAL_REQUESTS" "$TOTAL_TOKENS" "$COST_CENTER_JSON" "$MODEL_JSON" "$VARIANCE" "$PREV_MONTH_SPEND" "$ANOMALIES" "$FORECAST_VALUES" "$DAILY_BURN" "$DAYS_REMAINING" "$EXHAUSTION_DATE" "$TOTAL_BUDGET" "$BUDGET_RISK_LEVEL" "$BUDGET_RISK_PERCENT" "$BUDGET_RISK_THRESHOLD_EXCEEDED")
    archive_report "$TARGET_MONTH" "json" "$JSON" >/dev/null
    CSV=$(output_csv "$TARGET_MONTH" "$COST_CENTER_JSON")
    archive_report "$TARGET_MONTH" "csv" "$CSV" >/dev/null
    echo "$MD"
    ;;
*) die_usage "Unknown format: $OUTPUT_FORMAT" ;;
esac

if [[ "$NOTIFY" == "true" ]]; then send_notifications "$TARGET_MONTH" "$TOTAL_SPEND" "$VARIANCE" "$ANOMALIES"; fi

variance_exceeds_threshold "$VARIANCE" "$VARIANCE_THRESHOLD" && {
    warning "Variance exceeded: ${VARIANCE}%"
    EXIT_CODE=1
}
[[ "$ANOMALIES" != "[]" && -n "$ANOMALIES" ]] && {
    warning "Anomalies detected"
    EXIT_CODE=1
}

exit "$EXIT_CODE"
