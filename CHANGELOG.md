# CHANGELOG - Obhod Project
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
