#!/usr/bin/env bash
# demo/scripts/lib/chargeback_analysis.sh - Spend analysis library for chargeback reports
#
# Purpose: Provides functions for calculating variance and detecting anomalies
#          in spend patterns across months and cost centers.
#
# Responsibilities:
#   - Calculate percentage variance between current and previous month
#   - Detect spend spikes (anomalies) per cost center
#   - Check if variance exceeds defined thresholds
#
# Non-scope:
#   - Does NOT interact with the database directly (relies on chargeback_db.sh)
#   - Does NOT handle report rendering
#
# Invariants:
#   - Relies on 'bc' for floating-point calculations
#

if [[ -n "${ACP_CHARGEBACK_ANALYSIS_LOADED:-}" ]]; then
    return 0
fi
readonly ACP_CHARGEBACK_ANALYSIS_LOADED=1

# Calculate variance between current and previous month
calculate_variance() {
    local current="$1"
    local previous="$2"

    if [[ -z "$previous" || "$previous" == "0" || "$previous" == "0.00" ]]; then
        echo "N/A"
        return
    fi

    # Calculate percentage change: ((current - previous) / previous) * 100
    local variance
    variance=$(echo "scale=2; (($current - $previous) / $previous) * 100" | bc 2>/dev/null || echo "0")
    echo "$variance"
}

# Check if variance exceeds threshold
variance_exceeds_threshold() {
    local variance="$1"
    local threshold="$2"

    if [[ "$variance" == "N/A" ]]; then
        return 1
    fi

    # Use bc for float comparison
    if echo "$variance >= $threshold" | bc 2>/dev/null | grep -q "1"; then
        return 0
    elif echo "$variance <= -$threshold" | bc 2>/dev/null | grep -q "1"; then
        return 0
    fi
    return 1
}

# Detect anomalies (cost centers with unusual spend patterns)
detect_anomalies() {
    local _cost_center_data="$1"
    local threshold="$2"
    local _current_total="$3"
    local prev_month_start="$4"
    local prev_month_end="$5"

    get_cost_center_anomalies_json "$MONTH_START" "$MONTH_END" "$prev_month_start" "$prev_month_end" "$threshold"
}

