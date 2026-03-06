# AI Control Plane - Makefile Color Definitions
#
# Purpose: Define terminal color codes for consistent output formatting
# Responsibilities:
#   - Define ANSI color codes
#   - Handle non-terminal environments (NO_COLOR support)
#
# Non-scope:
#   - Does not print output directly
#   - Does not detect terminal capabilities beyond MAKE_TERMOUT

# Only define colors when running in a terminal
ifeq ($(MAKE_TERMOUT),)
COLOR_RESET :=
COLOR_BOLD :=
COLOR_GREEN :=
COLOR_YELLOW :=
COLOR_RED :=
COLOR_CYAN :=
else
ESC := $(shell printf '\033')
COLOR_RESET := $(ESC)[0m
COLOR_BOLD := $(ESC)[1m
COLOR_GREEN := $(ESC)[32m
COLOR_YELLOW := $(ESC)[33m
COLOR_RED := $(ESC)[31m
COLOR_CYAN := $(ESC)[36m
endif
