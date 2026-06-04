# Global temporary files registry for BusyBox ash compatibility
set -u
TEMP_FILES=""

register_temp_file() {
    local file="$1"
    [ -n "$file" ] && TEMP_FILES="$TEMP_FILES $file"
}

cleanup_temp_files() {
    local file
    for file in $TEMP_FILES; do
        [ -f "$file" ] && rm -f "$file"
    done
}

trap cleanup_temp_files EXIT INT TERM

validate_interface_name() {
    local ifname="$1"
    echo "$ifname" | grep -qE '^[a-zA-Z0-9][a-zA-Z0-9_\-\.]{0,14}$'
}

# Check if string is valid IPv4
is_ipv4() {
    local ip="$1"
    # Basic check using expr or grep for ash compatibility
    echo "$ip" | grep -qE '^((25[0-5]|(2[0-4]|1[0-9]|[1-9]|)[0-9])\.){3}(25[0-5]|(2[0-4]|1[0-9]|[1-9]|)[0-9])$'
}

# Check if string is valid IPv4 with CIDR mask
is_ipv4_cidr() {
    local ip="$1"
    echo "$ip" | grep -qE '^((25[0-5]|(2[0-4]|1[0-9]|[1-9]|)[0-9])\.){3}(25[0-5]|(2[0-4]|1[0-9]|[1-9]|)[0-9])/(3[0-2]|[12][0-9]|[0-9])$'
}

is_ipv4_ip_or_ipv4_cidr() {
    is_ipv4 "$1" || is_ipv4_cidr "$1"
}

is_domain() {
    local str="$1"
    # Improved RFC-compliant domain validation
    echo "$str" | grep -qE '^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$'
}

is_domain_suffix() {
    local str="$1"
    local normalized="${str#.}"

    is_domain "$normalized"
}

# Checks if the given string is a valid base64-encoded sequence
is_base64() {
    local str="$1"

    if echo "$str" | base64 -d > /dev/null 2>&1; then
        return 0
    fi
    return 1
}

# Checks if the given string looks like a Shadowsocks userinfo
is_shadowsocks_userinfo_format() {
    local str="$1"
    echo "$str" | grep -qE '^[^:]+:[^:]+(:[^:]+)?$'
}

# Compares the current package version with the required minimum
is_min_package_version() {
    local current="$1"
    local required="$2"

    local lowest
    lowest="$(printf '%s\n' "$current" "$required" | sort -V | head -n1)"

    [ "$lowest" = "$required" ]
}

# Checks if the given file exists
file_exists() {
    local filepath="$1"

    if [ -f "$filepath" ]; then
        return 0
    else
        return 1
    fi
}

# Checks if a service script exists in /etc/init.d
service_exists() {
    local service="$1"

    if [ -x "/etc/init.d/$service" ]; then
        return 0
    else
        return 1
    fi
}

# Returns the inbound tag name by appending the postfix to the given section
get_inbound_tag_by_section() {
    local section="$1"
    local postfix="in"

    echo "$section-$postfix"
}

# Returns the outbound tag name by appending the postfix to the given section
get_outbound_tag_by_section() {
    local section="$1"
    local postfix="out"

    echo "$section-$postfix"
}

# Constructs and returns a domain resolver tag by appending a fixed postfix to the given section
get_domain_resolver_tag() {
    local section="$1"
    local postfix="domain-resolver"

    echo "$section-$postfix"
}

# Converts a comma-separated string into a JSON array string
comma_string_to_json_array() {
    local input="$1"

    if [ -z "$input" ]; then
        echo "[]"
        return
    fi

    local replaced="${input//,/\",\"}"

    echo "[\"$replaced\"]"
}

# Decodes a URL-encoded string
url_decode() {
    local encoded="$1"
    # Note: we do NOT convert '+' to space here. In URIs, '+' as space
    # is only valid in application/x-www-form-urlencoded (query strings).
    # Converting '+' in the full URL would corrupt Base64 in SS:// userinfo.
    printf '%b' "$(echo "$encoded" | sed 's/%/\\x/g')"
}

# Returns the scheme (protocol) part of a URL
url_get_scheme() {
    local url="$1"
    echo "${url%%://*}"
}

