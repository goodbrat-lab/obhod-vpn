module("obhod_api", package.seeall)

local json = require "luci.jsonc"

-- Helper to make requests via UDS using curl
local function make_request(method, endpoint)
    local cmd = string.format("curl -s -X %s --unix-socket /var/run/obhoud.sock http://localhost%s 2>/dev/null", method, endpoint)
    local f = io.popen(cmd)
    if not f then return nil end
    local content = f:read("*a")
    f:close()
    
    if content and content ~= "" then
        local data = json.parse(content)
        return data
    end
    return nil
end

function get_status()
    return make_request("GET", "/api/status") or {
        running = false,
        uptime_seconds = 0,
        memory_usage_mb = 0,
        wan_ready = false,
        dns_healthy = false,
        active_tunnels = {}
    }
end

function get_tunnels()
    return make_request("GET", "/api/tunnels") or {}
end

function get_rules()
    return make_request("GET", "/api/rules") or {}
end

function restart_daemon()
    return make_request("POST", "/api/restart")
end
