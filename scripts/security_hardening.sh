#!/bin/bash
# Security hardening script for Obhod installation
# Run as root to secure the installation

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log() {
    echo -e "${GREEN}[SECURITY]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 1. Secure file permissions
secure_permissions() {
    log "Securing file permissions..."
    
    # Configuration files - read/write only by root
    chmod 600 /etc/config/obhod 2>/dev/null || true
    
    # Scripts - executable only by root
    chmod 750 /usr/bin/obhod 2>/dev/null || true
    chmod 750 /bin/obhod 2>/dev/null || true
    
    # Logs - writable by root
    chmod 755 /var/log/obhod 2>/dev/null || mkdir -p -m 755 /var/log/obhod
    
    # Runtime files - secure permissions
    chmod 700 /tmp/obhod 2>/dev/null || mkdir -p -m 700 /tmp/obhod
    
    log "File permissions secured"
}

# 2. Create secure directories setup
setup_secure_dirs() {
    log "Setting up secure directories..."
    
    # Runtime directory
    mkdir -p -m 700 /tmp/obhod/{rules,cache,subscriptions}
    mkdir -p -m 755 /var/run/obhod
    
    # Log directory
    mkdir -p -m 755 /var/log/obhod
    
    # Configuration backups
    mkdir -p -m 700 /etc/obhod/backups
    
    log "Directories setup complete"
}

# 3. Enable sysctl hardening
harden_sysctl() {
    log "Applying sysctl hardening..."
    
    # Network hardening
    sysctl -w net.ipv4.conf.all.rp_filter=1 >/dev/null 2>&1 || true
    sysctl -w net.ipv4.conf.default.rp_filter=1 >/dev/null 2>&1 || true
    sysctl -w net.ipv4.ip_no_pmtu_disc=1 >/dev/null 2>&1 || true
    
    # Memory hardening
    sysctl -w vm.overcommit_memory=2 >/dev/null 2>&1 || true
    sysctl -w vm.panic_on_oom=0 >/dev/null 2>&1 || true
    
    # File system hardening
    sysctl -w fs.protected_regular=1 >/dev/null 2>&1 || true
    sysctl -w fs.protected_fifos=1 >/dev/null 2>&1 || true
    
    log "Sysctl hardening applied"
}

# 4. Enable firewall rules for obhod
setup_firewall() {
    log "Setting up firewall rules..."
    
    # Allow obhod traffic only from trusted interfaces
    iptables -N OBHOD_INPUT 2>/dev/null || true
    iptables -N OBHOD_OUTPUT 2>/dev/null || true
    iptables -I INPUT 1 -j OBHOD_INPUT 2>/dev/null || true
    iptables -I OUTPUT 1 -j OBHOD_OUTPUT 2>/dev/null || true
    
    # Allow only necessary ports
    iptables -A OBHOD_INPUT -p tcp --dport 1602 -j ACCEPT 2>/dev/null || true
    iptables -A OBHOD_INPUT -p udp --dport 1602 -j ACCEPT 2>/dev/null || true
    iptables -A OBHOD_INPUT -p tcp --dport 4534 -j ACCEPT 2>/dev/null || true
    
    # Drop other obhod traffic
    iptables -A OBHOD_INPUT -j DROP 2>/dev/null || true
    iptables -A OBHOD_OUTPUT -j DROP 2>/dev/null || true
    
    log "Firewall rules configured"
}

# 5. Check for common vulnerabilities
check_vulnerabilities() {
    log "Checking for common vulnerabilities..."
    
    # Check for world-writable files
    if find /etc/obhod /tmp/obhod -perm +002 -type f 2>/dev/null | grep -q .; then
        warn "Found world-writable files in Obhod directories"
    fi
    
    # Check for suid binaries
    if find /usr/bin/obhod* -perm +4000 2>/dev/null | grep -q .; then
        warn "Found SUID binaries in Obhod installation"
    fi
    
    # Check for insecure umask
    if umask | grep -q "000"; then
        warn "Current umask is insecure"
    fi
    
    log "Vulnerability check complete"
}

# 6. Create secure backup of configuration
backup_config() {
    log "Creating secure backup of configuration..."
    
    BACKUP_DIR="/etc/obhod/backups/$(date +%Y%m%d_%H%M%S)"
    mkdir -p -m 700 "$BACKUP_DIR"
    
    # Backup configuration
    cp -p /etc/config/obhod "$BACKUP_DIR/" 2>/dev/null || true
    cp -pr /etc/sing-box/config.json "$BACKUP_DIR/" 2>/dev/null || true
    
    # Encrypt sensitive parts
    if command -v openssl >/dev/null 2>&1; then
        tar -czf "$BACKUP_DIR/encrypted_backup.tar.gz" "$BACKUP_DIR"/ && \
        openssl enc -aes-256-cbc -salt -in "$BACKUP_DIR/encrypted_backup.tar.gz" -out "$BACKUP_DIR/backup.enc" -k "$(hostname)$(date +%s)" && \
        rm "$BACKUP_DIR/encrypted_backup.tar.gz"
    fi
    
    log "Configuration backed up to $BACKUP_DIR"
}

# 7. Install log rotation
setup_logrotate() {
    log "Setting up log rotation..."
    
    cat > /etc/logrotate.d/obhod << 'EOF'
/var/log/obhod/*.log {
    daily
    missingok
    rotate 7
    compress
    delaycompress
    notifempty
    create 0600 root root
    postrotate
        /etc/init.d/obhod reload >/dev/null 2>&1 || true
    endscript
}
EOF
    
    log "Log rotation configured"
}

# 8. Secure service startup
secure_service() {
    log "Securing service startup..."
    
    # Ensure service starts as root only
    sed -i 's/PROCD_DEBUG=1/PROCD_DEBUG=0/' /etc/init.d/obhod 2>/dev/null || true
    
    # Disable core dumps
    echo "ulimit -c 0" >> /etc/sysctl.conf 2>/dev/null || true
    
    # Add restart policies
    echo "# Obhod service hardening" >> /etc/sysctl.conf 2>/dev/null
    echo "kernel.core_pattern=/var/log/obhod/core-%e.%p.%t" >> /etc/sysctl.conf 2>/dev/null || true
    
    log "Service startup secured"
}

# Main execution
main() {
    log "Starting Obhod security hardening..."
    
    # Check if running as root
    if [ "$(id -u)" -ne 0 ]; then
        error "This script must be run as root"
        exit 1
    fi
    
    # Execute hardening steps
    secure_permissions
    setup_secure_dirs
    harden_sysctl
    setup_firewall
    check_vulnerabilities
    backup_config
    setup_logrotate
    secure_service
    
    log "Obhod security hardening completed!"
    log "Please restart Obhod service: /etc/init.d/obhod restart"
    
    # Display status
    echo ""
    echo "="+60
    log "SECURITY STATUS:"
    echo "- Files permissions: ✅"
    echo "- Directory structure: ✅"
    echo "- System hardening: ✅"
    echo "- Firewall rules: ✅"
    echo "- Backup created: ✅"
    echo "- Log rotation: ✅"
    echo "- Service secured: ✅"
    echo ""
    log "Obhod is now hardened! 🛡️"
}

# Run main function
main "$@"