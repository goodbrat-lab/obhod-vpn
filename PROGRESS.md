# Progress Log: Obhod VPN Project

## Current State (May 7, 2026 — Session 2)
- **Version**: 0.3.0 (Alpha with Go Backend)
- **Repo**: `https://github.com/goodbrat-lab/obhod-vpn`
- **Installation**: `sh <(wget -qO- https://raw.githubusercontent.com/goodbrat-lab/obhod-vpn/main/install.sh)`

## Completed Tasks

### Session 1 (v0.1.0 → v0.3.0)
1. **Localization Fix**: Automated `.po` to `.lmo` compilation and dual-path installation for compatibility with OpenWrt 21.xx - 25.xx.
2. **Go Core (obhoud)**: Initialized module, implemented Watchdog, cross-compilation for 5 architectures.
3. **Backend Fixes**: WAN readiness check, FakeIP cache cleanup.
4. **Build Pipeline**: Scripts for Go compilation and `.ipk` packaging.

### Session 2 (Bug Fixes — Critical)
5. **init.d Architecture Fixed** (critical blocker):
   - `start_service()` now calls `obhod start_main` (one-shot setup) THEN registers `obhoud watchdog` in procd.
   - `stop_service()` now calls `obhod stop_main` (proper teardown).
   - Both `files/etc/init.d/obhod` and `obhod-core/files/etc/init.d/obhod` updated.

6. **Watchdog Logic Fixed** (critical):
   - Removed manual `obhoud watchdog &` launch from `start_main` (was causing double launch).
   - Watchdog now: 1) waits 60s initial delay; 2) on DNS failure, restarts sing-box first (fast); 3) escalates to full `obhod restart` only after 2 failed sing-box restarts.

7. **dnsmasq lifecycle Fixed**:
   - `dnsmasq_configure()` and `dnsmasq_restore()` moved inside `start_main`/`stop_main` (atomic).
   - `shutdown_correctly` flag written inside `start_main`/`stop_main`.

8. **Bash `ip rule` check Fixed**:
   - Changed `grep -q "obhod"` to `grep -q "fwmark $NFT_FAKEIP_MARK/$NFT_FAKEIP_MARK"` (precise match).

9. **UCI Config Fixed**:
   - `files/etc/config/obhod`: new clean template with correct section types (`obhod`/`section`).
   - `obhod-core/files/etc/config/obhod`: added missing `option enabled '1'`.

10. **Build Scripts Fixed** (all hardcoded `/root/Obhod project` paths):
    - `scripts/build_go.sh` — dynamic paths via `${BASH_SOURCE[0]}`.
    - `scripts/package_full.sh` — dynamic paths, added `prerm` script, optional file checks.
    - `scripts/package_daemon.sh` — full rewrite, version bump to 0.3.0.
    - `scripts/package_luci.sh` — full rewrite, version bump to 0.3.0.
    - `scripts/build_all.sh` — dynamic paths, error checking on each step.

## Pending
- [ ] Integrate `obhoud` Go daemon further: add WAN-check logic to Go (currently in bash only)
- [ ] Transition JSON config generation from bash/jq to Go (`sing_box_config_manager.sh` → Go)
- [ ] Localization: verify Russian strings work on OpenWrt 25 (ucode LuCI)
- [ ] Add unit tests for watchdog package
- [ ] Verify cron job for subscription/community list updates

## Known Issues / Notes
- **`badwan_monitored_interfaces`**: supported in `service_triggers()` via procd interface trigger. Needs testing on real hardware.
- **`obhod start_main` is one-shot**: procd doesn't restart it if it fails (by design). Errors logged to syslog.
- **Legacy `obhod-watchdog` bash script**: kept for reference but superseded by Go watchdog.
