#!/usr/bin/env python3
import os
import subprocess
import sys
from pathlib import Path

BASE_DIR = Path(__file__).parent.parent.resolve()
SRC_DIR = BASE_DIR / "obhod-core" / "src"
DIST_DIR = BASE_DIR / "dist" / "binaries"

def build(goos, goarch, variant, suffix):
    output_name = f"obhod_{goos}_{suffix}"
    print(f"Building for {goos}/{goarch} {variant} (Target: {suffix}) -> {output_name}")
    
    # Setup environment
    env = os.environ.copy()
    env["GOOS"] = goos
    env["GOARCH"] = goarch
    env["CGO_ENABLED"] = "0"
    
    # Remove any existing MIPS/ARM specific flags
    env.pop("GOMIPS", None)
    env.pop("GOARM", None)
    
    if goarch in ("mips", "mipsle"):
        env["GOMIPS"] = variant
    elif goarch == "arm":
        env["GOARM"] = variant.lstrip("v")
        
    cmd = ["go", "build", "-ldflags=-s -w", "-o", str(DIST_DIR / output_name), "."]
    
    try:
        res = subprocess.run(cmd, cwd=str(SRC_DIR), env=env, check=True, capture_output=True, text=True)
        # Try UPX compression if available
        try:
            subprocess.run(["upx", "-9", str(DIST_DIR / output_name)], capture_output=True)
            print("  -> Compressed with UPX")
        except FileNotFoundError:
            pass
            
        size = (DIST_DIR / output_name).stat().st_size
        print(f"  -> Success: {size / 1024:.1f} KB")
    except subprocess.CalledProcessError as e:
        print(f"ERROR: Build failed for {goos}/{goarch} ({variant})")
        print(e.stderr)
        sys.exit(e.returncode)

def main():
    print("==================================================")
    # Print build information
    print("  Obhod Go Build (Python Edition)")
    print(f"  Source   : {SRC_DIR}")
    print(f"  Output   : {DIST_DIR}")
    print("==================================================")
    
    DIST_DIR.mkdir(parents=True, exist_ok=True)
    
    targets = [
        ("linux", "mipsle", "softfloat", "mipsel_24kc"),
        ("linux", "mips", "softfloat", "mips_24kc"),
        ("linux", "arm64", "", "aarch64_cortex-a53"),
        ("linux", "arm", "v7", "arm_cortex-a7_neon-vfpv4"),
        ("linux", "amd64", "", "x86_64"),
    ]
    
    for goos, goarch, variant, suffix in targets:
        build(goos, goarch, variant, suffix)
        
    print("\n=== All binaries built successfully ===")

if __name__ == "__main__":
    main()
