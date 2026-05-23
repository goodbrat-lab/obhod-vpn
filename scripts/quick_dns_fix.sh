#!/bin/sh
# Quick fix for Obhod DNS compatible versions
# This bypasses DNS inbound requirement and uses Mixed inbound instead

echo "🔧 Quick DNS compatibility fix..."

# Create a working configuration without DNS inbound
cat > /tmp/quick_fix.json << 'EOF'
{
  "log": {"level": "warn"},
  "inbounds": [
    {
      "type": "tproxy",
      "tag": "tproxy-in",
      "listen": "127.0.0.1",
      "listen_port": 1602,
      "sniff": true
    },
    {
      "type": "mixed",
      "tag": "mixed-in",
      "listen": "127.0.0.42",
      "listen_port": 53
    }
  ],
  "outbounds": [
    {"type": "direct", "tag": "direct-out"}
  ],
  "dns": {
    "servers": [
      {"tag": "dns-server", "server": "1.1.1.1", "server_port": 53}
    ],
    "rules": [
      {"inbound": ["mixed-in"], "server": "dns-server"}
    ]
  },
  "route": {
    "rules": [
      {"inbound": "tproxy-in", "outbound": "direct-out"}
    ],
    "final": "direct-out"
  }
}
EOF

# Test and apply if valid
if sing-box check -c /tmp/quick_fix.json >/dev/null 2>&1; then
    echo "✅ Configuration is valid, applying..."
    cp /tmp/quick_fix.json /etc/sing-box/config.json
    /etc/init.d/sing-box restart >/dev/null 2>&1
    sleep 2
    echo "✅ Fix applied! Please restart Obhod manually:"
    echo "   /etc/init.d/obhod restart"
else
    echo "❌ Configuration still invalid"
    echo "Please check sing-box version: sing-box version"
    echo "Minimum required: sing-box >= 1.12.0 with DNS support"
fi

# Cleanup
rm -f /tmp/quick_fix.json