# Extracts the userinfo (username[:password]) part from a URL
url_get_userinfo() {
    local url="$1"
    echo "$url" | sed -n -e 's#^[^:/?]*://##' -e '/@/!d' -e 's/@.*//p'
}

# Extracts the host part from a URL
url_get_host() {
    local url="$1"

    url="${url#*://}"
    url="${url#*@}"
    url="${url%%[/?#]*}"

    echo "${url%%:*}"
}

# Extracts the port number from a URL
url_get_port() {
    local url="$1"

    url="${url#*://}"
    url="${url#*@}"
    url="${url%%[/?#]*}"

    case "$url" in
    *:*) echo "${url#*:}" ;;
    *) echo "" ;;
    esac
}

# Extracts the path from a URL (without query or fragment; returns "/" if empty)
url_get_path() {
    local url="$1"
    echo "$url" | sed -n -e 's#^[^:/?]*://##' -e 's#^[^/]*##' -e 's#\([^?]*\).*#\1#p'
}

# Extracts the value of a specific query parameter from a URL
url_get_query_param() {
    local url="$1"
    local param="$2"

    local raw
    raw=$(echo "$url" | sed -n "s/.*[?&]$param=\([^&?#]*\).*/\1/p")

    [ -z "$raw" ] && echo "" && return

    echo "$raw"
}

# Extracts the basename (filename without extension) from a URL
url_get_basename() {
    local url="$1"

    local filename="${url##*/}"
    local basename="${filename%%.*}"

    echo "$basename"
}

# Extracts and returns the file extension from the given URL
url_get_file_extension() {
    local url="$1"

    local basename="${url##*/}"
    case "$basename" in
    *.*) echo "${basename##*.}" ;;
    *) echo "" ;;
    esac
}

# Remove url fragment (everything after the first '#')
url_strip_fragment() {
    local url="$1"

    echo "${url%%#*}"
}

# Decodes and returns a base64-encoded string
base64_decode() {
    local str="$1"
    local decoded_url

    decoded_url="$(echo "$str" | base64 -d 2> /dev/null)"

    echo "$decoded_url"
}

# Generates a unique 16-character ID using cryptographically secure random source
gen_id() {
    if [ -r /dev/urandom ]; then
        # Use /dev/urandom for better entropy
        od -An -N8 -tx8 /dev/urandom | tr -d ' \n' | md5sum | cut -c1-16
    else
        # Fallback to original method but with better randomness
        printf '%s%s%s' "$(date +%s)" "$RANDOM" "$(hostname)" | md5sum | cut -c1-16
    fi
}

# Adds a missing UCI option with the given value if it does not exist
migration_add_new_option() {
    local package="$1"
    local section="$2"
    local option="$3"
    local value="$4"

    local current
    current="$(uci -q get "$package.$section.$option")"
    if [ -z "$current" ]; then
        obhod_log "Adding missing option '$option' with value '$value'"
        uci set "$package.$section.$option=$value"
        uci commit "$package"
        return 0
    else
        return 1
    fi
}

escape_sed() {
    echo "$1" | sed 's/[&/\]/\\&/g'
}

# Migrates a configuration key in an OpenWrt config file from old_key_name to new_key_name
migration_rename_config_key() {
    local config="$1"
    local key_type="$2"
    local old_key_name="$3"
    local new_key_name="$4"

    local esc_key_type esc_old esc_new
    esc_key_type=$(escape_sed "$key_type")
    esc_old=$(escape_sed "$old_key_name")
    esc_new=$(escape_sed "$new_key_name")

    if grep -q "$key_type $old_key_name" "$config"; then
        obhod_log "Deprecated $key_type found: $old_key_name migrating to $new_key_name"
        sed -i "s/$esc_key_type $esc_old/$esc_key_type $esc_new/g" "$config"
    fi
}

