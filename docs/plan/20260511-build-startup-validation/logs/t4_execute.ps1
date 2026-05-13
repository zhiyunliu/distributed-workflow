$ErrorActionPreference = 'Stop'
Set-Location "E:/projects/golang/work/src/github.com/zhiyunliu/distributed-workflow"

$planLogDir = "docs/plan/20260511-build-startup-validation/logs"
New-Item -ItemType Directory -Force -Path $planLogDir | Out-Null

$saPwd = "P@ssw0rd12345!"
$jwtSecret = "dev-jwt-secret-20260511"
$dbDsn = "sqlserver://sa:$saPwd@localhost:1433?database=master&encrypt=disable"
$redisAddr = "localhost:6379"

$reproCommands = @(
  "docker run -d --name dw-sqlserver -e ACCEPT_EULA=Y -e MSSQL_SA_PASSWORD='$saPwd' -p 1433:1433 mcr.microsoft.com/mssql/server:2022-latest",
  "docker run -d --name dw-redis -p 6379:6379 redis:7-alpine",
  "Set-Location config-management/backend; `$env:DB_DSN='$dbDsn'; `$env:JWT_SECRET='$jwtSecret'; `$env:HTTP_ADDR=':7080'; go run ./cmd/api",
  "Set-Location runtime-execution; `$env:DB_DSN='$dbDsn'; `$env:REDIS_ADDR='$redisAddr'; `$env:GRPC_ADDR=':50051'; `$env:HTTP_ADDR=':8081'; go run ./cmd/engine",
  "Set-Location runtime-execution; `$env:ENGINE_ADDR='127.0.0.1:50051'; `$env:GRPC_ADDR=':50052'; go run ./cmd/worker"
)

function Ensure-ContainerRunning {
  param([string]$Name, [string]$RunCommand)
  $running = docker ps --format "{{.Names}}" | Select-String -SimpleMatch $Name
  if (-not $running) {
    $exists = docker ps -a --format "{{.Names}}" | Select-String -SimpleMatch $Name
    if ($exists) {
      docker start $Name | Out-Null
    } else {
      Invoke-Expression $RunCommand | Out-Null
    }
  }
}

function Stop-IfRunning {
  param([System.Diagnostics.Process]$Proc)
  if ($null -ne $Proc) {
    try {
      if (-not $Proc.HasExited) {
        Stop-Process -Id $Proc.Id -Force -ErrorAction SilentlyContinue
      }
    } catch {}
  }
}

function Kill-PortOwners {
  param([int[]]$Ports)
  foreach ($port in $Ports) {
    $conns = Get-NetTCPConnection -LocalPort $port -State Listen -ErrorAction SilentlyContinue
    if ($conns) {
      foreach ($c in $conns) {
        if ($c.OwningProcess -and $c.OwningProcess -ne 0) {
          Stop-Process -Id $c.OwningProcess -Force -ErrorAction SilentlyContinue
        }
      }
    }
  }
}

function Wait-Alive {
  param([int]$Pid, [int]$TimeoutSec)
  try {
    Wait-Process -Id $Pid -Timeout $TimeoutSec -ErrorAction Stop
    return $false
  } catch {
    if ($_.Exception.Message -match "has not been terminated") {
      return $true
    }
    throw
  }
}

$maxAttempts = 3
$attempt = 0
$final = $null

