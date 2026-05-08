#!/usr/bin/env python3
"""
Build .ipk packages for Obhod from Windows.
Requires: Python 3.x (no external dependencies)
Usage: python3 scripts/build_ipk.py
"""

import os
import sys
import tarfile
import shutil
import hashlib
from pathlib import Path

# --- Config ---
VERSION = "0.3.0"
RELEASE = "2"

BASE_DIR = Path(__file__).parent.parent.resolve()
CORE_FILES = BASE_DIR / "obhod-core" / "files"
DIST_DIR = BASE_DIR / "dist"
PACKAGES_DIR = DIST_DIR / "packages"
BINARIES_DIR = DIST_DIR / "binaries"

ARCH_MAP = {
    "mipsle_softfloat": "mipsel_24kc",   # opkg arch name for Xiaomi AX3000T
    "mips_softfloat":   "mips_24kc",
    "arm64":            "aarch64_cortex-a53",
    "arm_v7":           "arm_cortex-a7_neon-vfpv4",
    "amd64":            "x86_64",
}

def sha256(path: Path) -> str:
    h = hashlib.sha256()
    with open(path, "rb") as f:
        while chunk := f.read(65536):
            h.update(chunk)
    return h.hexdigest()

def add_to_tar(tar: tarfile.TarFile, real_path: Path, arcname: str, mode: int = None):
    info = tar.gettarinfo(str(real_path), arcname=arcname)
    if mode is not None:
        info.mode = mode
    # Clear metadata that causes PAX extended headers in busybox-incompatible format
    info.uid = 0
    info.gid = 0
    info.uname = "root"
    info.gname = "root"
    info.mtime = 0
    with open(real_path, "rb") as f:
        tar.addfile(info, f)

def build_data_tar(tmp_dir: Path, binary_path: Path) -> Path:
    """Create data.tar.gz with all installed files."""
    data_dir = tmp_dir / "data"
    data_dir.mkdir(parents=True)

    # Directories to create
    dirs = [
        "usr/bin",
        "usr/lib/obhod",
        "etc/init.d",
        "etc/config",
    ]
    for d in dirs:
        (data_dir / d).mkdir(parents=True, exist_ok=True)

    # Copy files
    files = {
        binary_path:                                         data_dir / "usr/bin/obhoud",
        CORE_FILES / "usr/bin/obhod":                        data_dir / "usr/bin/obhod",
        CORE_FILES / "etc/init.d/obhod":                     data_dir / "etc/init.d/obhod",
        CORE_FILES / "etc/config/obhod":                     data_dir / "etc/config/obhod",
    }

    # lib files (all .sh and .jq)
    lib_src = CORE_FILES / "usr/lib"
    if lib_src.exists():
        for f in lib_src.rglob("*"):
            if f.is_file():
                rel = f.relative_to(lib_src)
                dest = data_dir / "usr/lib" / rel
                dest.parent.mkdir(parents=True, exist_ok=True)
                files[f] = dest

    # --- LuCI UI files (menu entry + JS views + ACL) ---
    LUCI_SRC = BASE_DIR / "luci-app-obhod"

    # JS view files → /www/luci-static/resources/view/obhod/
    js_view_src = LUCI_SRC / "htdocs/luci-static/resources/view/obhod"
    if js_view_src.exists():
        for f in js_view_src.iterdir():
            if f.is_file():
                dest = data_dir / "www/luci-static/resources/view/obhod" / f.name
                dest.parent.mkdir(parents=True, exist_ok=True)
                files[f] = dest

    # Menu descriptor → /usr/share/luci/menu.d/
    menu_src = LUCI_SRC / "root/usr/share/luci/menu.d/luci-app-obhod.json"
    if menu_src.exists():
        dest = data_dir / "usr/share/luci/menu.d/luci-app-obhod.json"
        dest.parent.mkdir(parents=True, exist_ok=True)
        files[menu_src] = dest

    # ACL → /usr/share/rpcd/acl.d/
    acl_src = LUCI_SRC / "root/usr/share/rpcd/acl.d/luci-app-obhod.json"
    if acl_src.exists():
        dest = data_dir / "usr/share/rpcd/acl.d/luci-app-obhod.json"
        dest.parent.mkdir(parents=True, exist_ok=True)
        files[acl_src] = dest

    for src, dst in files.items():
        if not src.exists():
            print(f"  WARNING: missing source file: {src}")
            continue
        shutil.copy2(src, dst)

    # Set execute bits (stored in tar)
    exec_files = [
        data_dir / "usr/bin/obhoud",
        data_dir / "usr/bin/obhod",
        data_dir / "etc/init.d/obhod",
    ]

    data_tar = tmp_dir / "data.tar.gz"
    # GNU_FORMAT: busybox tar on OpenWrt does not support PAX extended headers (0x78)
    with tarfile.open(data_tar, "w:gz", format=tarfile.GNU_FORMAT) as tar:
        for item in sorted(data_dir.rglob("*")):
            arcname = "./" + str(item.relative_to(data_dir)).replace("\\", "/")
            if item.is_dir():
                info = tarfile.TarInfo(name=arcname)
                info.type = tarfile.DIRTYPE
                info.mode = 0o755
                info.uid = 0; info.gid = 0
                info.uname = "root"; info.gname = "root"
                info.mtime = 0
                tar.addfile(info)
            else:
                mode = 0o755 if item in exec_files else 0o644
                add_to_tar(tar, item, arcname, mode)

    return data_tar

