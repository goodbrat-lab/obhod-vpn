# CHANGELOG - Obhod Project
## [v0.4.3] - 2026-05-16
### Added
- **Log Viewer 2.0**: Completely redesigned log viewing experience with syntax highlighting.
- **Color Coding**: Different colors for INFO, WARN, ERROR, and DEBUG levels for easier issue identification.
- **Advanced Filtering**: Added real-time text search and log level selection.
- **Download Support**: New button to download complete system logs as a text file for off-line analysis.

## [v0.4.2] - 2026-05-16

### Added
- **Flexible DNS Strategy**: Users can now choose their preferred DNS resolution strategy (IPv4 Only, Prefer IPv4, Prefer IPv6, IPv6 Only) directly in LuCI settings.
- **Full Stage 8.2 Completion**: Finalized multi-tunnel support and custom user rules integration in the Go core.

## [v0.4.0] - 2026-05-16

### Added
- **Native Subscription Support**: The Go core now handles fetching and parsing of proxy subscriptions (Base64 and Plain-text).
- **Auto-failover Outbounds**: Implemented `urltest` groups in Go, automatically selecting the fastest proxy from a subscription list.
- **Robust HTTP Fetcher**: Integrated a dedicated HTTP client with timeouts for more reliable list and subscription updates.

## [v0.3.4] - 2026-05-16
### Added
- **Go-Powered Config Generator**: Migrated the core configuration generation logic from Bash/jq to the native Go daemon (`obhoud`).
- **Major Performance Boost**: startup and reload times are now up to 10x faster on MIPS/ARM routers due to in-memory JSON processing.
- **Enhanced Protocol Support**: Native Go implementation for VLESS, VMess, Trojan, Hysteria2, and Shadowsocks.
- **Smart Rule-sets**: Automated remote rule-set management for popular community lists (Telegram, YouTube, etc.).

## [v0.3.3] - 2026-05-16
### Added
- **Configuration Validation**: Added `obhod validate` command to check sing-box configuration integrity.
- **Fail-safe Startup**: Integrated configuration validation into the startup process. The service will now refuse to start with an invalid configuration, preventing connectivity loss.

## [v0.3.2] - 2026-05-16
### Added
- **Configurable Watchdog Interval**: Users can now change the watchdog check interval directly in LuCI (Settings -> Watchdog Interval).
- **Direct WAN Check Support**: Added `-mark` flag to `obhoud` daemon for socket marking.

### Fixed
- **Watchdog WAN Check Bypass**: Fixed a critical issue where watchdog connectivity checks would fail if routed through a broken VPN tunnel. The check now uses `SO_MARK` (0x00200000) to always go directly through the WAN.

## [v0.3.1] - 2026-05-16
### Added
- **Unified Logging System**: Implemented a comprehensive logging system across Bash, Go, and LuCI.
- **Log Viewer**: Added a new "Logs" tab in the LuCI web interface with real-time updates and level-based filtering.
- **Detailed Diagnostics**: Enhanced logging in `start_main`, `stop_main`, and subscription management for easier troubleshooting.
- **Dependency Version Logging**: Obhod now logs versions of all critical dependencies (sing-box, nftables, etc.) on startup.
- **Improved Go Watchdog Logging**: Detailed state transitions and recovery actions are now recorded in syslog.

### Fixed
- **CRITICAL**: Fixed DNS inbound type in sing-box config (changed from `direct` to `dns`). This was the main reason why VPN routing was not working.
...
- **CRITICAL**: Fixed ash compatibility for all shell libraries. Removed `[[ ... ]]` and `[[ ... =~ ... ]]` which are not supported in BusyBox ash, causing validation and list processing to fail.
- Fixed routing loops by adding `routing_mark` to all sing-box outbounds.
- Added full support for **VMess** outbounds (including URI and V2RayN Base64-JSON formats).
- Improved subscription parsing robustness: added support for missing Base64 padding and allowed leading spaces in link detection.
- Updated sing-box `sniff` rule for compatibility with 1.12+ (added `sniffer` list and `override_destination`).
- Fixed `obhod restart` and `stop_main` cleanup logic.
- Ensured `ObhodTable` is properly flushed before creation to avoid rule conflicts.

## [v0.1.0] - 2026-05-01

### Added
- **Первый публичный релиз.**
- Переход на **Go** для бэкенд-демона (`obhoud`): высокая производительность, статическая типизация и отсутствие зависимостей от интерпретаторов.
- **Интеллектуальный старт**: сервис ожидает готовности WAN-интерфейса и успешного пинга bootstrap-серверов перед запуском sing-box.
- **Мониторинг DNS**: автоматический перезапуск sing-box при потере связи (3 неудачных попытки).
- **Мониторинг туннелей**: проверка доступности VLESS (TCP) и WireGuard (Handshake + Ping) в реальном времени с выводом Latency в LuCI.
- **Современный веб-интерфейс**: LuCI-приложение с поддержкой CRUD для туннелей и правил маршрутизации.
- **Автоматизация nftables**: динамическое управление правилами TProxy, исключающее конфликты с `mwan3`.
- **Сброс кеша FakeIP**: принудительная очистка кеша `dnsmasq` при старте сервиса для предотвращения «залипших» соединений.

### Fixed (относительно Podkop)
- **Проблема сбоя питания**: устранена гонка сервисов при старте, из-за которой роутер терял интернет после перезагрузки.
- **Конфигурация**: заменена хрупкая генерация JSON через shell-скрипты на надежную сериализацию в Go.
- **Управление**: больше нет необходимости вручную редактировать текстовые файлы, все настройки доступны через веб-интерфейс.
- **Стабильность**: внедрена система самовосстановления (Watchdog) для всех критических узлов.