# Forecast future spend using linear regression on historical data
# Input: historical_data (month|spend pairs, pipe-separated lines)
# Output: predicted spend for next 3 months (comma-separated: m1,m2,m3)
forecast_spend() {
    local historical_data="$1"

    # Parse historical data into arrays
    local -a spends=()
    local count=0

    while IFS='|' read -r month spend; do
        # Skip header and empty lines
        [[ -z "$month" || "$month" == " month" || "$month" == "month" ]] && continue
        [[ "$month" =~ ^-+$ ]] && continue
        [[ "$month" =~ ^\( ]] && continue

        spend=$(echo "$spend" | xargs)
        [[ -z "$spend" || "$spend" == "0" ]] && continue

        spends+=("$spend")
        ((count++))
    done <<<"$historical_data"

    # Need at least 2 data points for regression
    if [[ $count -lt 2 ]]; then
        echo "N/A,N/A,N/A"
        return
    fi

    # Reverse to get chronological order (oldest first)
    local -a reversed=()
    for ((i = ${#spends[@]} - 1; i >= 0; i--)); do
        reversed+=("${spends[$i]}")
    done

    # Calculate linear regression: y = mx + b
    # Using least squares: m = (n*sum(xy) - sum(x)*sum(y)) / (n*sum(x^2) - sum(x)^2)
    local n=$count
    local sum_x=0 sum_y=0 sum_xy=0 sum_x2=0

    for ((i = 0; i < n; i++)); do
        local x=$((i + 1)) # 1-indexed months
        local y=${reversed[$i]}
        sum_x=$((sum_x + x))
        sum_y=$(echo "$sum_y + $y" | bc)
        sum_xy=$(echo "$sum_xy + ($x * $y)" | bc)
        sum_x2=$((sum_x2 + x * x))
    done

    # Calculate slope (m) and intercept (b)
    local denominator=$((n * sum_x2 - sum_x * sum_x))
    if [[ "$denominator" -eq 0 ]]; then
        echo "N/A,N/A,N/A"
        return
    fi

    local slope
    slope=$(echo "scale=6; ($n * $sum_xy - $sum_x * $sum_y) / $denominator" | bc)

    local intercept
    intercept=$(echo "scale=6; ($sum_y - $slope * $sum_x) / $n" | bc)

    # Predict next 3 months
    local last_x=$((n + 1))
    local m1 m2 m3

    m1=$(echo "scale=4; $slope * $last_x + $intercept" | bc)
    m2=$(echo "scale=4; $slope * ($last_x + 1) + $intercept" | bc)
    m3=$(echo "scale=4; $slope * ($last_x + 2) + $intercept" | bc)

    # Ensure non-negative predictions
    m1=$(echo "if ($m1 < 0) 0 else $m1" | bc)
    m2=$(echo "if ($m2 < 0) 0 else $m2" | bc)
    m3=$(echo "if ($m3 < 0) 0 else $m3" | bc)

    echo "${m1},${m2},${m3}"
}

# Calculate burn rate (daily spend average) and days until budget exhaustion
# Input: current_spend, month_start, total_budget
# Output: daily_burn_rate|days_remaining|exhaustion_date
calculate_burn_rate() {
    local current_spend="$1"
    local month_start="$2"
    local total_budget="$3"

    if [[ -z "$current_spend" || "$current_spend" == "0" ]]; then
        echo "0|N/A|N/A"
        return
    fi

    # Calculate days elapsed in current month
    local today
    today=$(date +%Y-%m-%d)
    local days_elapsed
    # Cross-platform date calculation (GNU date uses -d, BSD date uses -v)
    local today_epoch month_start_epoch
    today_epoch=$(date -d "$today" +%s 2>/dev/null || date -j -f "%Y-%m-%d" "$today" +%s 2>/dev/null || echo "0")
    month_start_epoch=$(date -d "$month_start" +%s 2>/dev/null || date -j -f "%Y-%m-%d" "$month_start" +%s 2>/dev/null || echo "0")
    days_elapsed=$(((today_epoch - month_start_epoch) / 86400 + 1))

    # Ensure at least 1 day to avoid division by zero
    [[ "$days_elapsed" -lt 1 ]] && days_elapsed=1

    # Calculate daily burn rate
    local daily_rate
    daily_rate=$(echo "scale=4; $current_spend / $days_elapsed" | bc)

    # Calculate days remaining if budget exists
    local days_remaining="N/A"
    local exhaustion_date="N/A"

    if [[ -n "$total_budget" && "$total_budget" != "0" && "$total_budget" != "0.00" ]]; then
        local remaining
        remaining=$(echo "$total_budget - $current_spend" | bc)

        if [[ $(echo "$remaining > 0" | bc) -eq 1 && "$daily_rate" != "0" ]]; then
            days_remaining=$(echo "scale=0; $remaining / $daily_rate" | bc)

            # Calculate exhaustion date (cross-platform)
            exhaustion_date=$(date -d "+$days_remaining days" +%Y-%m-%d 2>/dev/null || date -v+"${days_remaining}"d +%Y-%m-%d 2>/dev/null || echo "N/A")
        elif [[ $(echo "$remaining <= 0" | bc) -eq 1 ]]; then
            days_remaining="0"
            exhaustion_date="EXHAUSTED"
        fi
    fi

    echo "${daily_rate}|${days_remaining}|${exhaustion_date}"
}

# Check if forecasted spend exceeds budget threshold
# Input: forecast_3mo_total, total_budget, threshold_percent
# Output: pipe-delimited risk_level|budget_percent|threshold_exceeded
check_budget_risk() {
    local forecast_total="$1"
    local total_budget="$2"
    local threshold="${3:-80}"

    if [[ -z "$total_budget" || "$total_budget" == "0" || "$total_budget" == "0.00" ]]; then
        echo "unknown|N/A|false"
        return
    fi

    local budget_percent
    budget_percent=$(echo "scale=2; ($forecast_total / $total_budget) * 100" | bc)

    local risk_level="low"
    local threshold_exceeded="false"

    if [[ $(echo "$budget_percent >= $threshold" | bc) -eq 1 ]]; then
        risk_level="high"
        threshold_exceeded="true"
    elif [[ $(echo "$budget_percent >= 50" | bc) -eq 1 ]]; then
        risk_level="medium"
    fi

    echo "${risk_level}|${budget_percent}|${threshold_exceeded}"
}
