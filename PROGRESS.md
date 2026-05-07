# Progress Log: Obhod VPN Project (Restoration & Enhancement)

## Current State (May 7, 2026)
- **Version**: 0.3.0 (Alpha with Go Backend)
- **Repo**: `https://github.com/goodbrat-lab/obhod-vpn`
- **Installation**: `sh <(wget -qO- https://raw.githubusercontent.com/goodbrat-lab/obhod-vpn/main/install.sh)`

## Completed Tasks
1.  **Localization Fix**: Automated `.po` to `.lmo` compilation and dual-path installation (`/usr/share/luci/i18n/` and `/usr/lib/lua/luci/i18n/`) for compatibility with OpenWrt 21.xx - 25.xx.
2.  **Go Core (obhoud)**: 
    - Initialized Go module and project structure.
    - Implemented **Intelligent Watchdog** in Go (low memory, native DNS resolution, zero dependencies).
    - Created cross-compilation script `build_go.sh` supporting `mipsle`, `mips`, `arm64`, `armv7`, and `amd64`.
3.  **Backend Fixes**: 
    - Added WAN readiness check (waits up to 120s for internet) before starting services.
    - Integrated FakeIP cache cleanup (`rm -f /tmp/sing-box/cache.db`) to prevent stale IP issues.
4.  **Autonomous Watchdog**: Implemented both bash (legacy) and Go (new) versions.

## Pending
- Integrate `obhoud` into the main `init.d` script.
- Automate architecture detection in `install.sh` to deliver the correct binary.
- Transition JSON config generation from bash/jq to Go.

## Known Issues / Pending
- **Localization**: Interface may still show in English on some OpenWrt versions; need to double-check `.lmo` loading paths for ucode-based LuCI.
- **Auto-Update**: Cron job for background subscription updates is defined in logic but needs verification of scheduling.

## Tomorrow's Goals
- Verify subscription update cron job.
- Fix localization (Russian language) for modern LuCI.
- Further optimize `sing-box` config generation if specific protocols need adjustments.

*Session paused at 21:30.*