while ($attempt -lt $maxAttempts) {
  $attempt++
  $stamp = Get-Date -Format "yyyyMMdd-HHmmss"
  $backendLog = Join-Path $planLogDir "t4-backend-attempt$attempt-$stamp.log"
  $engineLog  = Join-Path $planLogDir "t4-engine-attempt$attempt-$stamp.log"
  $workerLog  = Join-Path $planLogDir "t4-worker-attempt$attempt-$stamp.log"

  $backendProc = $null
  $engineProc = $null
  $workerProc = $null

  try {
    Ensure-ContainerRunning -Name "dw-sqlserver" -RunCommand "docker run -d --name dw-sqlserver -e ACCEPT_EULA=Y -e MSSQL_SA_PASSWORD='$saPwd' -p 1433:1433 mcr.microsoft.com/mssql/server:2022-latest"
    Ensure-ContainerRunning -Name "dw-redis" -RunCommand "docker run -d --name dw-redis -p 6379:6379 redis:7-alpine"

    $depPorts = @(1433, 6379)
    foreach ($p in $depPorts) {
      $listen = Get-NetTCPConnection -LocalPort $p -State Listen -ErrorAction SilentlyContinue
      if (-not $listen) {
        throw "dependency port not listening: $p"
      }
    }

    $backendCmd = "Set-Location 'E:/projects/golang/work/src/github.com/zhiyunliu/distributed-workflow/config-management/backend'; `$env:DB_DSN='$dbDsn'; `$env:JWT_SECRET='$jwtSecret'; `$env:HTTP_ADDR=':7080'; go run ./cmd/api"
    $engineCmd = "Set-Location 'E:/projects/golang/work/src/github.com/zhiyunliu/distributed-workflow/runtime-execution'; `$env:DB_DSN='$dbDsn'; `$env:REDIS_ADDR='$redisAddr'; `$env:GRPC_ADDR=':50051'; `$env:HTTP_ADDR=':8081'; go run ./cmd/engine"
    $workerCmd = "Set-Location 'E:/projects/golang/work/src/github.com/zhiyunliu/distributed-workflow/runtime-execution'; `$env:ENGINE_ADDR='127.0.0.1:50051'; `$env:GRPC_ADDR=':50052'; go run ./cmd/worker"

    $backendProc = Start-Process -FilePath "pwsh" -ArgumentList "-NoProfile","-Command",$backendCmd -RedirectStandardOutput $backendLog -RedirectStandardError $backendLog -PassThru
    if (-not (Wait-Alive -Pid $backendProc.Id -TimeoutSec 12)) { throw "backend exited early" }

    $engineProc = Start-Process -FilePath "pwsh" -ArgumentList "-NoProfile","-Command",$engineCmd -RedirectStandardOutput $engineLog -RedirectStandardError $engineLog -PassThru
    if (-not (Wait-Alive -Pid $engineProc.Id -TimeoutSec 12)) { throw "engine exited early" }

    $workerProc = Start-Process -FilePath "pwsh" -ArgumentList "-NoProfile","-Command",$workerCmd -RedirectStandardOutput $workerLog -RedirectStandardError $workerLog -PassThru
    if (-not (Wait-Alive -Pid $workerProc.Id -TimeoutSec 12)) { throw "worker exited early" }

    $svcPorts = @(7080,8081,50051,50052)
    $portResults = @()
    foreach ($p in $svcPorts) {
      $c = Get-NetTCPConnection -LocalPort $p -State Listen -ErrorAction SilentlyContinue
      if ($c) {
        $portResults += "${p}|LISTEN|$($c[0].OwningProcess)"
      } else {
        $portResults += "${p}|CLOSED|-"
        throw "service port not listening: $p"
      }
    }

    $aliveAll60 = (Wait-Alive -Pid $backendProc.Id -TimeoutSec 60) -and (Wait-Alive -Pid $engineProc.Id -TimeoutSec 60) -and (Wait-Alive -Pid $workerProc.Id -TimeoutSec 60)
    if (-not $aliveAll60) { throw "one or more services exited within 60 seconds" }

    $fatalHits = Select-String -Path $backendLog,$engineLog,$workerLog -Pattern "fatal|panic" -CaseSensitive:$false -ErrorAction SilentlyContinue
    if ($fatalHits) {
      $fatalText = ($fatalHits | Select-Object -First 5 | ForEach-Object { "[$($_.Path)] $($_.Line)" }) -join " || "
      throw "fatal/panic found: $fatalText"
    }

    $final = [pscustomobject]@{
      pass = $true
      attempt = $attempt
      dependency_ports = @("1433|LISTEN","6379|LISTEN")
      service_ports = $portResults
      logs = @($backendLog,$engineLog,$workerLog)
      repro = $reproCommands
    }
    break
  } catch {
    $errText = $_.Exception.Message

    Stop-IfRunning $backendProc
    Stop-IfRunning $engineProc
    Stop-IfRunning $workerProc

    if ($attempt -lt $maxAttempts) {
      Kill-PortOwners -Ports @(7080,8081,50051,50052)
      try { docker restart dw-sqlserver | Out-Null } catch {}
      try { docker restart dw-redis | Out-Null } catch {}
      $final = [pscustomobject]@{
        pass = $false
        attempt = $attempt
        last_error = $errText
        action = "auto-fix-applied-and-retrying"
      }
    } else {
      $final = [pscustomobject]@{
        pass = $false
        attempt = $attempt
        last_error = $errText
        logs = @($backendLog,$engineLog,$workerLog)
        repro = $reproCommands
      }
    }
  }
}

$resultPath = Join-Path $planLogDir "t4-result.json"
$final | ConvertTo-Json -Depth 6 | Set-Content -Path $resultPath -Encoding UTF8
Write-Output "==T4_RESULT=="
Get-Content -Path $resultPath