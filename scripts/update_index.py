#!/usr/bin/env python3
import os
import tarfile
import hashlib
import gzip
from pathlib import Path

BASE_DIR = Path(__file__).parent.parent.resolve()
DIST_DIR = BASE_DIR / "dist" / "packages"

def sha256(path: Path) -> str:
    h = hashlib.sha256()
    with open(path, "rb") as f:
        while chunk := f.read(65536):
            h.update(chunk)
    return h.hexdigest()

def update_index():
    print(f"Generating Packages index in {DIST_DIR}...")
    packages_txt = ""

    # Sort to ensure deterministic index order
    ipk_files = sorted(DIST_DIR.glob("*.ipk"))

    for ipk in ipk_files:
        print(f"  Processing {ipk.name}...")
        control_data = ""
        
        # Read control.tar.gz from IPK (new format: tar.gz; old format: ar — skip gracefully)
        try:
            with tarfile.open(ipk, "r:gz") as tar:
                # Extract control.tar.gz
                control_member = None
                for member in tar.getmembers():
                    if "control.tar.gz" in member.name:
                        control_member = member
                        break
                
                if control_member:
                    f_control_tar = tar.extractfile(control_member)
                    if f_control_tar:
                        # Parse the control file inside control.tar.gz
                        with tarfile.open(fileobj=f_control_tar, mode="r:gz") as ctrl_tar:
                            for member in ctrl_tar.getmembers():
                                if member.name.endswith("control") or member.name == "./control":
                                    control_file = ctrl_tar.extractfile(member)
                                    if control_file:
                                        control_data = control_file.read().decode("utf-8")
                                        break
        except Exception as e:
            print(f"    SKIP: {ipk.name} — cannot parse ({e})")

        if control_data:
            # Ensure the control fields end with a newline
            control_data = control_data.strip() + "\n"
            packages_txt += control_data
            packages_txt += f"Filename: {ipk.name}\n"
            packages_txt += f"Size: {ipk.stat().st_size}\n"
            packages_txt += f"SHA256sum: {sha256(ipk)}\n"
            packages_txt += "\n"

    # Write Packages
    packages_path = DIST_DIR / "Packages"
    with open(packages_path, "w", newline="\n", encoding="utf-8") as f:
        f.write(packages_txt)

    # Write Packages.gz
    packages_gz_path = DIST_DIR / "Packages.gz"
    with gzip.open(packages_gz_path, "wb") as f:
        f.write(packages_txt.encode("utf-8"))

    # Copy to root directory for install.sh local testing compatibility if needed
    root_packages_path = BASE_DIR / "Packages"
    root_packages_gz_path = BASE_DIR / "Packages.gz"
    with open(root_packages_path, "w", newline="\n", encoding="utf-8") as f:
        f.write(packages_txt)
    with gzip.open(root_packages_gz_path, "wb") as f:
        f.write(packages_txt.encode("utf-8"))

    print("Index generation completed successfully (Packages and Packages.gz created).")

if __name__ == "__main__":
    update_index()
