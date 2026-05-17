# Build And Publish

## Build

Production-сборка выполняется локально, без автоматического коммита и push:

```bash
cd /root/Obhod
./scripts/build_go.sh
./scripts/package_full.sh x86_64
./scripts/package_luci.sh
./scripts/update_index.sh
```

Для полного цикла локальной сборки всех поддерживаемых архитектур:

```bash
cd /root/Obhod
./scripts/build_all.sh
```

Поддерживаемые архитектуры:

- `mipsel_24kc`
- `mips_24kc`
- `aarch64_cortex-a53`
- `arm_cortex-a7_neon-vfpv4`
- `x86_64`

## Publish

Публикация выполняется отдельным шагом после ручной проверки артефактов.

Перед публикацией проверьте, что в `dist/packages/` находятся только нужные файлы репозитория пакетов:

- `Packages`
- `Packages.gz`
- `index.txt`
- `luci-app-obhod_1.1.5-1_all.ipk`
- `obhod_1.1.5-1_<arch>.ipk`

Затем:

```bash
git add .
git commit -m "release: prepare v1.1.5 packages"
git push origin main
```

После этого можно создать GitHub release и приложить нужные `.ipk`.
