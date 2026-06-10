# Benchmark de entrenamiento concurrente - riesgo-delictivo
# Mide tiempo total de ejecución variando la cantidad de workers.
# Cada configuración se repite N veces para promediar.

param(
    [int[]]$Workers = @(1, 2, 4, 8, 12, 16, 20, 24),
    [int]$Runs = 5,
    [int]$Epochs = 300,
    [string]$DataPath = "..\data\datos_limpios.csv",
    [string]$ModelPath = "..\data\model.json"
)

$ErrorActionPreference = "Stop"
$startDir = Get-Location

# 1. Compilar una sola vez
Write-Host "=== Compilando trainer.exe ==="
Set-Location -LiteralPath $PSScriptRoot
$build = go build -o trainer.exe .\cmd\trainer 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR de compilación:`n$build" -ForegroundColor Red
    Set-Location -LiteralPath $startDir
    exit 1
}
Write-Host "Compilado OK`n"

# Verificar que existe el archivo de datos
if (-not (Test-Path -LiteralPath $DataPath)) {
    Write-Host "ERROR: No se encuentra '$DataPath'" -ForegroundColor Red
    Write-Host "Directorio actual: $(Get-Location)"
    Write-Host "Ruta absoluta esperada: $(Resolve-Path $DataPath -ErrorAction SilentlyContinue)"
    Set-Location -LiteralPath $startDir
    exit 1
}

# 2. Benchmark
$results = @()
$totalRuns = $Workers.Count * $Runs
$current = 0

foreach ($w in $Workers) {
    Write-Host "--- Workers: $w ---" -ForegroundColor Cyan
    for ($run = 1; $run -le $Runs; $run++) {
        $current++
        $msg = "[$current/$totalRuns] workers=$w  run=$run/$Runs"
        Write-Host $msg -NoNewline

        $sw = [System.Diagnostics.Stopwatch]::StartNew()
        $output = & .\trainer.exe -datos $DataPath -modelo $ModelPath -workers $w -epocas $Epochs 2>&1
        $sw.Stop()
        $exitCode = $LASTEXITCODE

        if ($exitCode -ne 0) {
            Write-Host "`nERROR: trainer salió con código $exitCode" -ForegroundColor Red
            Write-Host ($output | Out-String)
            Set-Location -LiteralPath $startDir
            exit 1
        }

        # Extraer tiempos parciales (null-safe)
        # La línea de carga:    [carga    ] ... en 95ms
        # La línea de entrena:  [entrena  ] 10 épocas con 4 workers en 2ms
        $cargaLines   = @($output | Select-String "\[carga")
        $entrenaLines = @($output | Select-String "workers en")

        $cargaStr   = if ($cargaLines.Count -gt 0)   { $cargaLines[0].ToString() }   else { "" }
        $entrenaStr = if ($entrenaLines.Count -gt 0) { $entrenaLines[0].ToString() } else { "" }

        $cargaMs   = $null
        $entrenaMs = $null
        if ($cargaStr -match "([\d.]+)s\b") {
            $cargaMs = [math]::Round([double]$matches[1] * 1000, 1)
        } elseif ($cargaStr -match "([\d.]+)ms") {
            $cargaMs = [math]::Round([double]$matches[1], 1)
        }
        if ($entrenaStr -match "([\d.]+)s\b") {
            $entrenaMs = [math]::Round([double]$matches[1] * 1000, 1)
        } elseif ($entrenaStr -match "([\d.]+)ms") {
            $entrenaMs = [math]::Round([double]$matches[1], 1)
        }

        $results += [PSCustomObject]@{
            Workers      = $w
            Run          = $run
            Epochs       = $Epochs
            TotalMs      = $sw.ElapsedMilliseconds
            CargaMs      = if ($cargaMs) { $cargaMs } else { "N/A" }
            EntrenaMs    = if ($entrenaMs) { $entrenaMs } else { "N/A" }
        }

        Write-Host " -> $($sw.ElapsedMilliseconds)ms total"
    }
    Write-Host ""
}

# 3. Guardar resultados
$csvPath = Join-Path $PSScriptRoot "benchmark_results.csv"
$results | Export-Csv -LiteralPath $csvPath -NoTypeInformation -Encoding UTF8
Set-Location -LiteralPath $startDir

# 4. Resumen
Write-Host "=== Resultados guardados en benchmark_results.csv ===" -ForegroundColor Green
Write-Host ""
$results | Format-Table Workers, Run, TotalMs, CargaMs, EntrenaMs -AutoSize

# 5. Promedios por cantidad de workers
Write-Host "=== Promedios por workers ===" -ForegroundColor Yellow
$results | Group-Object Workers | ForEach-Object {
    $avg = ($_.Group | Measure-Object TotalMs -Average).Average
    $min = ($_.Group | Measure-Object TotalMs -Minimum).Minimum
    $max = ($_.Group | Measure-Object TotalMs -Maximum).Maximum
    $baseAvg = ($results | Where-Object Workers -EQ 1 | Measure-Object TotalMs -Average).Average
    [PSCustomObject]@{
        Workers   = $_.Name
        AvgMs     = [math]::Round($avg, 0)
        MinMs     = $min
        MaxMs     = $max
        Speedup   = if ($_.Name -eq 1) { 1.0 } else { [math]::Round($baseAvg / $avg, 2) }
    }
} | Format-Table -AutoSize

Write-Host "Listo. Abrí benchmark_results.csv en Excel/Python para graficar."
