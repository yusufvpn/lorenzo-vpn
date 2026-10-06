# PowerShell Native AES-256 Project Decryption Tool
param (
    [string]$Password = "LorenzoStrictLeaderSecret2026",
    [string]$InputFile = "encrypted-vault\lorenzo_project_vault.aes",
    [string]$OutputFolder = "encrypted-vault\restored_source"
)

$ErrorActionPreference = "Stop"

if (!(Test-Path $InputFile)) {
    Write-Error "الملف المشفر غير موجود: $InputFile"
    exit 1
}

Write-Host "==================================================================" -ForegroundColor Cyan
Write-Host "       👑 LORENZO VPN - AES-256 VAULT DECRYPTOR 👑                " -ForegroundColor Yellow
Write-Host "==================================================================" -ForegroundColor Cyan

$InStream = [System.IO.File]::OpenRead($InputFile)

# Read Salt (16 bytes)
$Salt = New-Object byte[] 16
$InStream.Read($Salt, 0, 16) | Out-Null

# Read IV (16 bytes)
$IV = New-Object byte[] 16
$InStream.Read($IV, 0, 16) | Out-Null

# Read CipherBytes
$CipherLen = $InStream.Length - 32
$CipherBytes = New-Object byte[] $CipherLen
$InStream.Read($CipherBytes, 0, $CipherLen) | Out-Null
$InStream.Close()

# Derive Key using PBKDF2
$Rfc = New-Object System.Security.Cryptography.Rfc2898DeriveBytes($Password, $Salt, 100000, [System.Security.Cryptography.HashAlgorithmName]::SHA256)
$Key = $Rfc.GetBytes(32)

# Create AES Decryptor
$Aes = [System.Security.Cryptography.Aes]::Create()
$Aes.KeySize = 256
$Aes.BlockSize = 128
$Aes.Mode = [System.Security.Cryptography.CipherMode]::CBC
$Aes.Padding = [System.Security.Cryptography.PaddingMode]::PKCS7
$Aes.Key = $Key
$Aes.IV = $IV

$Decryptor = $Aes.CreateDecryptor()
try {
    $PlainBytes = $Decryptor.TransformFinalBlock($CipherBytes, 0, $CipherBytes.Length)
} catch {
    Write-Host "❌ فشل فك التشفير: كلمة المرور غير صحيحة أو الملف تالف!" -ForegroundColor Red
    exit 1
}

$TempZip = "encrypted-vault\temp_restore.zip"
[System.IO.File]::WriteAllBytes($TempZip, $PlainBytes)

if (Test-Path $OutputFolder) { Remove-Item $OutputFolder -Recurse -Force }
New-Item -ItemType Directory -Path $OutputFolder | Out-Null
Expand-Archive -Path $TempZip -DestinationPath $OutputFolder -Force
Remove-Item $TempZip -Force

Write-Host "✅ تم فك التشفير واستعادة الكود بنجاح في: $OutputFolder" -ForegroundColor Green
Write-Host "==================================================================" -ForegroundColor Cyan
