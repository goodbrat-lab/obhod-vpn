COLOR_CYAN="\033[0;36m"
COLOR_GREEN="\033[0;32m"
COLOR_YELLOW="\033[0;33m"
COLOR_RED="\033[0;31m"
COLOR_RESET="\033[0m"

# Default values if not set by caller
OBHOD_LOG_COMPONENT="${OBHOD_LOG_COMPONENT:-core}"
OBHOD_LOG_CONTEXT="${OBHOD_LOG_CONTEXT:-main}"

_get_log_level_weight() {
    case "$(echo "$1" | tr '[:upper:]' '[:lower:]')" in
        debug) echo 0 ;;
        info)  echo 1 ;;
        warn)  echo 2 ;;
        error) echo 3 ;;
        fatal) echo 4 ;;
        *)     echo 1 ;; # Default to info
    esac
}

_get_current_log_level() {
    # Cache log level to avoid repeated uci calls
    if [ -z "$_CACHED_LOG_LEVEL" ]; then
        _CACHED_LOG_LEVEL=$(uci -q get obhod.settings.log_level || echo "info")
    fi
    echo "$_CACHED_LOG_LEVEL"
}

obhod_log() {
    local message="$1"
    local level="${2:-info}"
    local component="${3:-$OBHOD_LOG_COMPONENT}"
    local context="${4:-$OBHOD_LOG_CONTEXT}"

    local current_level=$(_get_current_log_level)
    local weight_msg=$(_get_log_level_weight "$level")
    local weight_curr=$(_get_log_level_weight "$current_level")

    [ "$weight_msg" -lt "$weight_curr" ] && return 0

    local timestamp
    timestamp=$(date +"%Y-%m-%d %H:%M:%S")
    
    local level_upper
    level_upper=$(echo "$level" | tr 'a-z' 'A-Z')

    # Standardized format: [timestamp] [LEVEL] [component] [context] message
    local formatted_msg="[$timestamp] [$level_upper] [$component] [$context] $message"

    # Log to syslog (logger handles its own levels, but we embed the full message)
    local priority="daemon.info"
    case "$level" in
        debug) priority="daemon.debug" ;;
        info)  priority="daemon.info" ;;
        warn)  priority="daemon.warning" ;;
        error|fatal) priority="daemon.err" ;;
    esac

    logger -p "$priority" -t "obhod" "$formatted_msg"

    # If running interactively, also print to stderr with colors
    if [ -t 2 ]; then
        local color=$COLOR_RESET
        case "$level" in
            debug) color=$COLOR_CYAN ;;
            info)  color=$COLOR_GREEN ;;
            warn)  color=$COLOR_YELLOW ;;
            error|fatal) color=$COLOR_RED ;;
        esac
        echo -e "${color}$formatted_msg${COLOR_RESET}" >&2
    fi
}

# Compatibility aliases
nolog() {
    # nolog was used for interactive output only. Now log handles it if -t 2.
    # We keep it as a no-op or just redirect to log debug if needed.
    :
}

echolog() {
    obhod_log "$1" "$2"
}

