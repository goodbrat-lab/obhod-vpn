#!/bin/sh
# Install sing-box and fix DNS compatibility issue for Obhod

echo "📦 Installing sing-box..."

# Install package manager if not present
if ! command -v opkg >/dev/null 2>&1; then
    echo "❌ opkg not found. This script requires OpenWrt with opkg."
    exit 1
fi

# Update package list
opkg update >/dev/null 2>&1

# Install sing-box
if ! opkg install sing-box >/dev/null 2>&1; then
    echo "❌ Failed to install sing-box"
    echo "Checking available packages:"
    opkg list | grep sing
    exit 1
fi

# Verify installation
if ! command -v sing-box >/dev/null 2>&1; then
    echo "❌ sing-box not found after installation"
    exit 1
fi

echo "✅ sing-box installed: $(sing-box version | head -n1)"

# Create working configuration without DNS inbound
cat > /tmp/working_config.json << 'EOF'
{
  "log":{"level":"warn"},
  "inbounds":[
    {
      "type":"tproxy",
      "tag":"tproxy-in",
      "listen":"127.0.0.1",
      "listen_port":1602,
      "sniff":true
    }
  ],
  "outbounds":[
    {"type":"direct","tag":"direct-out"}
  ],
  "dns":{
    "servers":[
      {"tag":"dns-server","server":"1.1.1.1","server_port":53}
    ]
  },
  "route":{
    "rules":[
      {"inbound":"tproxy-in","outbound":"direct-out"}
    ],
    "final":"direct-out"
  }
}
EOF

# Test configuration
if sing-box check -c /tmp/working_config.json >/dev/null 2>&1; then
    echo "✅ Basic configuration valid"
    cp /tmp/working_config.json /etc/sing-box/config.json
else
    echo "❌ Basic configuration invalid. Testing minimal config..."
    
    # Try even simpler config
    cat > /tmp/minimal_config.json << 'EOF'
{
  "log":{"level":"warn"},
  "outbounds":[
    {"type":"direct","tag":"direct-out"}
  ]
}
EOF
    
    if sing-box check -c /tmp/minimal_config.json >/dev/null 2>&1; then
        echo "✅ Minimal configuration valid"
        cp /tmp/minimal_config.json /etc/sing-box/config.json
    else
        echo "❌ sing-box configuration invalid - check installation"
        sing-box check -c /tmp/minimal_config.json
        exit 1
    fi
fi

# Start sing-box
echo "🔄 Starting sing-box..."
/etc/init.d/sing-box restart >/dev/null 2>&1
sleep 2

# Start Obhod
echo "🔄 Starting Obhod..."
if [ -f /etc/init.d/obhod ]; then
    chmod +x /etc/init.d/obhod
    /etc/init.d/obhod restart
else
    echo "❌ Obhod init script not found"
fi

# Test if working
sleep 3
if /etc/init.d/sing-box status >/dev/null 2>&1; then
    echo "✅ sing-box is running!"
else
    echo "❌ sing-box failed to start"
fi

# Cleanup
rm -f /tmp/working_config.json /tmp/minimal_config.json

echo "🎉 Installation complete! Check status:"
echo "   /etc/init.d/sing-box status"
echo "   /etc/init.d/obhod status"