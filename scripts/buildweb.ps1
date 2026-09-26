[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'
$repoRoot = Split-Path -Parent $PSScriptRoot
$wasmDir = Join-Path $repoRoot 'artifacts\wasm'
$wasmPath = Join-Path $wasmDir 'dungeonflux.wasm'
$gzipPath = "$wasmPath.gz"
$brotliPath = "$wasmPath.br"
$compressor = Join-Path $PSScriptRoot 'buildweb'

New-Item -ItemType Directory -Force -Path $wasmDir | Out-Null
Remove-Item -LiteralPath $wasmPath, $gzipPath, $brotliPath -Force -ErrorAction SilentlyContinue

$oldGOOS = $env:GOOS
$oldGOARCH = $env:GOARCH
try {
    $env:GOOS = 'js'
    $env:GOARCH = 'wasm'
    & go build -trimpath -ldflags='-s -w' -o $wasmPath (Join-Path $repoRoot 'web\shell')
    if ($LASTEXITCODE -ne 0) {
        throw "go build failed with exit code $LASTEXITCODE"
    }
}
finally {
    if ($null -eq $oldGOOS) { Remove-Item Env:GOOS -ErrorAction SilentlyContinue } else { $env:GOOS = $oldGOOS }
    if ($null -eq $oldGOARCH) { Remove-Item Env:GOARCH -ErrorAction SilentlyContinue } else { $env:GOARCH = $oldGOARCH }
}

$wasmExec = Join-Path (& go env GOROOT) 'lib\wasm\wasm_exec.js'
if ($LASTEXITCODE -ne 0 -or -not (Test-Path -LiteralPath $wasmExec)) {
    throw "wasm_exec.js was not found in the building Go toolchain"
}
Copy-Item -LiteralPath $wasmExec -Destination (Join-Path $wasmDir 'wasm_exec.js') -Force

$env:GOCACHE = Join-Path $repoRoot 'artifacts\cache\go'
$env:GOTMPDIR = Join-Path $repoRoot 'artifacts\tmp\ORCH-W'
New-Item -ItemType Directory -Force -Path $env:GOCACHE, $env:GOTMPDIR | Out-Null
& go run $compressor -input $wasmPath -output $gzipPath
if ($LASTEXITCODE -ne 0) { throw "gzip compression failed with exit code $LASTEXITCODE" }

$rawSize = (Get-Item -LiteralPath $wasmPath).Length
$gzipSize = (Get-Item -LiteralPath $gzipPath).Length
Write-Host ("WASM size: {0:N0} bytes; gzip size: {1:N0} bytes ({2:P1} smaller)" -f $rawSize, $gzipSize, (1 - ($gzipSize / $rawSize)))
Write-Host 'Brotli encoder is not vendored; skipping optional dungeonflux.wasm.br compression.'

Write-Host "Built $wasmPath"
