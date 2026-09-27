[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [ValidatePattern('^[A-Za-z0-9][A-Za-z0-9_-]*$')]
    [string]$Instance,

    [Parameter()]
    [string]$Run,

    [Parameter()]
    [ValidatePattern('^[A-Za-z]+$')]
    [string]$Level,

    [Parameter()]
    [string]$Trace,

    [Parameter()]
    [switch]$Follow
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$logRoot = Join-Path $repoRoot "artifacts\runtime\$Instance\logs"
if (-not (Test-Path -LiteralPath $logRoot -PathType Container)) {
    throw "Log directory does not exist: $logRoot"
}

$logFiles = @(Get-ChildItem -LiteralPath $logRoot -Filter '*.jsonl' -File | Sort-Object Name)
if ($logFiles.Count -eq 0) {
    throw "No JSONL log files found in: $logRoot"
}

function Get-RecordValue {
    param(
        [Parameter(Mandatory = $true)]
        [object]$Record,

        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    $property = $Record.PSObject.Properties[$Name]
    if ($null -eq $property) {
        return ''
    }
    return [string]$property.Value
}

function Test-LogRecord {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Line
    )

    if (-not $Run -and -not $Level -and -not $Trace) {
        return $true
    }

    try {
        $record = $Line | ConvertFrom-Json
    }
    catch {
        return $false
    }

    if ($Run -and ((Get-RecordValue -Record $record -Name 'run') -ne $Run)) {
        return $false
    }

    if ($Level -and ((Get-RecordValue -Record $record -Name 'level') -ine $Level)) {
        return $false
    }

    if ($Trace) {
        $recordTrace = Get-RecordValue -Record $record -Name 'trace'
        if (-not $recordTrace) {
            $recordTrace = Get-RecordValue -Record $record -Name 'trace_id'
        }
        if ($recordTrace -ne $Trace) {
            return $false
        }
    }

    return $true
}

function Write-FilteredLine {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Line
    )

    if (Test-LogRecord -Line $Line) {
        Write-Output $Line
    }
}

if ($Follow) {
    Get-Content -LiteralPath $logFiles.FullName -Wait | ForEach-Object {
        Write-FilteredLine -Line ([string]$_)
    }
    exit 0
}

foreach ($file in $logFiles) {
    Get-Content -LiteralPath $file.FullName | ForEach-Object {
        Write-FilteredLine -Line ([string]$_)
    }
}
