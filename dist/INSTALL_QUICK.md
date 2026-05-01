# Инструкция по установке Obhod

## 1. Выбор пакета под вашу архитектуру
Чтобы узнать точную архитектуру вашего роутера, выполните команду на роутере:
`opkg print-architecture | awk '{print $2}'`

- **Cudy WR3000 / WR3000E**: Используйте **`aarch64_cortex-a53`**
- **Xiaomi AX3000T**: Обычно **`aarch64_cortex-a53`** или **`mips_24kc`** (зависит от версии)
- **Веб-интерфейс**: Пакет `luci-app-obhod` (all) нужен всегда.

## 2. Важное предусловие (Конфликт и Рантайм)
Для работы Obhod требуется пакет `dnsmasq-full` и рантайм Lua для интерфейса.

**Выполните на роутере перед установкой Obhod:**
```bash
opkg update
opkg remove dnsmasq --force-remove
opkg install dnsmasq-full luci-lua-runtime
```


## 3. Команды для установки (пример для aarch64_cortex-a53)

### С компьютера (перенос файлов):
```bash
scp obhod_0.1.0-1_aarch64_cortex-a53.ipk luci-app-obhod_0.1.0-1_all.ipk root@192.168.1.1:/tmp/
```

### На роутере (установка):
```bash
opkg install /tmp/obhod_0.1.0-1_aarch64_cortex-a53.ipk /tmp/luci-app-obhod_0.1.0-1_all.ipk
```

## 4. После установки
1. Зайдите в веб-интерфейс: **Службы -> Obhod VPN**.
2. Добавьте туннель (вкладка **Tunnels**) через импорт VLESS-ссылки или вручную.
3. Добавьте правила маршрутизации (вкладка **Routing Rules**) для нужных доменов.
4. Проверьте статус на главной странице модуля.

---
**Логи демона:** `logread -f -e obhoud`
