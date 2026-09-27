[CmdletBinding()]
param(
    [string]$Lane = "",
    [string]$Todo = "",
    [string[]]$Packages = @(),
    [switch]$Full
)

$ErrorActionPreference = "Stop"
$repoRoot = Split-Path -Parent $PSScriptRoot
Set-Location $repoRoot

if ($Full) {
    $reportName = "ORCH"
} elseif ([string]::IsNullOrWhiteSpace($Lane)) {
    throw "-Lane is required unless -Full is supplied."
} else {
    $reportName = $Lane
}

$artifactRoot = Join-Path $repoRoot "artifacts"
$cacheRoot = Join-Path $artifactRoot "cache\go"
$tmpRoot = Join-Path $artifactRoot "tmp\$reportName"
$coverageRoot = Join-Path $artifactRoot "coverage\$reportName"
$testRoot = Join-Path $artifactRoot "test\$reportName"
New-Item -ItemType Directory -Force -Path $cacheRoot, $tmpRoot, $coverageRoot, $testRoot | Out-Null
$env:GOCACHE = $cacheRoot
$env:GOTMPDIR = $tmpRoot
$env:TMP = $tmpRoot
$env:TEMP = $tmpRoot

$transcriptPath = Join-Path $testRoot ("gate-{0}.log" -f (Get-Date -Format "yyyyMMdd-HHmmss"))
$script:Failures = 0

function Write-GateLine {
    param([string]$Message)
    $utf8NoBom = New-Object System.Text.UTF8Encoding($false)
    [IO.File]::AppendAllText($transcriptPath, ($Message + [Environment]::NewLine), $utf8NoBom)
    Write-Output $Message
}

function Invoke-GateCommand {
    param(
        [string]$Label,
        [string]$FilePath,
        [string[]]$ArgumentList
    )
    Write-GateLine "`n>>> $Label"
    try {
        & $FilePath @ArgumentList 2>&1 | ForEach-Object { Write-GateLine $_.ToString() }
        $exitCode = $LASTEXITCODE
    } catch {
        Write-GateLine $_.Exception.Message
        $exitCode = 1
    }
    if ($null -eq $exitCode) { $exitCode = 0 }
    if ($exitCode -ne 0) {
        $script:Failures++
        Write-GateLine "FAILED ($exitCode): $Label"
        return $false
    }
    Write-GateLine "PASS: $Label"
    return $true
}

function Invoke-BuildGate {
    param([string[]]$TargetPackages)
    $label = "go build ./..."
    Write-GateLine "`n>>> $label"
    $output = @()
    try {
        $output = @(& go build ./... 2>&1 | ForEach-Object { $_.ToString() })
        foreach ($line in $output) { Write-GateLine $line }
        $exitCode = $LASTEXITCODE
    } catch {
        Write-GateLine $_.Exception.Message
        $exitCode = 1
    }
    if ($null -eq $exitCode) { $exitCode = 0 }
    if ($exitCode -eq 0) {
        Write-GateLine "PASS: $label"
        return $true
    }

    $failedPackages = @($output | ForEach-Object {
        if ($_ -match '^#\s+([^\s]+)$') { $Matches[1] }
    } | Sort-Object -Unique)
    $outside = @($failedPackages | Where-Object {
        $failed = $_
        -not (@($TargetPackages | Where-Object {
            $target = $_ -replace '^\./', ''
            $failed -eq $_ -or $failed.EndsWith("/$target")
        }).Count -gt 0)
    })
    if ($outside.Count -gt 0 -and $outside.Count -eq $failedPackages.Count) {
        Write-GateLine ("build broken outside target: {0}" -f ($outside -join ", "))
    }
    $script:Failures++
    Write-GateLine "FAILED ($exitCode): $label"
    return $false
}

function Get-TodoPaths {
    param([string]$TodoID)
    $todoFile = Join-Path $repoRoot "TODOS.md"
    if (-not (Test-Path $todoFile)) { throw "TODOS.md is required for -Todo." }
    $lines = Get-Content $todoFile
    $found = $false
    foreach ($line in $lines) {
        if ($line -match "^\s*-\s*\[[^]]+\]\s*$([regex]::Escape($TodoID))\s") { $found = $true; continue }
        if ($found -and $line -match "paths:\s*(.*)$") {
            $matches = [regex]::Matches($Matches[1], '`([^`]+)`')
            if ($matches.Count -eq 0) { throw "Todo $TodoID has no backtick-delimited paths." }
            return @($matches | ForEach-Object { $_.Groups[1].Value })
        }
        if ($found -and $line -match "^\s*-\s*\[") { break }
    }
    throw "Todo $TodoID was not found in TODOS.md."
}

