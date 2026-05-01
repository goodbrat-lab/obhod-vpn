include $(TOPDIR)/rules.mk

PKG_NAME:=obhod
PKG_VERSION:=0.1.0
PKG_RELEASE:=1

PKG_BUILD_DIR:=$(BUILD_DIR)/$(PKG_NAME)-$(PKG_VERSION)

include $(INCLUDE_DIR)/package.mk

define Package/obhod
  SECTION:=net
  CATEGORY:=Network
  TITLE:=Reliable selective VPN routing for OpenWrt
  DEPENDS:=+sing-box +nftables +dnsmasq-full +ip-full +curl
endef

define Package/obhod/description
  Obhod is a smart VPN routing tool for OpenWrt 25.xx, replacing Podkop.
  It handles sing-box lifecycle, FakeIP caching, and nftables routing
  gracefully after power failures.
endef

define Build/Prepare
	mkdir -p $(PKG_BUILD_DIR)
	# Copy source files to build directory
	cp -a ./backend $(PKG_BUILD_DIR)/
endef

# Architecture can be overridden via ARCH variable, defaults to mipsle (for many older Xiaomi routers),
# but modern AX3000T is often arm or aarch64.
# We will use GOARCH=$$(if $$(ARCH),$$(ARCH),mipsle) or similar. 
# For simplicity and user request, we use GOARCH=$${GOARCH:-mipsle}.
define Build/Compile
	( \
		cd $(PKG_BUILD_DIR)/backend; \
		CGO_ENABLED=0 \
		GOOS=linux \
		GOARCH=$${GOARCH:-mipsle} \
		go build -ldflags="-s -w" -o $(PKG_BUILD_DIR)/obhoud ./cmd/obhoud/main.go \
	)
endef

define Package/obhod/install
	$(INSTALL_DIR) $(1)/usr/bin
	$(INSTALL_BIN) $(PKG_BUILD_DIR)/obhoud $(1)/usr/bin/obhoud

	$(INSTALL_DIR) $(1)/etc/init.d
	$(INSTALL_BIN) ./files/etc/init.d/obhod $(1)/etc/init.d/obhod

	$(INSTALL_DIR) $(1)/etc/config
	$(INSTALL_CONF) ./files/etc/config/obhod $(1)/etc/config/obhod

	$(INSTALL_DIR) $(1)/var/run/obhod
endef

$(eval $(call BuildPackage,obhod))
