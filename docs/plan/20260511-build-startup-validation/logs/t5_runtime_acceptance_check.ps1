$ErrorActionPreference = 'Stop'
Set-Location 'E:/projects/golang/work/src/github.com/zhiyunliu/distributed-workflow'

$logDir = 'docs/plan/20260511-build-startup-validation/logs'
New-Item -ItemType Directory -Force -Path $logDir | Out-Null
$stamp = Get-Date -Format 'yyyyMMdd-HHmmss'
$backendOutLog = Join-Path $logDir "t5-backend-$stamp.out.log"
$backendErrLog = Join-Path $logDir "t5-backend-$stamp.err.log"
$engineOutLog = Join-Path $logDir "t5-engine-$stamp.out.log"
$engineErrLog = Join-Path $logDir "t5-engine-$stamp.err.log"
$workerOutLog = Join-Path $logDir "t5-worker-$stamp.out.log"
$workerErrLog = Join-Path $logDir "t5-worker-$stamp.err.log"

Set-Location 'config-management/backend'
go build ./cmd/api
$be = $LASTEXITCODE
Set-Location '../../'

Set-Location 'runtime-execution'
go build ./cmd/engine
$en = $LASTEXITCODE
go build ./cmd/worker
$wk = $LASTEXITCODE
Set-Location '../'

$dep1433 = [bool](Get-NetTCPConnection -State Listen -LocalPort 1433 -ErrorAction SilentlyContinue)
$dep6379 = [bool](Get-NetTCPConnection -State Listen -LocalPort 6379 -ErrorAction SilentlyContinue)
if (-not ($dep1433 -and $dep6379)) {
  throw "dependency ports not ready: 1433=$dep1433,6379=$dep6379"
}

$saPwd = 'P@ssw0rd12345!'
$jwt = 'dev-jwt-secret-20260511'
$dbDsn = "sqlserver://sa:$saPwd@localhost:1433?database=master&encrypt=disable"
$redis = 'localhost:6379'

$backendCmd = "Set-Location 'E:/projects/golang/work/src/github.com/zhiyunliu/distributed-workflow/config-management/backend'; `$env:DB_DSN='$dbDsn'; `$env:JWT_SECRET='$jwt'; `$env:HTTP_ADDR=':7080'; go run ./cmd/api"
$engineCmd = "Set-Location 'E:/projects/golang/work/src/github.com/zhiyunliu/distributed-workflow/runtime-execution'; `$env:DB_DSN='$dbDsn'; `$env:REDIS_ADDR='$redis'; `$env:GRPC_ADDR=':50051'; `$env:HTTP_ADDR=':8081'; go run ./cmd/engine"
$workerCmd = "Set-Location 'E:/projects/golang/work/src/github.com/zhiyunliu/distributed-workflow/runtime-execution'; `$env:ENGINE_ADDR='127.0.0.1:50051'; `$env:GRPC_ADDR=':50052'; go run ./cmd/worker"

$bp = Start-Process pwsh -ArgumentList '-NoProfile','-Command',$backendCmd -RedirectStandardOutput $backendOutLog -RedirectStandardError $backendErrLog -PassThru
$ep = Start-Process pwsh -ArgumentList '-NoProfile','-Command',$engineCmd -RedirectStandardOutput $engineOutLog -RedirectStandardError $engineErrLog -PassThru
$wp = Start-Process pwsh -ArgumentList '-NoProfile','-Command',$workerCmd -RedirectStandardOutput $workerOutLog -RedirectStandardError $workerErrLog -PassThru

Start-Sleep -Seconds 6

$ports = @(7080, 8081, 50051, 50052)
$listeners = @()
foreach ($p in $ports) {
  $c = Get-NetTCPConnection -State Listen -LocalPort $p -ErrorAction SilentlyContinue
  $listeners += [pscustomobject]@{
    port = $p
    listening = [bool]$c
    pid = if ($c) { $c[0].OwningProcess } else { $null }
  }
}
$portsPass = -not ($listeners | Where-Object { -not $_.listening })

$alive60 = $true
 $aliveFailures = @()
foreach ($procId in @($bp.Id, $ep.Id, $wp.Id)) {
  try {
    Wait-Process -Id $procId -Timeout 60 -ErrorAction Stop
    $alive60 = $false
    $aliveFailures += "process_exited_within_60s:$procId"
  } catch {
    if ($_.Exception.Message -match 'Cannot find a process') {
      $alive60 = $false
      $aliveFailures += "process_not_found_before_60s:$procId"
      continue
    }
    if ($_.Exception.Message -notmatch 'has not been terminated') {
      $alive60 = $false
      $aliveFailures += "wait_process_unexpected_error:${procId}:$($_.Exception.Message)"
      continue
    }
  }
}

$fatal = Select-String -Path $backendOutLog,$backendErrLog,$engineOutLog,$engineErrLog,$workerOutLog,$workerErrLog -Pattern 'fatal|panic' -CaseSensitive:$false -ErrorAction SilentlyContinue
$noFatal = -not $fatal

$result = [pscustomobject]@{
  timestamp = (Get-Date).ToString('o')
  go_compile = @(
    @{ component = 'backend'; exit = $be },
    @{ component = 'engine'; exit = $en },
    @{ component = 'worker'; exit = $wk }
  )
  go_compile_all_pass = (($be -eq 0) -and ($en -eq 0) -and ($wk -eq 0))
  dependency_ports = @{ port1433 = $dep1433; port6379 = $dep6379 }
  pids = @{ backend = $bp.Id; engine = $ep.Id; worker = $wp.Id }
  listeners = $listeners
  listeners_pass = $portsPass
  alive_ge_60s = $alive60
  alive_failures = $aliveFailures
  no_fatal_panic = $noFatal
  fatal_samples = if ($fatal) { $fatal | Select-Object -First 5 | ForEach-Object { $_.Line } } else { @() }
  logs = @($backendOutLog, $backendErrLog, $engineOutLog, $engineErrLog, $workerOutLog, $workerErrLog)
}

$resultPath = Join-Path $logDir "t5-runtime-acceptance-$stamp.json"
$result | ConvertTo-Json -Depth 6 | Set-Content -Path $resultPath -Encoding UTF8

foreach ($p in @($bp, $ep, $wp)) {
  try {
    if ($p -and -not $p.HasExited) {
      Stop-Process -Id $p.Id -Force -ErrorAction SilentlyContinue
    }
  } catch {}
}

Write-Output '==T5_RUNTIME_ACCEPTANCE=='
Get-Content $resultPath
