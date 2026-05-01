-- Controller for Obhod
module("luci.controller.obhod", package.seeall)

function index()
    if not nixio.fs.access("/etc/config/obhod") then
        return
    end

    -- Главное меню в Services
    local page
    page = entry({"admin", "services", "obhod"}, cbi("obhod/settings"), _("Obhod VPN"), 60)
    page.dependent = true

    -- Дополнительные вкладки (классический стиль)
    entry({"admin", "services", "obhod", "tunnels"}, cbi("obhod/tunnels"), _("Tunnels"), 10).leaf = true
    entry({"admin", "services", "obhod", "rules"}, cbi("obhod/rules"), _("Routing Rules"), 20).leaf = true
    
    -- API
    entry({"admin", "services", "obhod", "status"}, call("action_status")).leaf = true
end

function action_status()
    local api = require "obhod_api"
    local res = api.get_status()
    luci.http.prepare_content("application/json")
    luci.http.write_json(res or {running = false})
end
