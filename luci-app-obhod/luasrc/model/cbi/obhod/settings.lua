-- CBI Model for Obhod Settings
local _ = luci.i18n.translate
local m, s, o

m = Map("obhod", _("Obhod VPN"), _("Smart selective routing for OpenWrt. Architecture: Go + Sing-box + nftables."))

s = m:section(NamedSection, "settings", "global", _("Global Settings"))
s.anonymous = true
s.addremove = false

o = s:option(Flag, "enabled", _("Enabled"), _("Enable or disable the Obhod service."))
o.rmempty = false

o = s:option(Value, "fwmark", _("Firewall Mark"), _("Mark for VPN traffic (1-255). Default is 255."))
o.datatype = "range(1, 255)"
o.placeholder = "255"

o = s:option(Value, "dns_port", _("DNS Port"), _("Local port for sing-box DNS inbound."))
o.datatype = "port"
o.placeholder = "15353"

o = s:option(Value, "tproxy_port", _("TProxy Port"), _("Local port for sing-box TProxy inbound."))
o.datatype = "port"
o.placeholder = "11080"

o = s:option(ListValue, "log_level", _("Log Level"))
o:value("debug", _("Debug"))
o:value("info", _("Info"))
o:value("warn", _("Warning"))
o:value("error", _("Error"))
o.default = "info"

return m
