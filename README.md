# 🔐 VaultSeal – GPG API Service

**VaultSeal** คือบริการ API สำหรับจัดการกุญแจ PGP และการเข้ารหัส/ถอดรหัสไฟล์แบบอัตโนมัติ  
ออกแบบมาเพื่อใช้งานในระบบ Backend, Integration, และ File Exchange ที่ต้องการความปลอดภัยระดับองค์กร

ระบบรันบน Docker ใช้ **Go** และ **GnuPG (GPG)** เป็นแกนหลัก  
รองรับทั้งการใช้งานแบบ **Encrypt-only** และ **Encrypt & Decrypt (Full Mode)**

---

## ✨ Key Features

- 🔑 สร้าง PGP Key Pair (Public / Private) ผ่าน API  
- 🔒 เข้ารหัสไฟล์ด้วย Public Key  
- 🔓 ถอดรหัสไฟล์ด้วย Private Key + Passphrase  
- 🐳 ทำงานบน Docker (Portable, Reproducible)  
- ⚙️ Configurable ผ่าน Environment Variables  
- 🏢 เหมาะสำหรับ Bank Integration, SFTP Replacement, Secure File Exchange  

---

## 🧱 Architecture Overview

```
Client
  │
  ├─ Upload File
  ▼
VaultSeal API (Go)
  │
  ├─ GnuPG
  │   ├─ Public Key
  │   └─ Private Key + Passphrase
  ▼
Encrypted / Decrypted File
```

> **Design Decision**  
> VaultSeal ใช้ GPG CLI โดยตรงเพื่อหลีกเลี่ยงความเสี่ยงด้าน Cryptography Bug  
> และสอดคล้องกับ Security / Compliance Practice ระดับองค์กร

---

## 🛠 Installation

```bash
docker build -t vaultseal .
```

> ⚠️ ไฟล์กุญแจ `.asc` ต้องอยู่ใน directory เดียวกับ `Dockerfile`

---

## 🚀 Running the Service

### 🔒 Mode 1: Encryption Only (Public Key Only)

```bash
docker run -d   --name vaultseal   -e GPG_KEY_FILE=KBankH2HPgpUAT.asc   -p 8080:8080   vaultseal
```

---

### 🔐 Mode 2: Full Mode (Encrypt & Decrypt)

```bash
docker run -d   --name vaultseal   -e GPG_KEY_FILE=demo_public.asc   -e GPG_PRIV_KEY_FILE=demo_private.asc   -e GPG_PASSPHRASE=your_password_here   -p 8080:8080   vaultseal
```

---

## 📡 API Usage

### 🔑 Generate Key Pair

```bash
curl "http://localhost:8080/keys/generate?name=PENK&email=demo@example.com"   --output my_keys.txt
```

---

### 🔒 Encrypt File

```bash
curl -X POST -F "file=@document.txt"   http://localhost:8080/encrypt -J -O
```

---

### 🔓 Decrypt File

```bash
curl -X POST -F "file=@document.txt.gpg"   http://localhost:8080/decrypt -J -O
```

---

## ⚠️ Important Notes

- ห้าม commit Private Key
- แนะนำใช้ Secret Manager
- ระบบใช้ `--always-trust` เพื่อรองรับ Automation

---

## 🏁 Summary

VaultSeal คือ GPG API Service ที่ออกแบบมาเพื่อใช้งานจริงใน Production  
ลดความซับซ้อน แต่ยังรักษามาตรฐานความปลอดภัย
