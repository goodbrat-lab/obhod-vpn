# 🔧 РЕШЕНИЕ: Проблема с DNS inbound в sing-box

## 🚨 Проблема:
Ошибка: `unknown inbound type: dns` - sing-box не поддерживает DNS inbound в текущей версии.

## 🎯 ИЗВЕСТНЫЕ РЕШЕНИЯ:

### ✅ Способ 1: Изменить конфигурацию (без DNS inbound)
```bash
# Заменить DNS inbound на Mixed inbound
{
  "inbounds": [
    {
      "type": "tproxy",
      "tag": "tproxy-in", 
      "listen": "127.0.0.1",
      "listen_port": 1602
    },
    {
      "type": "mixed",  // ← ВМЕСТО dns
      "tag": "service-mixed-in",
      "listen": "127.0.0.42", 
      "listen_port": 53
    }
  ]
}
```

### ✅ Способ 2: Правильная версия sing-box
Установить full-версию sing-box с поддержкой DNS:
```bash
# OpenWrt with opkg:
opkg update
opkg remove sing-box
opkg install sing-box-full

# Или скомпилировать с флагами:
go install -tags with_dnspodhook github.com/sagernet/sing-box/cmd/sing-box@master
```

### ✅ Способ 3: Быстрое исправление
```bash
# Создать рабочую конфигурацию:
cat > /etc/sing-box/config.json << 'EOF'
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
      {"tag":"dns","server":"1.1.1.1","server_port":53}
    ],
    "final":"dns"
  },
  "route":{
    "rules":[
      {"inbound":"tproxy-in","outbound":"direct-out"}
    ],
    "final":"direct-out"
  }
}
EOF

/etc/init.d/sing-box restart
/etc/init.d/obhod restart
```

## 🔍 Проверка версии:
```bash
sing-box version
# Требуется: >= 1.12.0 с поддержкой DNS inbound
```

## 📋 Приоритеты:
1. **Простое исправление:** Использовать Mixed inbound вместо DNS
2. **Полное решение:** Установить sing-box-full  
3. **Компиляция:** Собрать с нужными флагами

## 🎯 Какой способ выбрать:
- Если нет возможности переустановить → **Способ 3**
- Если есть доступ к opkg → **Способ 2** 
- Для production → **Способ 1** (минимальный working config)

**Рекомендация:** Начните с **Способа 3** для быстрого восстановления!