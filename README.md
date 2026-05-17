# Obhod

Obhod - приложение для OpenWrt с выборочной маршрутизацией трафика через `sing-box` и FakeIP. Нужные ресурсы направляются в туннель, остальной трафик идет напрямую.

Текущий релиз проекта: `1.1.5`.

## Возможности

- LuCI-интерфейс для настройки и диагностики
- Go-демон `obhoud` для watchdog и системных проверок
- Генерация конфигурации `sing-box` из UCI
- Поддержка FakeIP и selective routing
- Импорт удаленных domain/subnet lists и subscriptions
- Логи и встроенные проверки состояния сервиса

## Быстрая установка

```bash
sh <(wget -q -O - https://raw.githubusercontent.com/goodbrat-lab/obhod-vpn/main/install.sh)
```

Инсталлятор устанавливает:

- `obhod_<version>_<arch>.ipk`
- `luci-app-obhod_<version>_all.ipk`

После установки интерфейс доступен в LuCI: `Services -> Obhod`.

## Диагностика

- Основные логи: `logread -e obho[ud]`
- Поток логов: `logread -f -e obho[ud]`
- Проверка сервиса: `/etc/init.d/obhod status`
- Валидация конфигурации: `/usr/bin/obhod validate`

Для расширенного логирования:

```bash
uci set obhod.settings.log_level='debug'
uci commit obhod
/etc/init.d/obhod restart
```

## Сборка

Локальная сборка production-артефактов:

```bash
cd /root/Obhod
./scripts/build_go.sh
./scripts/package_full.sh x86_64
./scripts/package_luci.sh
./scripts/update_index.sh
```

Для остальных архитектур используйте:

- `mipsel_24kc`
- `mips_24kc`
- `aarch64_cortex-a53`
- `arm_cortex-a7_neon-vfpv4`
- `x86_64`

## Публикация

Сборка и публикация разделены. Скрипты сборки больше не выполняют автоматический `git push`.

Перед публикацией проверьте содержимое `dist/packages/` и загрузите только нужные артефакты репозитория пакетов.

## Лицензия

MIT
