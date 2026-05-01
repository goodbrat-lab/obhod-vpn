include $(TOPDIR)/rules.mk

LUCI_TITLE:=LuCI support for Obhod VPN
LUCI_DEPENDS:=+obhod +curl +luci-lua-runtime
LUCI_PKGARCH:=all

PKG_NAME:=luci-app-obhod
PKG_VERSION:=0.1.0
PKG_RELEASE:=1

include $(TOPDIR)/feeds/luci/luci.mk

# call BuildPackage - OpenWrt buildroot signature
