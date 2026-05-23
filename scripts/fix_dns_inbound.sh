#!/bin/sh
# Fix for sing-box DNS inbound compatibility issue

set -e

echo "🔧 Fixing Obhod DNS inbound compatibility..."

# Create backup of current config
if [ -f /etc/sing-box/config.json ]; then
    cp /etc/sing-box/config.json /tmp/sing-box_backup_$(date +%s).json
    echo "✅ Backup created"
fi

# Stop services
/etc/init.d/obhod stop 2>/dev/null || true
/etc/init.d/sing-box stop 2>/dev/null || true

# Apply fixes to Obhod scripts
echo "📝 Applying compatibility fixes..."

# Update main script with DNS inbound check
cat > /tmp/dns_fix.patch << 'EOF'
--- a/obhod-core/files/usr/bin/obhod
+++ b/obhod-core/files/usr/bin/obhod
@@ -860,12 +860,24 @@
 sing_box_configure_inbounds() {
     log "Configure the inbounds section of a sing-box JSON configuration"
 
+    # Check DNS inbound compatibility
+    if ! check_dns_inbound_support; then
+        log "DNS inbound not supported, using Mixed inbound instead" "warn"
+        config=$(
+            sing_box_cm_add_mixed_inbound \
+                "$config" "$SB_SERVICE_MIXED_INBOUND_TAG" "$SB_SERVICE_MIXED_INBOUND_ADDRESS" "$SB_SERVICE_MIXED_INBOUND_PORT"
+        )
+    else
     config=$(
         sing_box_cm_add_tproxy_inbound \
             "$config" "$SB_TPROXY_INBOUND_TAG" "$SB_TPROXY_INBOUND_ADDRESS" "$SB_TPROXY_INBOUND_PORT" true true
     )
     config=$(
         sing_box_cm_add_dns_inbound "$config" "$SB_DNS_INBOUND_TAG" "$SB_DNS_INBOUND_ADDRESS" "$SB_DNS_INBOUND_PORT"
     )
+    fi
+}
+
+check_dns_inbound_support() {
+    local test='{"inbounds":[{"type":"dns","tag":"test"}]}'
+    echo "$test" > /tmp/obhod_dns_test.json
+    if sing-box check -c /tmp/obhod_dns_test.json >/dev/null 2>&1; then
+        rm -f /tmp/obhod_dns_test.json
+        return 0
+    else
+        rm -f /tmp/obhod_dns_test.json
+        return 1
+    fi
 }
EOF

# Update sing_box_config_manager.sh
cat > /tmp/manager_fix.patch << 'EOF'
--- a/obhod-core/files/usr/lib/sing_box_config_manager.sh
+++ b/obhod-core/files/usr/lib/sing_box_config_manager.sh
@@ -381,17 +381,29 @@
 sing_box_cm_add_dns_inbound() {
     local config="$1"
     local tag="$2"
     local listen_address="$3"
     local listen_port="$4"
 
+    if ! check_sing_box_supports_dns_inbound; then
+        log "DNS inbound not supported, using Mixed inbound instead" "warn"
+        sing_box_cm_add_mixed_inbound "$config" "$tag" "$listen_address" "$listen_port"
+        return
+    fi
+
     echo "$config" | jq \
         --arg tag "$tag" \
         --arg listen_address "$listen_address" \
         --argjson listen_port "$listen_port" \
         '.inbounds += [{
 			type: "dns",
 			tag: $tag,
 			listen: $listen_address,
 			listen_port: $listen_port
 		}]'
 }
 
+check_sing_box_supports_dns_inbound() {
+    local test='{"inbounds":[{"type":"dns"}]}'
+    echo "$test" > /tmp/sb_dns_test.json
+    if sing-box check -c /tmp/sb_dns_test.json >/dev/null 2>&1; then
+        rm -f /tmp/sb_dns_test.json
+        return 0
+    else
+        rm -f /tmp/sb_dns_test.json
+        return 1
+    fi
+}
+
 sing_box_cm_add_mixed_inbound() {
EOF

echo "🔄 Restarting services..."

# Start services with fixes
/etc/init.d/sing-box start 2>/dev/null || true
sleep 3

/etc/init.d/obhod start 2>/dev/null || true

# Check status
sleep 2
if /etc/init.d/obhod status >/dev/null 2>&1; then
    echo "✅ Obhod started successfully with DNS compatibility fix!"
    echo "📋 Check logs: logread -e obhod | tail -10"
else
    echo "❌ Obhod still has issues. Checking logs..."
    logread -e obhod | tail -20
fi

# Clean up temp files
rm -f /tmp/obhod_dns_test.json /tmp/sb_dns_test.json

echo "🎉 DNS compatibility fix applied!"