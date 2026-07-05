# Evaluacion experimental PC4 - riesgo-delictivo
# Mide: latencia de predicciones, throughput, comparacion PC3 vs PC4

param(
    [string]$ApiUrl = "http://localhost:8080",
    [int]$Samples = 100,
    [int]$EpochsBench = 100
)

$results = @()

# ============================================================
# 1. Latencia de /predecir (sin cache - primera consulta)
# ============================================================
Write-Host "=== Latencia /predecir (sin cache) ===" -ForegroundColor Cyan
$times = @()
for ($i = 0; $i -lt $Samples; $i++) {
    $h = Get-Random -Min 0 -Max 24
    $b = Get-Random -Min 0 -Max 48
    $d = Get-Random -Min 0 -Max 7
    $sw = [System.Diagnostics.Stopwatch]::StartNew()
    try { $null = Invoke-RestMethod -Uri "$ApiUrl/predecir?hora=$h&barrio_id=$b&dia_semana=$d" -TimeoutSec 5 }
    catch { Write-Host "x" -NoNewline; continue }
    $sw.Stop()
    $times += $sw.ElapsedMilliseconds
    if ($i % 20 -eq 0) { Write-Host "." -NoNewline }
}
Write-Host ""
$sorted = $times | Sort-Object
$avg = ($times | Measure-Object -Average).Average
$p50 = $sorted[[math]::Floor($sorted.Count * 0.5)]
$p95 = $sorted[[math]::Floor($sorted.Count * 0.95)]
$p99 = $sorted[[math]::Floor($sorted.Count * 0.99)]

Write-Host "Sin cache: avg=${avg}ms p50=${p50}ms p95=${p95}ms p99=${p99}ms samples=$($times.Count)"

$results += [PSCustomObject]@{
    Test = "latencia_sin_cache"
    AvgMs = [math]::Round($avg, 1)
    P50Ms = $p50
    P95Ms = $p95
    P99Ms = $p99
    Samples = $times.Count
}

# ============================================================
# 2. Latencia /predecir (con cache - segunda consulta identica)
# ============================================================
Write-Host "`n=== Latencia /predecir (con cache Redis) ===" -ForegroundColor Cyan
$times = @()
for ($i = 0; $i -lt $Samples; $i++) {
    $h = Get-Random -Min 0 -Max 24
    $b = Get-Random -Min 0 -Max 48
    $d = Get-Random -Min 0 -Max 7
    # Primera: llena el cache
    try { $null = Invoke-RestMethod -Uri "$ApiUrl/predecir?hora=$h&barrio_id=$b&dia_semana=$d" -TimeoutSec 5 } catch {}
    Start-Sleep -Milliseconds 50
    # Segunda: cache hit
    $sw = [System.Diagnostics.Stopwatch]::StartNew()
    try { $null = Invoke-RestMethod -Uri "$ApiUrl/predecir?hora=$h&barrio_id=$b&dia_semana=$d" -TimeoutSec 5 } catch {}
    $sw.Stop()
    $times += $sw.ElapsedMilliseconds
    if ($i % 20 -eq 0) { Write-Host "." -NoNewline }
}
Write-Host ""
$sorted = $times | Sort-Object
$avg = ($times | Measure-Object -Average).Average
$p50 = $sorted[[math]::Floor($sorted.Count * 0.5)]
$p95 = $sorted[[math]::Floor($sorted.Count * 0.95)]

Write-Host "Con cache: avg=${avg}ms p50=${p50}ms p95=${p95}ms samples=$($times.Count)"

$results += [PSCustomObject]@{
    Test = "latencia_con_cache"
    AvgMs = [math]::Round($avg, 1)
    P50Ms = $p50
    P95Ms = $p95
    P99Ms = "N/A"
    Samples = $times.Count
}

# ============================================================
# 3. Latencia /predicciones (todos los barrios)
# ============================================================
Write-Host "`n=== Latencia /predicciones (48 barrios juntos) ===" -ForegroundColor Cyan
$times = @()
for ($i = 0; $i -lt 20; $i++) {
    $h = Get-Random -Min 0 -Max 24
    $d = Get-Random -Min 0 -Max 7
    $sw = [System.Diagnostics.Stopwatch]::StartNew()
    try { $null = Invoke-RestMethod -Uri "$ApiUrl/predicciones?hora=$h&dia_semana=$d" -TimeoutSec 5 } catch {}
    $sw.Stop()
    $times += $sw.ElapsedMilliseconds
}
$avg = ($times | Measure-Object -Average).Average
Write-Host "/predicciones: avg=${avg}ms"

$results += [PSCustomObject]@{
    Test = "latencia_predicciones_todas"
    AvgMs = [math]::Round($avg, 1)
    P50Ms = "N/A"
    P95Ms = "N/A"
    P99Ms = "N/A"
    Samples = $times.Count
}

# ============================================================
# 4. Throughput (predicciones por segundo)
# ============================================================
Write-Host "`n=== Throughput (predicciones/segundo) ===" -ForegroundColor Cyan
$duration = 10
$count = 0
$sw = [System.Diagnostics.Stopwatch]::StartNew()
while ($sw.Elapsed.TotalSeconds -lt $duration) {
    $h = Get-Random -Min 0 -Max 24
    $b = Get-Random -Min 0 -Max 48
    $d = Get-Random -Min 0 -Max 7
    try { $null = Invoke-RestMethod -Uri "$ApiUrl/predecir?hora=$h&barrio_id=$b&dia_semana=$d" -TimeoutSec 3 } catch { continue }
    $count++
}
$sw.Stop()
$qps = [math]::Round($count / $sw.Elapsed.TotalSeconds, 1)
Write-Host "Throughput: $count predicciones en $([math]::Round($sw.Elapsed.TotalSeconds,1))s = ${qps} qps"

$results += [PSCustomObject]@{
    Test = "throughput_qps"
    AvgMs = $qps
    P50Ms = "N/A"
    P95Ms = "N/A"
    P99Ms = "N/A"
    Samples = $count
}

# ============================================================
# 5. Guardar resultados
# ============================================================
$csvPath = Join-Path $PSScriptRoot "evaluacion_experimental.csv"
$results | Export-Csv -LiteralPath $csvPath -NoTypeInformation -Encoding UTF8
Write-Host "`n=== Resultados guardados en evaluacion_experimental.csv ===" -ForegroundColor Green
$results | Format-Table -AutoSize
