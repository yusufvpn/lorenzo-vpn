$WshShell = New-Object -ComObject WScript.Shell
$DesktopPath = [System.Environment]::GetFolderPath('Desktop')
$ShortcutPath = Join-Path $DesktopPath "Lorenzo VPN.lnk"

$Shortcut = $WshShell.CreateShortcut($ShortcutPath)
$Shortcut.TargetPath = "C:\Users\yusuf\Desktop\vpn\dist\windows\LorenzoVPN.exe"
$Shortcut.WorkingDirectory = "C:\Users\yusuf\Desktop\vpn\dist\windows"
$Shortcut.IconLocation = "C:\Users\yusuf\Desktop\vpn\dist\assets\lorenzo.ico,0"
$Shortcut.Description = "Lorenzo VPN - The Strict Leader"
$Shortcut.Save()

Write-Host "SUCCESS: Desktop shortcut created at $ShortcutPath"
