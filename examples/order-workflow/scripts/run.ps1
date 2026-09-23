param(
    [string]$Runnora = (Join-Path $PSScriptRoot '..\..\..\..\runnora\runnora.exe')
)
$ErrorActionPreference = 'Stop'
$Runnora = (Resolve-Path -LiteralPath $Runnora).Path
$sample = Split-Path $PSScriptRoot -Parent
Push-Location $sample
try {
    $stamp = Get-Date -Format 'yyyyMMdd-HHmmss-fff'
    $reportDir = Join-Path 'reports' $stamp
    New-Item -ItemType Directory -Force $reportDir | Out-Null
    # Run sequentially: each scenario resets the same in-memory sample shop.
    foreach ($name in @('order-lifecycle', 'insufficient-stock', 'double-cancel', 'stock-recovery', 'invalid-recovery')) {
        & $Runnora run --config config.yaml --report-format text --report-out "$reportDir/$name.txt" "runbooks/$name.yml"
        $runExit = $LASTEXITCODE
        if ($runExit -ne 0) {
            throw "$name failed (exit $runExit). See $reportDir."
        }
    }
    Write-Host "All 5 scenarios passed. Reports: $sample/$reportDir"
} finally {
    Pop-Location
}
