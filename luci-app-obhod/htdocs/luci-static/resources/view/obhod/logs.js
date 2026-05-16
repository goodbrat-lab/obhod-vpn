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
    const container = E('div', { 'id': 'obhod_log_view_container' }, [
        E('div', { 'class': 'cbi-value', 'style': 'margin-bottom: 10px; display: flex; gap: 10px; align-items: center;' }, [
            E('label', { 'class': 'cbi-value-title', 'style': 'margin-bottom: 0' }, _('Filter')),
            E('div', { 'class': 'cbi-value-field', 'style': 'padding-bottom: 0' }, [
                E('select', { 
                    'id': 'log_level_filter',
                    'class': 'cbi-input-select',
                    'change': (ev) => {
                        this.updateLogs();
                    }
                }, [
                    E('option', { 'value': '' }, _('All Levels')),
                    E('option', { 'value': 'DEBUG' }, 'DEBUG'),
                    E('option', { 'value': 'INFO' }, 'INFO'),
                    E('option', { 'value': 'WARN' }, 'WARN'),
                    E('option', { 'value': 'ERROR' }, 'ERROR'),
                    E('option', { 'value': 'FATAL' }, 'FATAL')
                ]),
                E('input', {
                    'id': 'log_search_filter',
                    'class': 'cbi-input-text',
                    'placeholder': _('Search...'),
                    'style': 'margin-left: 10px',
                    'keyup': (ev) => {
                        this.updateLogs();
                    }
                }),
                E('button', {
                    'class': 'cbi-button cbi-button-action',
                    'style': 'margin-left: 10px',
                    'click': () => this.downloadLogs()
                }, _('Download Logs'))
            ])
        ]),
        E('div', {
            'id': 'obhod_log_text',
            'style': 'width:100%; height:500px; font-family:monospace; font-size:12px; border:1px solid #ccc; padding:10px; background:#1e1e1e; color:#d4d4d4; white-space: pre; overflow: auto; border-radius: 4px;'
        }, [ _('Loading logs...') ])
    ]);

    // Initial load
    setTimeout(() => this.updateLogs(), 100);

    return container;
  };

  o.updateLogs = function() {
      const levelFilter = document.getElementById('log_level_filter')?.value || '';
      const searchFilter = document.getElementById('log_search_filter')?.value.toLowerCase() || '';
      
      fs.exec('logread', ['-e', 'obho[ud]']).then(res => {
          let lines = (res.stdout || '').split('\n').filter(l => l.length > 0);
          
          if (levelFilter) {
              lines = lines.filter(line => line.includes('[' + levelFilter + ']'));
          }
          
          if (searchFilter) {
              lines = lines.filter(line => line.toLowerCase().includes(searchFilter));
          }

          const logContainer = document.getElementById('obhod_log_text');
          if (!logContainer) return;

          const shouldScroll = logContainer.scrollTop + logContainer.clientHeight >= logContainer.scrollHeight - 20;

          // Clear and render colored spans
          logContainer.innerHTML = '';
          if (lines.length === 0) {
              logContainer.textContent = _('No logs found matching criteria');
              return;
          }

          lines.slice(-500).forEach(line => {
              let color = '#d4d4d4'; // Default
              if (line.includes('[ERROR]') || line.includes('[FATAL]')) color = '#f44336';
              else if (line.includes('[WARN]')) color = '#ff9800';
              else if (line.includes('[INFO]')) color = '#4caf50';
              else if (line.includes('[DEBUG]')) color = '#2196f3';

              const span = E('span', { 'style': `color: ${color}; display: block;` }, line);
              logContainer.appendChild(span);
          });

          if (shouldScroll) {
              logContainer.scrollTop = logContainer.scrollHeight;
          }
      });
  };

  o.downloadLogs = function() {
      fs.exec('logread', ['-e', 'obho[ud]']).then(res => {
          const blob = new Blob([res.stdout || ''], { type: 'text/plain' });
          const url = window.URL.createObjectURL(blob);
          const a = document.createElement('a');
          a.href = url;
          a.download = `obhod_logs_${new Date().toISOString().replace(/[:.]/g, '-')}.txt`;
          document.body.appendChild(a);
          a.click();
          window.URL.revokeObjectURL(url);
          document.body.removeChild(a);
      });
  };

  poll.add(() => {
    o.updateLogs();
  }, 3);
}

const EntryPoint = {
  createLogsContent,
};

return baseclass.extend(EntryPoint);
