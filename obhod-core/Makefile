include $(TOPDIR)/rules.mk

PKG_NAME:=obhod

PKG_VERSION := 1.1.10

PKG_RELEASE:=1

PKG_MAINTAINER:=ITDog <obhod@itdog.info>
PKG_LICENSE:=GPL-2.0-or-later

include $(INCLUDE_DIR)/package.mk

define Package/obhod
	SECTION:=net
	CATEGORY:=Network
	DEPENDS:=+sing-box +curl +jq +kmod-nft-tproxy +coreutils-base64 +bind-dig
	CONFLICTS:=https-dns-proxy nextdns luci-app-passwall luci-app-passwall2
	TITLE:=Domain routing app
	URL:=https://obhod.net
	PKGARCH:=$(ARCH)
endef

define Package/obhod/description
	Domain routing. Use of VLESS, Shadowsocks technologies
endef

define Build/Configure
endef

define Build/Compile
endef

define Package/obhod/prerm
#!/bin/sh

grep -q "105 obhod" /etc/iproute2/rt_tables && sed -i "/105 obhod/d" /etc/iproute2/rt_tables

/etc/init.d/obhod stop

exit 0
endef

define Package/obhod/conffiles
/etc/config/obhod
endef

define Package/obhod/install
	$(INSTALL_DIR) $(1)/etc/init.d
	$(INSTALL_BIN) ./files/etc/init.d/obhod $(1)/etc/init.d/obhod

	$(INSTALL_DIR) $(1)/etc/config
	$(INSTALL_CONF) ./files/etc/config/obhod $(1)/etc/config/obhod

	$(INSTALL_DIR) $(1)/usr/bin
	$(INSTALL_BIN) ./files/usr/bin/obhod $(1)/usr/bin/obhod

	$(INSTALL_DIR) $(1)/usr/lib/obhod
	$(CP) ./files/usr/lib/* $(1)/usr/lib/obhod/

	sed -i -e 's/__COMPILED_VERSION_VARIABLE__/$(PKG_VERSION)/g' $(1)/usr/lib/obhod/constants.sh
endef

$(eval $(call BuildPackage,obhod))