def build_control_tar(tmp_dir: Path, arch_ipk: str, binary_path: Path) -> Path:
    """Create control.tar.gz."""
    ctrl_dir = tmp_dir / "control"
    ctrl_dir.mkdir()

    # control file
    control_text = f"""Package: obhod
Version: {VERSION}-{RELEASE}
Architecture: {arch_ipk}
Maintainer: Obhod Team <obhod@itdog.info>
Section: net
Priority: optional
Depends: sing-box, nftables, dnsmasq-full, ip-full, curl, jq, coreutils-base64
Description: Obhod VPN - reliable selective routing for OpenWrt
 Obhod provides stable and intelligent selective traffic routing on
 OpenWrt routers. Features: WAN-ready startup, DNS health watchdog,
 FakeIP cache management, nftables TProxy routing.
"""
    # IMPORTANT: all text files must use LF (not CRLF) — opkg rejects CRLF in control
    (ctrl_dir / "control").write_bytes(control_text.encode("utf-8"))

    postinst_text = """#!/bin/sh
[ -n "${IPKG_INSTROOT}" ] && exit 0
chmod +x /usr/bin/obhoud /usr/bin/obhod /etc/init.d/obhod 2>/dev/null
/etc/init.d/obhod enable 2>/dev/null
# Clear LuCI cache so the menu entry appears immediately
rm -rf /tmp/luci-indexcache* /tmp/luci-modulecache/ 2>/dev/null
/etc/init.d/rpcd restart 2>/dev/null
exit 0
"""
    (ctrl_dir / "postinst").write_bytes(postinst_text.encode("utf-8"))

    prerm_text = """#!/bin/sh
[ -n "${IPKG_INSTROOT}" ] && exit 0
/etc/init.d/obhod stop 2>/dev/null
/etc/init.d/obhod disable 2>/dev/null
grep -q "105 obhod" /etc/iproute2/rt_tables 2>/dev/null && sed -i "/105 obhod/d" /etc/iproute2/rt_tables
exit 0
"""
    (ctrl_dir / "prerm").write_bytes(prerm_text.encode("utf-8"))

    control_tar = tmp_dir / "control.tar.gz"
    with tarfile.open(control_tar, "w:gz", format=tarfile.GNU_FORMAT) as tar:
        for item in sorted(ctrl_dir.iterdir()):
            arcname = "./" + item.name
            mode = 0o755 if item.name in ("postinst", "prerm") else 0o644
            add_to_tar(tar, item, arcname, mode)

    return control_tar

def build_ipk(suffix: str, arch_ipk: str):
    binary_path = BINARIES_DIR / f"obhoud_linux_{suffix}"
    if not binary_path.exists():
        print(f"  SKIP: binary not found: {binary_path.name}")
        return None

    output_name = f"obhod_{VERSION}-{RELEASE}_{arch_ipk}.ipk"
    output_path = PACKAGES_DIR / output_name

    print(f"  [{suffix}] -> {output_name}")

    tmp_dir = Path(f"/tmp/obhod_ipk_{suffix}") if os.name != "nt" else BASE_DIR / f".tmp_ipk_{suffix}"
    if tmp_dir.exists():
        shutil.rmtree(tmp_dir)
    tmp_dir.mkdir(parents=True)

    try:
        data_tar    = build_data_tar(tmp_dir, binary_path)
        control_tar = build_control_tar(tmp_dir, arch_ipk, binary_path)

        # debian-binary must be LF only
        debian_binary = tmp_dir / "debian-binary"
        debian_binary.write_bytes(b"2.0\n")

        # Final .ipk = tar of {debian-binary, control.tar.gz, data.tar.gz}
        # MUST be GNU_FORMAT — busybox opkg cannot parse PAX headers
        with tarfile.open(output_path, "w:gz", format=tarfile.GNU_FORMAT) as ipk:
            for f in [debian_binary, control_tar, data_tar]:
                info = tarfile.TarInfo(name=f.name)
                info.size = f.stat().st_size
                info.mode = 0o644
                info.uid = 0; info.gid = 0
                info.uname = "root"; info.gname = "root"
                info.mtime = 0
                with open(f, "rb") as fh:
                    ipk.addfile(info, fh)

        size_kb = output_path.stat().st_size // 1024
        checksum = sha256(output_path)[:12]
        print(f"    OK: {size_kb} KB  sha256:{checksum}...")
        return output_path

    finally:
        shutil.rmtree(tmp_dir, ignore_errors=True)

def main():
    print(f"\nObhod IPK Builder — v{VERSION}-{RELEASE}")
    print(f"Base:     {BASE_DIR}")
    print(f"Packages: {PACKAGES_DIR}\n")

    PACKAGES_DIR.mkdir(parents=True, exist_ok=True)

    built = []
    for suffix, arch_ipk in ARCH_MAP.items():
        result = build_ipk(suffix, arch_ipk)
        if result:
            built.append(result)

    # Write index
    index_path = PACKAGES_DIR / "index.txt"
    with open(index_path, "w") as f:
        for p in built:
            f.write(f"{p.name}\n")
    
    print(f"\nBuilt {len(built)}/{len(ARCH_MAP)} packages:")
    for p in built:
        print(f"  {p.name}")
    print(f"\nIndex: {index_path}")

if __name__ == "__main__":
    main()
