param(
    [string]$Runnora = (Join-Path $PSScriptRoot '..\..\..\..\runnora\runnora.exe'),
    [string]$Ddq = (Join-Path $PSScriptRoot '..\..\..\..\design-doc-quarto-template\cli\target\release\ddq.exe'),
    [ValidateSet('Both', 'Html', 'Pdf')]
    [string]$Format = 'Both',
    [switch]$UpdateTemplate
)
$ErrorActionPreference = 'Stop'
$Runnora = (Resolve-Path -LiteralPath $Runnora).Path
$Ddq = (Resolve-Path -LiteralPath $Ddq).Path
$sample = Split-Path $PSScriptRoot -Parent
$repo = (Resolve-Path (Join-Path $sample '..\..')).Path
Push-Location $repo
try {
    New-Item -ItemType Directory -Force "$sample/bin" | Out-Null
    go build -o "$sample/bin/runnora-docgen.exe" .
    if ($LASTEXITCODE -ne 0) { throw 'Document generator build failed' }
} finally {
    Pop-Location
}
Push-Location $sample
try {
    # Baseline generation shows what OpenAPI alone produces, before workflow authoring.
    & $Runnora generate --config config.yaml --openapi openapi.yaml --out baseline --tags inventory,orders --server http://127.0.0.1:18081 --emit-response-example --force
    if ($LASTEXITCODE -ne 0) { throw 'OpenAPI baseline generation failed' }
    & ./bin/runnora-docgen.exe generate --base-dir . --config config.yaml --out docs/generated --force runbooks/order-lifecycle.yml runbooks/insufficient-stock.yml runbooks/double-cancel.yml runbooks/stock-recovery.yml runbooks/invalid-recovery.yml
    if ($LASTEXITCODE -ne 0) { throw 'QMD generation failed' }
    # ddq owns its mechanism files. Explicitly update when changing ddq versions.
    if ($UpdateTemplate -or -not (Test-Path -LiteralPath 'docs/.template-version')) {
        & $Ddq update docs
        if ($LASTEXITCODE -ne 0) { throw 'Template update failed' }
    }
    # Quarto recreates _book for each format. Build PDF first (ddq copies it
    # to docs/design-doc.pdf), then leave the chapter HTML under _book.
    if ($Format -in @('Both', 'Pdf')) {
        & $Ddq pdf docs
        if ($LASTEXITCODE -ne 0) { throw 'ddq PDF build failed' }
    }
    if ($Format -in @('Both', 'Html')) {
        & $Ddq html docs
        if ($LASTEXITCODE -ne 0) { throw 'ddq HTML build failed' }
    }
} finally {
    Pop-Location
}
