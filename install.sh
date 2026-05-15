#!/bin/sh

# Obhod VPN Installer v0.3.0
# Usage (busybox-compatible, works on OpenWrt):
#   wget -qO /tmp/install.sh "https://raw.githubusercontent.com/goodbrat-lab/obhod-vpn/main/install.sh" && sh /tmp/install.sh

set -e

REPO_RAW="https://raw.githubusercontent.com/goodbrat-lab/obhod-vpn/main"
VERSION="0.3.0-6"

echo "=========================================="
echo "  Установщик Obhod VPN v${VERSION}"
echo "=========================================="

# 1. Detect architecture (opkg canonical name)
# opkg print-architecture lists pseudo-archs first (all, noarch), then real ones
ARCH=$(opkg print-architecture | awk '{print $2}' | grep -vE "^(all|noarch)$" | head -n1)
echo "Определена архитектура: $ARCH"

# 2. Map to package name
case "$ARCH" in
    mipsel_24kc|mipsel_74kc|mipsel_mips32)
        PKG_ARCH="mipsel_24kc" ;;
    mips_24kc|mips_74kc|mips_mips32)
        PKG_ARCH="mips_24kc" ;;
    aarch64_cortex-a53|aarch64_cortex-a72|aarch64_generic)
        PKG_ARCH="aarch64_cortex-a53" ;;
    arm_cortex-a7_neon-vfpv4|arm_cortex-a15_neon-vfpv4|arm_cortex-a9)
        PKG_ARCH="arm_cortex-a7_neon-vfpv4" ;;
    x86_64)
        PKG_ARCH="x86_64" ;;
    *)
        echo ""
        echo "ВНИМАНИЕ: Неизвестная архитектура '$ARCH'"
        echo "Доступные пакеты: mipsel_24kc, mips_24kc, aarch64_cortex-a53,"
        echo "                    arm_cortex-a7_neon-vfpv4, x86_64"
        echo ""
        echo "Задайте PKG_ARCH вручную и запустите снова:"
        echo "  PKG_ARCH=mipsel_24kc sh <(wget -qO- $REPO_RAW/install.sh)"
        exit 1 ;;
esac

PKG_NAME="obhod_${VERSION}_${PKG_ARCH}.ipk"
PKG_URL="${REPO_RAW}/dist/packages/${PKG_NAME}"

echo "Пакет: $PKG_NAME"

# 3. Update package lists first (before downloading .ipk to avoid false warnings)
echo "Обновление списков пакетов..."
opkg update 2>/dev/null || true

# Install sing-box if not present
if ! opkg list-installed | grep -q "^sing-box "; then
    echo "Установка sing-box..."
    opkg install sing-box || echo "ВНИМАНИЕ: не удалось установить sing-box, пожалуйста, установите вручную"
fi

# 4. Download (use unique name to avoid opkg scanning it)
IPK_TMP="/tmp/obhod_$$.ipk"
echo "Скачивание $PKG_NAME..."
wget --no-check-certificate -q --show-progress -O "$IPK_TMP" "$PKG_URL" 2>/dev/null || \
wget --no-check-certificate -O "$IPK_TMP" "$PKG_URL"

if [ ! -s "$IPK_TMP" ]; then
    echo "ОШИБКА: Сбой скачивания или файл пуст!"
    echo "URL: $PKG_URL"
    exit 1
fi

echo "Скачано: $(wc -c < "$IPK_TMP") байт"

# 5. Install Obhod
echo "Установка Obhod..."
opkg install --force-reinstall --force-overwrite --add-arch "${PKG_ARCH}:200" "$IPK_TMP"
rm -f "$IPK_TMP"


# 6. Post-install
echo ""
echo "=========================================="
echo "  Obhod успешно установлен!"
echo "=========================================="
echo ""
echo "  Дальнейшие шаги:"
echo "  1. Откройте веб-интерфейс (LuCI) в меню: Сервисы -> Obhod"
echo "  2. Добавьте вашу подписку или ссылку на прокси"
echo "  3. Поставьте галочку \"Включить\" и нажмите \"Сохранить и применить\""
echo "  4. Проверьте логи: logread | grep obhod"
echo ""
