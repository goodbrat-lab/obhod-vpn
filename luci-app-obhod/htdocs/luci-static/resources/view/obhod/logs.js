"use strict";
"require baseclass";
"require form";
"require ui";
"require fs";
"require poll";

function createLogsContent(section) {
  let o;

  o = section.option(form.DummyValue, "_log_view");
  o.rawhtml = true;
  o.cfgvalue = function(section_id) {
    const container = E('div', { 'id': 'obhod_log_view' }, [
        E('div', { 'class': 'cbi-value', 'style': 'margin-bottom: 10px' }, [
            E('label', { 'class': 'cbi-value-title' }, _('Filter by level')),
            E('div', { 'class': 'cbi-value-field' }, [
                E('select', { 
                    'id': 'log_level_filter',
                    'class': 'cbi-input-select',
                    'change': (ev) => {
                        this.updateLogs(ev.target.value);
                    }
                }, [
                    E('option', { 'value': '' }, _('All')),
                    E('option', { 'value': 'DEBUG' }, 'DEBUG'),
                    E('option', { 'value': 'INFO' }, 'INFO'),
                    E('option', { 'value': 'WARN' }, 'WARN'),
                    E('option', { 'value': 'ERROR' }, 'ERROR'),
                    E('option', { 'value': 'FATAL' }, 'FATAL')
                ])
            ])
        ]),
        E('textarea', {
            'id': 'obhod_log_text',
            'style': 'width:100%; height:500px; font-family:monospace; font-size:12px; border:1px solid #ccc; padding:10px; background:#f9f9f9; color:#333; white-space: pre; overflow-x: auto',
            'readonly': 'readonly',
            'wrap': 'off'
        }, [ _('Loading logs...') ])
    ]);

    // Initial load
    setTimeout(() => this.updateLogs(''), 100);

    return container;
  };

  o.updateLogs = function(filter) {
      // Use regex to catch both obhod (bash) and obhoud (go)
      fs.exec('logread', ['-e', 'obho[ud]']).then(res => {
          let lines = (res.stdout || '').split('\n').filter(l => l.length > 0);
          if (filter) {
              lines = lines.filter(line => line.includes('[' + filter + ']'));
          }
          let text = lines.slice(-500).join('\n'); // Last 500 lines for better history
          let textarea = document.getElementById('obhod_log_text');
          if (textarea) {
              const shouldScroll = textarea.scrollTop + textarea.clientHeight >= textarea.scrollHeight - 20;
              textarea.value = text || _('No logs found matching criteria');
              if (shouldScroll) {
                  textarea.scrollTop = textarea.scrollHeight;
              }
          }
      });
  };

  poll.add(() => {
    let filter = document.getElementById('log_level_filter')?.value || '';
    o.updateLogs(filter);
  }, 3);
}

const EntryPoint = {
  createLogsContent,
};

return baseclass.extend(EntryPoint);
