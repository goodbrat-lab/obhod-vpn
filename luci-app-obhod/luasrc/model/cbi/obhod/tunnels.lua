-- CBI Model for Obhod Tunnels
local _ = luci.i18n.translate
local m, s, o

m = Map("obhod", _("Obhod - VPN Tunnels"))

s = m:section(TypedSection, "tunnel", _("VPN Tunnels"))
s.addremove = true
s.anonymous = false
s.template = "cbi/tblsection"

o = s:option(ListValue, "type", _("Type"))
o:value("vless", "VLESS")
o:value("wireguard", "WireGuard")

o = s:option(Value, "server", _("Server Address"))
o.rmempty = false

o = s:option(Value, "port", _("Port"))
o.datatype = "port"

o = s:option(Value, "uuid", _("UUID / Key"))
o.password = true
o:depends("type", "vless")

return m
