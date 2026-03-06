#!/usr/bin/env bash
# demo/scripts/lib/chargeback_io.sh - IO and utility library for chargeback reports
#
# Purpose: Handles file archival, timestamp generation, and calendar boundary
#          calculations for chargeback reporting.
#
# Responsibilities:
#   - Calculate month boundaries (start/end) for current and previous months
#   - Archive report content to specified directory with standard naming
#   - Generate ISO timestamps
#
# Non-scope:
#   - Does NOT interact with the database
#   - Does NOT handle report rendering
#
# Invariants:
#   - Uses GNU/BSD portable date logic for month calculations
#   - Relies on PROJECT_ROOT being set for archival
#

if [[ -n "${ACP_CHARGEBACK_IO_LOADED:-}" ]]; then
    return 0
fi
readonly ACP_CHARGEBACK_IO_LOADED=1

# Get ISO timestamp
get_timestamp() {
    date -u +"%Y-%m-%dT%H:%M:%SZ"
}

# Calculate month boundaries
# Sets: MONTH_START, MONTH_END, PREV_MONTH_START, PREV_MONTH_END
calculate_month_boundaries() {
    local target="$1"

    # Current month start and end
    MONTH_START="${target}-01"
    # Get last day of month (portable across GNU and BSD date)
    if date -d "$MONTH_START +1 month -1 day" +%Y-%m-%d >/dev/null 2>&1; then
        # GNU date
        MONTH_END=$(date -d "$MONTH_START +1 month -1 day" +%Y-%m-%d)
        PREV_MONTH_START=$(date -d "$MONTH_START -1 month" +%Y-%m)
        PREV_MONTH_START="${PREV_MONTH_START}-01"
        PREV_MONTH_END=$(date -d "$PREV_MONTH_START +1 month -1 day" +%Y-%m-%d)
    else
        # BSD date (macOS)
        MONTH_END=$(date -v+1m -v-1d -j -f "%Y-%m-%d" "$MONTH_START" +%Y-%m-%d 2>/dev/null || echo "${target}-28")
        local prev_month
        prev_month=$(date -v-1m -j -f "%Y-%m-%d" "$MONTH_START" +%Y-%m 2>/dev/null || echo "")
        if [[ -n "$prev_month" ]]; then
            PREV_MONTH_START="${prev_month}-01"
            PREV_MONTH_END=$(date -v+1m -v-1d -j -f "%Y-%m-%d" "$PREV_MONTH_START" +%Y-%m-%d 2>/dev/null || echo "${prev_month}-28")
        else
            # Fallback: parse manually
            local year month
            year=$(echo "$target" | cut -d- -f1)
            month=$(echo "$target" | cut -d- -f2)
            local prev_month_num=$((10#$month - 1))
            local prev_year=$year
            if [[ $prev_month_num -eq 0 ]]; then
                prev_month_num=12
                prev_year=$((year - 1))
            fi
            printf -v PREV_MONTH_START "%d-%02d-01" "$prev_year" "$prev_month_num"
            # Approximate end date
            if [[ $prev_month_num -eq 2 ]]; then
                printf -v PREV_MONTH_END "%d-%02d-28" "$prev_year" "$prev_month_num"
            elif [[ $prev_month_num -eq 4 || $prev_month_num -eq 6 || $prev_month_num -eq 9 || $prev_month_num -eq 11 ]]; then
                printf -v PREV_MONTH_END "%d-%02d-30" "$prev_year" "$prev_month_num"
            else
                printf -v PREV_MONTH_END "%d-%02d-31" "$prev_year" "$prev_month_num"
            fi
        fi
    fi
}

# Archive report to file
archive_report() {
    local report_month="$1"
    local format="$2"
    local content="$3"

    # Create archive directory
    # shellcheck disable=SC2154
    local archive_path="$PROJECT_ROOT/$ARCHIVE_DIR/$report_month"
    mkdir -p "$archive_path"

    local filename="chargeback-report-${report_month}.${format}"
    local filepath="$archive_path/$filename"

    echo "$content" >"$filepath"
    echo "$filepath"
}
