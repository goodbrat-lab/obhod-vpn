"use strict";
"require form";
"require baseclass";
"require ui";
"require tools.widgets as widgets";
"require view.obhod.main as main";

function createSectionContent(section) {
  let o = section.option(
    form.ListValue,
    "proxy_config_type",
    _("Proxy Configuration Type"),
  );
  o.value("url", _("URL"));
  o.value("selector", _("Selector"));
  o.value("urltest", _("URLTest"));
  o.value("subscription", _("Subscription")); // NEW TYPE
  o.value("outbound", _("Custom Outbound (JSON)"));
  o.value("interface", _("Interface (VPN)"));
  o.default = "url";
  o.rmempty = false;

  // --- FIELDS FOR TYPE: URL ---
  o = section.option(form.Value, "proxy_string", _("Proxy Link (VLESS/SS/...)"));
  o.placeholder = "vless://uuid@server:port...";
  o.depends("proxy_config_type", "url");

  // --- FIELDS FOR TYPE: SUBSCRIPTION ---
  o = section.option(form.Value, "subscription_url", _("Subscription URL"));
  o.placeholder = "https://example.com/sub/uuid";
  o.depends("proxy_config_type", "subscription");
  // No strict datatype to avoid errors with complex URLs like ?ru=1
  o.rmempty = true;

  o = section.option(form.ListValue, "subscription_update_interval", _("Update Interval"));
  o.value("1h", _("Every hour"));
  o.value("12h", _("Every 12 hours"));
  o.value("1d", _("Every day"));
  o.default = "1d";
  o.depends("proxy_config_type", "subscription");

  // --- FIELDS FOR TYPE: SELECTOR ---
  o = section.option(
    form.DynamicList,
    "selector_proxy_links",
    _("Proxies for Selector"),
  );
  o.depends("proxy_config_type", "selector");

  // --- FIELDS FOR TYPE: URLTEST ---
  o = section.option(
    form.DynamicList,
    "urltest_proxy_links",
    _("Proxies for URLTest"),
  );
  o.depends("proxy_config_type", "urltest");

  // Routing and other fields...
  o = section.option(form.ListValue, "connection_type", _("Connection Type"));
  o.value("proxy", _("Proxy"));
  o.value("direct", _("Direct"));
  o.default = "proxy";

  o = section.option(form.DynamicList, "user_domains", _("Domains"));
  o.placeholder = "google.com";

  o = section.option(form.DynamicList, "community_lists", _("Community Lists"));
  Object.entries(main.DOMAIN_LIST_OPTIONS).forEach(([key, label]) => {
    o.value(key, _(label));
  });
}

const EntryPoint = {
  createSectionContent,
};

return baseclass.extend(EntryPoint);