# Download URL to file with improved error handling
download_to_file() {
    local url="$1"
    local filepath="$2"
    local http_proxy_address="$3"
    local retries="${4:-3}"
    local wait="${5:-2}"

    # Determine downloader
    local use_curl=0
    local use_wget=0

    if command -v curl >/dev/null 2>&1; then
        use_curl=1
    elif command -v wget >/dev/null 2>&1; then
        use_wget=1
    else
        obhod_log "Neither curl nor wget command found" "error"
        return 1
    fi

    for attempt in $(seq 1 "$retries"); do
        if [ "$use_curl" -eq 1 ]; then
            if [ -n "$http_proxy_address" ]; then
                curl -s -L -x "http://$http_proxy_address" -o "$filepath" "$url"
                curl_result=$?
            else
                curl -s -L -o "$filepath" "$url"
                curl_result=$?
            fi
            
            if [ $curl_result -eq 0 ] && [ -s "$filepath" ]; then
                obhod_log "Successfully downloaded $url using curl (attempt $attempt)"
                return 0
            fi
            obhod_log "Download attempt $attempt/$retries failed for $url via curl (curl exit code: $curl_result)" "warn"
        else
            if [ -n "$http_proxy_address" ]; then
                http_proxy="http://$http_proxy_address" https_proxy="http://$http_proxy_address" wget -q -O "$filepath" "$url" 2>/dev/null
                wget_result=$?
            else
                wget -q -O "$filepath" "$url" 2>/dev/null
                wget_result=$?
            fi

            if [ $wget_result -eq 0 ] && [ -s "$filepath" ]; then
                obhod_log "Successfully downloaded $url using wget (attempt $attempt)"
                return 0
            fi
            obhod_log "Download attempt $attempt/$retries failed for $url via wget (wget exit code: $wget_result)" "warn"
        fi

        # Clean up partial download
        [ -f "$filepath" ] && rm -f "$filepath"
        
        [ "$attempt" -lt "$retries" ] && sleep "$wait"
    done

    obhod_log "Failed to download $url after $retries attempts" "error"
    return 1
}

# # Converts Windows-style line endings (CRLF) to Unix-style (LF)
convert_crlf_to_lf() {
    local filepath="$1"
    [ ! -f "$filepath" ] && return 1

    local tmpfile
    tmpfile=$(mktemp) || return 1
    register_temp_file "$tmpfile"
    tr -d '\r' < "$filepath" > "$tmpfile" && mv "$tmpfile" "$filepath"
}

#######################################
# Parses a whitespace-separated string, validates items as either domains
# or IPv4 addresses/subnets, and returns a comma-separated string of valid items.
# Arguments:
#   $1 - Input string (space-separated list of items)
#   $2 - Type of validation ("domains" or "subnets")
# Outputs:
#   Comma-separated string of valid domains or subnets
#######################################
parse_domain_or_subnet_string_to_commas_string() {
    local string="$1"
    local type="$2"

    echo "$string" | sed 's/\/\/.*//' | tr ', ' '\n' | grep -v '^$' | parse_domain_or_subnet_file_to_comma_string "-" "$type"
}

#######################################
# Parses a file line by line, validates entries as either domains or subnets,
# and returns a single comma-separated string of valid items.
# Arguments:
#   $1 - Path to the input file
#   $2 - Type of validation ("domains" or "subnets")
# Outputs:
#   Comma-separated string of valid domains or subnets
#######################################
parse_domain_or_subnet_file_to_comma_string() {
    local filepath="$1"
    local type="$2"

    local target_file="$filepath"
    if [ "$target_file" = "-" ]; then
        target_file=""
    fi

    if [ -n "$target_file" ] && [ ! -f "$target_file" ]; then
        return 1
    fi

    if [ "$type" = "domains" ]; then
        cat ${target_file:+"$target_file"} | grep -v '^[[:space:]]*$' | awk '{$1=$1}1' | grep -E '^[a-zA-Z0-9-]{1,63}(\.[a-zA-Z0-9-]{1,63})*$' | paste -sd, -
    elif [ "$type" = "subnets" ]; then
        cat ${target_file:+"$target_file"} | grep -v '^[[:space:]]*$' | awk '{$1=$1}1' | grep -E '^([0-9]{1,3}\.){3}[0-9]{1,3}(/[0-9]{1,2})?$' | paste -sd, -
    else
        obhod_log "Unknown type: $type" "error"
        return 1
    fi
}