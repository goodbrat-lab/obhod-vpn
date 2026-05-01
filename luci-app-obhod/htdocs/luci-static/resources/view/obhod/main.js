'use strict';
'require view';
'require form';
'require uci';

return view.extend({
	render: function() {
		return uci.load('obhod').then(function() {
			var m, s, o;

			m = new form.Map('obhod', _('Obhod VPN'), _('Reliable selective VPN routing.'));

			// Section 1: Settings (Typed, safer than Named)
			s = m.section(form.TypedSection, 'global', _('Global Settings'));
			s.anonymous = true;

			o = s.option(form.Flag, 'enabled', _('Enabled'));
			o.rmempty = false;

			o = s.option(form.Value, 'fwmark', _('Firewall Mark'));
			o.datatype = 'range(1, 255)';
			o.placeholder = '255';

			o = s.option(form.Value, 'dns_port', _('DNS Port'));
			o.datatype = 'port';
			o.placeholder = '15353';

			// Section 2: Tunnels
			s = m.section(form.GridSection, 'tunnel', _('VPN Tunnels'));
			s.addremove = true;
			s.anonymous = false;

			o = s.option(form.ListValue, 'type', _('Type'));
			o.value('vless', 'VLESS');
			o.value('wireguard', 'WireGuard');

			o = s.option(form.Value, 'server', _('Server Address'));
			o.rmempty = false;

			o = s.option(form.Value, 'port', _('Port'));
			o.datatype = 'port';

			// Section 3: Routing
			s = m.section(form.TypedSection, 'routing', _('Routing Rules'));
			s.addremove = true;
			s.anonymous = false;

			o = s.option(form.DynamicList, 'domains', _('Domains'));
			o.placeholder = 'google.com';

			return m.render();
		});
	}
});
