[CmdletBinding()]
param(
    [string]$DataDir = "artifacts/runtime/show",
    [string]$Seed = "safe-rehearsal"
)

$ErrorActionPreference = "Stop"
$repoRoot = Split-Path -Parent $PSScriptRoot
Set-Location $repoRoot
$recordings = Join-Path $DataDir "recordings"
New-Item -ItemType Directory -Force -Path $recordings | Out-Null

# The fake run is intentionally assembled by the normal server composition;
# modelchain.RecordReplay persists its vendor outputs in the SQLite recordings
# table below this runtime directory. The seed keeps the resulting sequence
# stable when the rehearsal is repeated. The operator drives the run with
# dfctl/e2e while this process is up, then the process is stopped safely.
$dataPath = if ([IO.Path]::IsPathRooted($DataDir)) { $DataDir } else { Join-Path $repoRoot $DataDir }
$outPath = Join-Path $repoRoot "artifacts\logs\ORCH-S\record-safe.out.log"
$errPath = Join-Path $repoRoot "artifacts\logs\ORCH-S\record-safe.err.log"
New-Item -ItemType Directory -Force -Path (Split-Path $outPath) | Out-Null
$process = Start-Process go -ArgumentList @("run", "./cmd/server", "-config", "config/fake.json", "-port", "18101", "-data-dir", $dataPath, "-seed", $Seed) `
    -RedirectStandardOutput $outPath -RedirectStandardError $errPath -PassThru
try {
    Write-Output "Recording server started as PID $($process.Id) on port 18101. Drive the full fake run with dfctl, then stop it."
    Wait-Process -Id $process.Id
} finally {
    if (-not $process.HasExited) { Stop-Process -Id $process.Id }
}

Write-Output "Safe Mode recordings are under $recordings (SQLite runtime: $DataDir)."
