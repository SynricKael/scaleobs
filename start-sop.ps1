# 服务器运维平台 - 开机启动脚本
# 启动 Gateway 后端 + Tauri 桌面客户端

$ObserverDir = "E:\ProgramOps\Observer"
$GatewayExe = "$ObserverDir\gateway-release.exe"
$ClientExe = "$ObserverDir\dashboard\src-tauri\target\release\sop-client.exe"
$LogFile = "$ObserverDir\startup.log"

function Log {
    param([string]$Msg)
    $time = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    "$time $Msg" | Out-File -FilePath $LogFile -Append -Encoding UTF8
}

Log "=== SOP Platform Starting ==="

# 启动 Gateway (后台, 无窗口)
if (Test-Path $GatewayExe) {
    $env:CONFIG_PATH = "$ObserverDir\config\services.yml"
    $gw = Start-Process -FilePath $GatewayExe -WindowStyle Hidden -WorkingDirectory $ObserverDir -PassThru
    Log "Gateway started (PID: $($gw.Id)) with CONFIG_PATH=$env:CONFIG_PATH"
} else {
    Log "WARNING: Gateway not found at $GatewayExe"
}

# 等待 Gateway 启动
Start-Sleep -Seconds 2

# 启动 Tauri 客户端
if (Test-Path $ClientExe) {
    $client = Start-Process -FilePath $ClientExe -WindowStyle Normal -PassThru
    Log "SOP Client started (PID: $($client.Id))"
} else {
    Log "WARNING: SOP Client not found at $ClientExe"
}

Log "Startup complete."