function Get-GoFiles {
    param([string[]]$PathPatterns)
    $files = @()
    foreach ($pattern in $PathPatterns) {
        $normalized = $pattern.Trim() -replace '\\', '/'
        $recursive = $normalized.EndsWith('/**')
        if ($recursive) { $normalized = $normalized.Substring(0, $normalized.Length - 3).TrimEnd('/') }
        $candidate = Join-Path $repoRoot ($normalized -replace '^\./', '' -replace '/', '\')
        if ($recursive -and (Test-Path $candidate -PathType Container)) {
            $items = @(Get-ChildItem -Path $candidate -Recurse -File -Filter "*.go")
        } else {
            $items = @(Get-ChildItem -Path $candidate -File -ErrorAction SilentlyContinue)
            if ($items.Count -eq 0 -and (Test-Path $candidate -PathType Container)) {
                $items = @(Get-ChildItem -Path $candidate -Recurse -File -Filter "*.go")
            }
        }
        foreach ($item in $items) {
            if ($item.Extension -eq ".go") { $files += $item.FullName }
        }
    }
    return @($files | Sort-Object -Unique)
}

function Get-PackageDirs {
    param([string[]]$GoFiles)
    $dirs = @()
    foreach ($file in $GoFiles) {
        $dir = Split-Path -Parent $file
        if ($dir -and (Get-ChildItem -Path $dir -Filter "*.go" -File -ErrorAction SilentlyContinue)) {
            $relative = Resolve-Path -Relative -Path $dir
            $dirs += ("./" + ($relative -replace "^\.\\", "" ) -replace "\\", "/")
        }
    }
    return @($dirs | Sort-Object -Unique)
}

function Get-ChangedGoFiles {
    $paths = @()
    try { $paths += @(git diff --name-only HEAD 2>$null) } catch { }
    try { $paths += @(git show --format= --name-only HEAD 2>$null) } catch { }
    $files = @()
    foreach ($path in ($paths | Sort-Object -Unique)) {
        if ($path -and $path.EndsWith(".go") -and (Test-Path (Join-Path $repoRoot $path))) {
            $files += (Join-Path $repoRoot $path)
        }
    }
    return @($files | Sort-Object -Unique)
}

function Test-ExistingPackages {
    param([string[]]$Candidates)
    $result = @()
    foreach ($package in $Candidates) {
        $dir = Join-Path $repoRoot ($package -replace "^\./", "" -replace "/", "\")
        if (Test-Path $dir -PathType Container) {
            if (@(Get-ChildItem $dir -Filter "*.go" -File).Count -gt 0) { $result += $package }
        }
    }
    return @($result | Sort-Object -Unique)
}

function Test-ExcludedCoverage {
    param([string]$Package)
    return $Package -match "(^|/)gen(/|$)|(^|/)cmd/server$|(^|/)internal/wire$|(^|/)internal/fakes$"
}

function Invoke-LaneGate {
    param([string[]]$TargetPackages, [string[]]$TargetFiles)
    if ($TargetFiles.Count -gt 0) {
        Invoke-GateCommand "gofmt" "gofmt" (@("-l") + $TargetFiles) | Out-Null
    } else { Write-GateLine "SKIP: gofmt (no touched Go files)" }
    if ($TargetPackages.Count -gt 0) {
        Invoke-GateCommand "go vet" "go" (@("vet") + $TargetPackages) | Out-Null
        Invoke-GateCommand "staticcheck" "go" (@("tool", "staticcheck") + $TargetPackages) | Out-Null
        foreach ($package in $TargetPackages) {
            $safeName = $package -replace "[^A-Za-z0-9_.-]", "_"
            $profile = Join-Path $coverageRoot "$safeName.out"
            Invoke-GateCommand "go test coverage $package" "go" @("test", "-count=1", "-coverprofile", $profile, $package) | Out-Null
            if ((Test-Path $profile) -and -not (Test-ExcludedCoverage $package)) {
                $profileLines = @(Get-Content -LiteralPath $profile | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
                if ($profileLines.Count -eq 1 -and $profileLines[0] -match '^mode: (set|count|atomic)$') {
                    Write-GateLine "COVERAGE ${package}: N/A (no executable statements)"
                    continue
                }
                $coverText = & go tool cover -func $profile 2>&1
                $total = $coverText | Where-Object { $_ -match "^total:" } | Select-Object -Last 1
                if ($total -and $total -match "([0-9]+(?:\.[0-9]+)?)%") {
                    $coverage = [double]$Matches[1]
                    Write-GateLine ("COVERAGE {0}: {1:N1}%" -f $package, $coverage)
                    if ($coverage -lt 70.0) { $script:Failures++; Write-GateLine "FAILED: coverage below 70%" }
                } else { $script:Failures++; Write-GateLine "FAILED: no coverage total for $package" }
            } elseif (Test-ExcludedCoverage $package) {
                Write-GateLine "COVERAGE ${package}: excluded by AGENTS.md §14"
            }
        }
    } else { Write-GateLine "SKIP: vet, staticcheck, tests, coverage (no touched Go packages)" }
    $archDir = Join-Path $repoRoot "internal\archtest"
    if ((Test-Path $archDir -PathType Container) -and (@(Get-ChildItem $archDir -Filter "*.go" -File).Count -gt 0)) {
        Invoke-GateCommand "archtest" "go" @("test", "./internal/archtest") | Out-Null
    } else { Write-GateLine "SKIP: archtest (package not present)" }
    Invoke-BuildGate $TargetPackages | Out-Null
}

function Invoke-FullGate {
    Invoke-GateCommand "go build ./..." "go" @("build", "./...") | Out-Null
    $webRoot = Join-Path $repoRoot "web"
    if ((Test-Path $webRoot -PathType Container) -and (@(Get-ChildItem $webRoot -Filter "*.go" -Recurse -File).Count -gt 0)) {
        try {
            $env:GOOS = "js"; $env:GOARCH = "wasm"
            Invoke-GateCommand "GOOS=js GOARCH=wasm go build ./web/..." "go" @("build", "./web/...") | Out-Null
        } finally {
            Remove-Item Env:GOOS -ErrorAction SilentlyContinue
            Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
        }
    } else { Write-GateLine "SKIP: WASM build (web packages not present)" }
    Invoke-GateCommand "go test ./..." "go" @("test", "-count=1", "./...") | Out-Null
    $walkRoot = Join-Path $repoRoot "internal\sim\walk"
    if ((Test-Path $walkRoot -PathType Container) -and (@(Get-ChildItem $walkRoot -Filter "*.go" -Recurse -File).Count -gt 0)) {
        Invoke-GateCommand "walk tests" "go" @("test", "-count=1", "./internal/sim/walk/...") | Out-Null
    } else { Write-GateLine "SKIP: walk tests (packages not present)" }
    $fullProfile = Join-Path $coverageRoot "full.out"
    Invoke-GateCommand "whole-module coverage (informational)" "go" @("test", "-count=1", "-coverprofile", $fullProfile, "./...") | Out-Null
    if (Test-Path $fullProfile) {
        $fullCoverText = & go tool cover -func $fullProfile 2>&1
        $fullTotal = $fullCoverText | Where-Object { $_ -match "^total:" } | Select-Object -Last 1
        if ($fullTotal) { Write-GateLine "WHOLE-MODULE $fullTotal" }
    }
}

Write-GateLine "DungeonFlux gate: $reportName"
if ($Full) {
    Invoke-FullGate
} else {
    if ($Todo) { $pathPatterns = Get-TodoPaths $Todo }
    elseif ($Packages.Count -gt 0) {
        $pathPatterns = @($Packages | ForEach-Object { $_ -split ',' } | ForEach-Object { $_.Trim() } | Where-Object { $_ })
    }
    else { $pathPatterns = @() }
    if ($pathPatterns.Count -gt 0) {
        $goFiles = Get-GoFiles $pathPatterns
    } else {
        $goFiles = Get-ChangedGoFiles
    }
    $packageDirs = Get-PackageDirs $goFiles
    $packageDirs = Test-ExistingPackages $packageDirs
    Write-GateLine ("Target packages: {0}" -f ($(if ($packageDirs.Count) { $packageDirs -join ", " } else { "none" })))
    Invoke-LaneGate $packageDirs $goFiles
}

Write-GateLine "`nFailures: $script:Failures"
Write-GateLine "`nSummary: report=$transcriptPath; failures=$script:Failures"
if ($script:Failures -ne 0) { exit 1 }
exit 0
