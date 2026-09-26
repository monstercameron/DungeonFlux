[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'
$repoRoot = Split-Path -Parent $PSScriptRoot
$wasmDir = Join-Path $repoRoot 'artifacts\wasm'
$wasmPath = Join-Path $wasmDir 'dungeonflux.wasm'
$brotliPath = "$wasmPath.br"

New-Item -ItemType Directory -Force -Path $wasmDir | Out-Null
Remove-Item -LiteralPath $wasmPath, $brotliPath -Force -ErrorAction SilentlyContinue

$oldGOOS = $env:GOOS
$oldGOARCH = $env:GOARCH
try {
    $env:GOOS = 'js'
    $env:GOARCH = 'wasm'
    & go build -o $wasmPath (Join-Path $repoRoot 'web\shell')
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

$brotli = Get-Command brotli -ErrorAction SilentlyContinue
if ($null -ne $brotli) {
    & $brotli.Source --force --quality=5 --output=$brotliPath $wasmPath
    if ($LASTEXITCODE -ne 0) { throw "brotli failed with exit code $LASTEXITCODE" }
    Write-Host "Created $brotliPath"
}
else {
    Write-Host 'brotli not found; skipping optional dungeonflux.wasm.br compression.'
}

Write-Host "Built $wasmPath"
