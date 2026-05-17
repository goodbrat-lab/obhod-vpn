# Установка Obhod

## Быстрый способ

Установка с GitHub:

```bash
sh <(wget -q -O - https://raw.githubusercontent.com/goodbrat-lab/obhod-vpn/main/install.sh)
```

Скрипт сам:

- определяет архитектуру роутера,
- устанавливает зависимости,
- скачивает `obhod` и `luci-app-obhod`,
- перезапускает LuCI-компоненты.

## Ручная установка

Нужны два пакета:

- `obhod_1.1.5-1_<arch>.ipk`
- `luci-app-obhod_1.1.5-1_all.ipk`

Установка:

```bash
opkg update
opkg install /tmp/obhod_1.1.5-1_<arch>.ipk
opkg install /tmp/luci-app-obhod_1.1.5-1_all.ipk
```

После установки:

```bash
/etc/init.d/obhod enable
/etc/init.d/obhod start
```

## Где находится интерфейс

LuCI: `Services -> Obhod`

## Проверка после установки

```bash
/etc/init.d/obhod status
/usr/bin/obhod validate
logread -e obho[ud]
```

## Отладка

Включение debug-логов:

```bash
uci set obhod.settings.log_level='debug'
uci commit obhod
/etc/init.d/obhod restart
```

Проверка FakeIP:

```bash
/usr/bin/obhod check_fakeip
```

## Сборка

Локальная сборка пакетов:

```bash
cd /root/Obhod
./scripts/build_go.sh
./scripts/package_full.sh x86_64
./scripts/package_luci.sh
./scripts/update_index.sh
```
