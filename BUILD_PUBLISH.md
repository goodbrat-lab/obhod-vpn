# Build and Publish Instructions

## 📦 Build Packages

To build the updated packages with security fixes:

```bash
# 1. Navigate to project directory
cd /root/Obhod

# 2. Build for x86_64 architecture
./scripts/build.sh x86_64

# 3. Build for other architectures (optional)
./scripts/build.sh aarch64
./scripts/build.sh mips
```

## 📤 Build Requirements

- OpenWrt SDK for your target architecture
- Bash shell
- Standard build tools (make, gcc, etc.)

## 🚀 Publish to GitHub

```bash
# Use the automated publish script
cd /root/Obhod
./scripts/publish.sh
```

## 📋 Manual Publishing Steps

1. **Commit Changes:**
   ```bash
   git add .
   git commit -m "Security fixes v1.1.1"
   git push origin main
   ```

2. **Create GitHub Release:**
   - Go to https://github.com/goodbrat-lab/obhod-vpn
   - Click "Releases" → "Create a new release"
   - Tag: `v1.1.1`
   - Title: `Security Fixes v1.1.1`
   - Description: Copy from PUBLISH_REPORT.md
   - Upload built packages from `dist/packages/`

## 🔍 Verify Publishing

Check that all files are uploaded:
- luci-app-obhod_1.1.1-1_all.ipk
- obhod_1.1.1-1_*arch*.ipk
- SHA256SUMS file

## ✅ Success

After successful publishing:
- Users can update with: `opkg update && opkg upgrade obhod luci-app-obhod`
- Security fixes are live
- Stability improvements applied