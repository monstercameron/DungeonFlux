[CmdletBinding(SupportsShouldProcess = $true, ConfirmImpact = 'High')]
param(
    [Parameter()]
    [ValidatePattern('^[A-Za-z0-9][A-Za-z0-9_-]*$')]
    [string]$Lane,

    [Parameter()]
    [switch]$Checkpoint
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$artifactsRoot = Join-Path $repoRoot 'artifacts'

function Remove-ArtifactPath {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Path
    )

    if (-not (Test-Path -LiteralPath $Path)) {
        return
    }

    if ($PSCmdlet.ShouldProcess($Path, 'Remove stale artifact output')) {
        Remove-Item -LiteralPath $Path -Recurse -Force
    }
}

function Remove-Children {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Path,

        [string[]]$Except = @()
    )

    if (-not (Test-Path -LiteralPath $Path)) {
        return
    }

    Get-ChildItem -LiteralPath $Path -Force | Where-Object {
        $Except -notcontains $_.Name
    } | ForEach-Object {
        Remove-ArtifactPath -Path $_.FullName
    }
}

if (-not $Lane -and -not $Checkpoint) {
    throw 'Specify -Lane <lane> or -Checkpoint.'
}

if ($Lane -and $Checkpoint) {
    throw 'Specify only one of -Lane and -Checkpoint.'
}

if ($Lane) {
    foreach ($area in @('build', 'test', 'coverage', 'tmp')) {
        Remove-ArtifactPath -Path (Join-Path $artifactsRoot "$area\$Lane")
    }
    exit 0
}

Remove-Children -Path (Join-Path $artifactsRoot 'build') -Except @('human')
Remove-Children -Path (Join-Path $artifactsRoot 'test')
Remove-Children -Path (Join-Path $artifactsRoot 'coverage')
Remove-Children -Path (Join-Path $artifactsRoot 'tmp')

$cachePath = Join-Path $artifactsRoot 'cache\go'
$driveName = (Split-Path -Qualifier $repoRoot).TrimEnd(':')
$drive = Get-PSDrive -Name $driveName
$minimumFreeBytes = 20GB
if ($drive.Free -lt $minimumFreeBytes) {
    Remove-ArtifactPath -Path $cachePath
    if ($PSCmdlet.ShouldProcess($cachePath, 'Recreate Go cache directory')) {
        New-Item -ItemType Directory -Path $cachePath -Force | Out-Null
    }
}
