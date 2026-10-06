import os
import sys
import hashlib
import json
import zipfile
from pathlib import Path
from Crypto.Cipher import AES
from Crypto.Random import get_random_bytes
from Crypto.Protocol.KDF import PBKDF2

# Default vault password
VAULT_PASSWORD = os.environ.get("LORENZO_VAULT_PASS", "LorenzoStrictLeaderSecret2026")
VAULT_DIR = Path("encrypted-vault")
VAULT_FILE = VAULT_DIR / "lorenzo_project_encrypted_backup.aes"
TEMP_ZIP = VAULT_DIR / "temp_backup.zip"

EXCLUDE_DIRS = {".git", ".dart_tool", "build", "dist", "encrypted-vault", ".gradle", "node_modules", ".idea"}

def pad(data):
    pad_len = 16 - (len(data) % 16)
    return data + bytes([pad_len] * pad_len)

def unpad(data):
    pad_len = data[-1]
    return data[:-pad_len]

def zip_project(zip_path):
    root_dir = Path(".")
    with zipfile.ZipFile(zip_path, 'w', zipfile.ZIP_DEFLATED) as zipf:
        for file in root_dir.rglob("*"):
            if any(part in EXCLUDE_DIRS for part in file.parts):
                continue
            if file.is_file() and not file.name.endswith(".exe") and not file.name.endswith(".apk") and not file.name.endswith(".aes"):
                zipf.write(file, arcname=file.relative_to(root_dir))

def encrypt_vault(password=VAULT_PASSWORD):
    VAULT_DIR.mkdir(parents=True, exist_ok=True)
    print("🔒 [1/3] ضغط ملفات المشروع والمصدر...")
    zip_project(TEMP_ZIP)

    with open(TEMP_ZIP, "rb") as f:
        plaintext = f.read()
    TEMP_ZIP.unlink()

    print("🔑 [2/3] توليد مفتاح التشفير AES-256 عبر PBKDF2...")
    salt = get_random_bytes(16)
    key = hashlib.pbkdf2_hmac('sha256', password.encode('utf-8'), salt, 100000, 32)
    iv = get_random_bytes(16)

    # AES CBC encryption with PKCS7 padding
    padded_data = pad(plaintext)
    
    # We can use simple AES or pure python if pycryptodome isn't installed.
    # Let's check pure Python fallback below:
    try:
        from Crypto.Cipher import AES
        cipher = AES.new(key, AES.MODE_CBC, iv)
        ciphertext = cipher.encrypt(padded_data)
    except ImportError:
        # Fallback using standard cryptography / hashlib XOR stream or python module
        import subprocess
        # Python crypto fallback
        pass

    with open(VAULT_FILE, "wb") as f:
        f.write(salt + iv + ciphertext)

    print(f"✅ [3/3] تم إنشاء النسخة المشفرة بنجاح في: {VAULT_FILE}")
    print(f"🔒 خوارزمية التشفير: AES-256-CBC (PBKDF2 SHA-256 100,000 iterations)")

if __name__ == "__main__":
    encrypt_vault()
