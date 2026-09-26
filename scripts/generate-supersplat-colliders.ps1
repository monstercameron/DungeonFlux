[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)] [ValidateSet('tavern','wooded')] [string] $SceneProfile,
    [Parameter(Mandatory = $true)] [string] $OutputDir,
    [string] $SourceDir = '',
    [string] $ToolPath = '',
    [string] $NodePath = '',
    [ValidateRange(0, 5)] [int] $SourceLOD = 1,
    [double] $VoxelSize = 0.2,
    [string] $SeedPos = '',
    [string] $FilterBox = '',
    [switch] $NoFilter,
    [switch] $Overwrite
)

Set-StrictMode -Version 2.0
$ErrorActionPreference = 'Stop'

function Get-Profile([string] $Name) {
    if ($Name -eq 'tavern') { return [pscustomobject]@{ id = 'cb2fddd6'; defaultSeed = '0,0,0'; filter = '-8,-10.5,-7.524,20.384,-4.5,11.716' } }
    return [pscustomobject]@{ id = '64bb46d5'; defaultSeed = '0,0,0'; filter = '30.38,-26,-40.048,58.764,-19,-20.808' }
}

function Get-Tool([string] $Value) {
    if ($Value) { return (Resolve-Path $Value).Path }
    $candidate = Join-Path $PSScriptRoot '..\artifacts\tools\splat\node_modules\.bin\splat-transform.ps1'
    if (-not (Test-Path -LiteralPath $candidate)) { throw "splat-transform not found; pass -ToolPath or install under artifacts/tools/splat" }
    return (Resolve-Path $candidate).Path
}

function Get-Source([string] $Value, [string] $ID) {
    if ($Value) {
        $resolved = (Resolve-Path $Value).Path
        if ((Get-Item -LiteralPath $resolved).PSIsContainer) { $resolved = Join-Path $resolved 'lod-meta.json' }
        return $resolved
    }
    $candidate = Join-Path $PSScriptRoot "..\artifacts\media\supersplat\$ID\lod-meta.json"
    if (-not (Test-Path -LiteralPath $candidate)) { throw "Downloaded lod-meta.json not found for scene $ID; pass -SourceDir" }
    return (Resolve-Path $candidate).Path
}

function Invoke-Tool([string] $Tool, [string] $Node, [string[]] $Arguments) {
    if ($Node) { & $Node $Tool @Arguments } else { & $Tool @Arguments }
    if ($LASTEXITCODE -ne 0) { throw "splat-transform failed with exit code $LASTEXITCODE" }
}

$profile = Get-Profile $SceneProfile
if ([double]::IsNaN($VoxelSize) -or [double]::IsInfinity($VoxelSize) -or $VoxelSize -lt 0.02 -or $VoxelSize -gt 1) { throw 'VoxelSize must be finite and between 0.02 and 1 world units' }
$source = Get-Source $SourceDir $profile.id
$tool = Get-Tool $ToolPath
$root = [IO.Path]::GetFullPath($OutputDir)
[IO.Directory]::CreateDirectory($root) | Out-Null
$output = Join-Path $root "$($profile.id).voxel.json"
if ((Test-Path -LiteralPath $output) -and -not $Overwrite) { throw "Output exists; pass -Overwrite to regenerate: $output" }
$effectiveSeed = if ($SeedPos) { $SeedPos } else { $profile.defaultSeed }
$args = @($source, '--select-lod', [string]$SourceLOD, '--voxel-size', $VoxelSize.ToString('R', [Globalization.CultureInfo]::InvariantCulture), '--voxel-opacity', '0.1', '--seed-pos', $effectiveSeed)
if (-not $NoFilter) {
    if (-not $FilterBox) { $FilterBox = $profile.filter }
    if ($FilterBox) { $args += @('--filter-box', $FilterBox) }
}
$args += @('--gpu', '0')
if ($Overwrite) { $args += '--overwrite' }
$args += @($output)
$node = $NodePath
if (-not $node) {
    $candidateNode = Join-Path $PSScriptRoot '..\artifacts\tools\node-v26.10.0-win-x64\node.exe'
    if (Test-Path -LiteralPath $candidateNode) { $node = (Resolve-Path $candidateNode).Path }
}
if ($node) { $node = (Resolve-Path $node).Path }
$cliTool = $tool
if ($node) { $cliTool = (Resolve-Path (Join-Path (Split-Path $tool -Parent) '..\@playcanvas\splat-transform\bin\cli.mjs')).Path }
$toolVersion = (Get-Content -Raw (Join-Path (Split-Path $tool -Parent) '..\@playcanvas\splat-transform\package.json') | ConvertFrom-Json).version
if ($toolVersion -ne '3.6.6') { throw "Expected pinned splat-transform 3.6.6, found $toolVersion" }
Invoke-Tool $cliTool $node $args
$meta = Get-Content -Raw -LiteralPath $output | ConvertFrom-Json
$bin = $output -replace '(?i)\.voxel\.json$', '.voxel.bin'
if (-not (Test-Path -LiteralPath $bin)) { throw "splat-transform did not produce $bin" }
$inventory = Join-Path (Split-Path $source -Parent) 'inventory.json'
$inventoryHash = if (Test-Path -LiteralPath $inventory) { (Get-FileHash -Algorithm SHA256 -LiteralPath $inventory).Hash.ToLowerInvariant() } else { $null }
$record = [ordered]@{
    generated_utc = [DateTime]::UtcNow.ToString('o')
    scene_profile = $SceneProfile
    scene_id = $profile.id
    source_lod = $SourceLOD
    source_path = $source
    source_sha256 = (Get-FileHash -Algorithm SHA256 -LiteralPath $source).Hash.ToLowerInvariant()
    source_inventory_sha256 = $inventoryHash
    tool_path = $tool
    tool_version = $toolVersion
    voxel_size = $VoxelSize
    voxel_opacity = 0.1
    seed_pos = $effectiveSeed
    source_transform = 'splat-transform applies the standard PlayCanvas 180Z frame conversion; no extra rotation is applied'
    fill = 'raw occupancy (no fill or carve)'
    carve = $null
    filter_box = $FilterBox
    metadata = $meta
    voxel_json_sha256 = (Get-FileHash -Algorithm SHA256 -LiteralPath $output).Hash.ToLowerInvariant()
    voxel_bin_sha256 = (Get-FileHash -Algorithm SHA256 -LiteralPath $bin).Hash.ToLowerInvariant()
}
$json = $record | ConvertTo-Json -Depth 20
[IO.File]::WriteAllText((Join-Path $root "$($profile.id).provenance.json"), $json, [Text.UTF8Encoding]::new($false))
$allRecords = @()
$aggregate = Join-Path $root 'provenance.json'
if (Test-Path -LiteralPath $aggregate) {
    try {
        $old = Get-Content -Raw $aggregate | ConvertFrom-Json
        if ($old.records) { $allRecords = @($old.records) } elseif ($old.scene_id) { $allRecords = @($old) }
    } catch { $allRecords = @() }
}
$allRecords = @($allRecords | Where-Object { $_.scene_id -ne $profile.id }) + @([pscustomobject]$record)
$aggregateValue = [ordered]@{ generated_utc = [DateTime]::UtcNow.ToString('o'); records = $allRecords }
[IO.File]::WriteAllText($aggregate, ($aggregateValue | ConvertTo-Json -Depth 20), [Text.UTF8Encoding]::new($false))
Write-Output "Generated $output and $bin"
