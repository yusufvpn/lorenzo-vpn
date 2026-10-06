# PowerShell Native AES-256 Project Encryption Vault
param (
    [string]$Password = "LorenzoStrictLeaderSecret2026",
    [string]$OutputFile = "encrypted-vault\lorenzo_project_vault.aes"
)

$ErrorActionPreference = "Stop"

Write-Host "==================================================================" -ForegroundColor Cyan
Write-Host "       👑 LORENZO VPN - AES-256 ENCRYPTED VAULT GENERATOR 👑      " -ForegroundColor Yellow
Write-Host "==================================================================" -ForegroundColor Cyan

# 1. Create temporary zip of source code
$VaultDir = "encrypted-vault"
if (!(Test-Path $VaultDir)) {
    New-Item -ItemType Directory -Path $VaultDir | Out-Null
}

$TempZip = Join-Path $VaultDir "temp_source.zip"
if (Test-Path $TempZip) { Remove-Item $TempZip -Force }

Write-Host "[1/3] ضغط الكود المصدري وملفات المشروع..." -ForegroundColor Green
$Exclude = @("*.exe", "*.apk", "*.aes", ".git*", "build*", "dist*", ".dart_tool*")

# Create zip archive of source directories
Compress-Archive -Path "server", "pc-app", "lorenzo_app", "assets", "Dockerfile" -DestinationPath $TempZip -Force

# 2. Derive Key using PBKDF2 (Rfc2898DeriveBytes)
Write-Host "[2/3] توليد مفاتيح التشفير AES-256 عبر PBKDF2 (100,000 دورة)..." -ForegroundColor Green
$PlainBytes = [System.IO.File]::ReadAllBytes($TempZip)
Remove-Item $TempZip -Force

# Generate random salt
$Salt = New-Object byte[] 16
$RNG = [System.Security.Cryptography.RandomNumberGenerator]::Create()
$RNG.GetBytes($Salt)

$Rfc = New-Object System.Security.Cryptography.Rfc2898DeriveBytes($Password, $Salt, 100000, [System.Security.Cryptography.HashAlgorithmName]::SHA256)
$Key = $Rfc.GetBytes(32) # 256-bit key

# Create AES Encryptor
$Aes = [System.Security.Cryptography.Aes]::Create()
$Aes.KeySize = 256
$Aes.BlockSize = 128
$Aes.Mode = [System.Security.Cryptography.CipherMode]::CBC
$Aes.Padding = [System.Security.Cryptography.PaddingMode]::PKCS7
$Aes.Key = $Key
$Aes.GenerateIV()
$IV = $Aes.IV

$Encryptor = $Aes.CreateEncryptor()
$CipherBytes = $Encryptor.TransformFinalBlock($PlainBytes, 0, $PlainBytes.Length)

# Format: [16 bytes Salt] + [16 bytes IV] + [CipherBytes]
$OutStream = [System.IO.File]::Create($OutputFile)
$OutStream.Write($Salt, 0, $Salt.Length)
$OutStream.Write($IV, 0, $IV.Length)
$OutStream.Write($CipherBytes, 0, $CipherBytes.Length)
$OutStream.Close()

$FileSizeKB = [math]::Round((Get-Item $OutputFile).Length / 1KB, 2)
Write-Host "[3/3] تم إنشاء الحافظة المشفرة بنجاح!" -ForegroundColor Yellow
Write-Host " 📁 مسار الملف المشفر: $OutputFile ($FileSizeKB KB)" -ForegroundColor Cyan
Write-Host " 🔒 التشفير: AES-256-CBC مع مفتاح مشتق من كلمة المرور السرية" -ForegroundColor Green
Write-Host "==================================================================" -ForegroundColor Cyan
