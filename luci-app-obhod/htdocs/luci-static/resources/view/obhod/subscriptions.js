"use strict";
"require form";
"require baseclass";
"require uci";

function createSubscriptionContent(section) {
  let o;

  o = section.option(form.Flag, "enabled", _("Enabled"));
  o.rmempty = false;
  o.default = "1";

  o = section.option(form.Value, "name", _("Name"));
  o.placeholder = _("e.g. MyProvider");
  o.rmempty = false;

  o = section.option(form.Value, "url", _("Subscription URL"));
  o.placeholder = "https://example.com/sub/uuid";
  o.rmempty = false;

  o = section.option(form.ListValue, "update_interval", _("Update Interval"));
  o.value("1h", _("Every hour"));
  o.value("12h", _("Every 12 hours"));
  o.value("1d", _("Every day"));
  o.default = "1d";
}

const EntryPoint = {
  createSubscriptionContent,
};

return baseclass.extend(EntryPoint);
