# Create an nftables table in the inet family
nft_create_table() {
    local name="$1"

    nft add table inet "$name"
}

# Create a set within a table for storing IPv4 addresses
nft_create_ipv4_set() {
    local table="$1"
    local name="$2"

    nft add set inet "$table" "$name" '{ type ipv4_addr; flags interval; auto-merge; }'
}

nft_create_ifname_set() {
    local table="$1"
    local name="$2"

    nft add set inet "$table" "$name" '{ type ifname; flags interval; }'
}

# Add one or more elements to a set
nft_add_set_elements() {
    local table="$1"
    local set="$2"
    local elements="$3"

    nft add element inet "$table" "$set" "{ $elements }"
}

nft_add_set_elements_from_file_chunked() {
    local filepath="$1"
    local nft_table_name="$2"
    local nft_set_name="$3"

    [ ! -f "$filepath" ] && return 1

    # Check if there are any elements first to avoid empty command failures
    if ! grep -q '[^[:space:]]' "$filepath"; then
        return 0
    fi

    # Stream elements to nft using streaming awk pipeline
    grep -v '^[[:space:]]*$' "$filepath" | awk '{$1=$1}1' | grep -E '^([0-9]{1,3}\.){3}[0-9]{1,3}(/[0-9]{1,2})?$' | awk -v table="$nft_table_name" -v set="$nft_set_name" '
    {
        if (!seen[$0]++) {
            items[count++] = $0
            if (count == 1000) {
                flush()
            }
        }
    }
    function flush() {
        if (count > 0) {
            printf "add element inet %s %s { ", table, set
            for (i = 0; i < count; i++) {
                printf "%s", items[i]
                if (i < count - 1) printf ", "
            }
            print " }"
            delete items
            count = 0
        }
    }
    END {
        flush()
    }' | nft -f -
}