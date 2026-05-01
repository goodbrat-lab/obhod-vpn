-- CBI Model for Obhod Routing Rules
local _ = luci.i18n.translate
local m, s, o

m = Map("obhod", _("Obhod - Routing Rules"))

s = m:section(TypedSection, "routing", _("Routing Rules"))
s.addremove = true
s.anonymous = false

o = s:option(ListValue, "target", _("Target Tunnel"))
local uci_ptr = luci.model.uci.cursor()
uci_ptr:foreach("obhod", "tunnel", function(st)
    o:value(st[".name"], st[".name"])
end)

o = s:option(DynamicList, "domains", _("Domains List"), _("Domains to route through the tunnel."))
o.placeholder = "google.com"

return m
