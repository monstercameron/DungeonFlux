$ErrorActionPreference='Stop'
$repo=(Resolve-Path (Join-Path $PSScriptRoot '..\..\..')).Path
$generator=Join-Path $repo 'scripts/generate-supersplat-colliders.ps1'
$out=Join-Path $PSScriptRoot ('output-'+[Guid]::NewGuid().ToString('N'))
foreach($id in @('cb2fddd6','64bb46d5')) {
    & $generator -SceneProfile $id -OutputDir $out -NodePath (Join-Path $PSScriptRoot 'stub-node.ps1') -VoxelSize 0.2
    $p=Get-Content -Raw (Join-Path $out "$id.provenance.json")|ConvertFrom-Json
    if($p.scene_id -ne $id -or $p.tool_version -ne '3.6.6'){throw 'scene or pinned version assertion failed'}
    $expectedLod=if($id -eq 'cb2fddd6'){'2'}else{'1'}
    $argv=Get-Content -Raw (Join-Path $out "$id.args.json")|ConvertFrom-Json
    if($argv[1] -ne '--select-lod' -or $argv[2] -ne $expectedLod -or $argv[4] -ne '0.2'){throw 'default LOD or CLI units/order failed'}
    if($p.source_lod -ne [int]$expectedLod){throw 'effective LOD provenance failed'}
    $actual=(Get-FileHash -Algorithm SHA256 (Join-Path $out "$id.voxel.bin")).Hash.ToLowerInvariant()
    if($actual -ne $p.voxel_bin_sha256){throw 'provenance hash mismatch'}
    $rejected=$false
    try { & $generator -SceneProfile $id -OutputDir $out -NodePath (Join-Path $PSScriptRoot 'stub-node.ps1') } catch { $rejected=$_.Exception.Message -match 'Output exists' }
    if(-not $rejected){throw 'existing output was not protected'}
}
$overrideOut=Join-Path $PSScriptRoot ('override-'+[Guid]::NewGuid().ToString('N'))
& $generator -SceneProfile cb2fddd6 -OutputDir $overrideOut -NodePath (Join-Path $PSScriptRoot 'stub-node.ps1') -SourceLOD 1 -VoxelSize 0.2
$overrideArgs=Get-Content -Raw (Join-Path $overrideOut 'cb2fddd6.args.json')|ConvertFrom-Json
if($overrideArgs[2] -ne '1'){throw 'explicit SourceLOD override failed'}
$records=(Get-Content -Raw (Join-Path $out 'provenance.json')|ConvertFrom-Json).records
if($records.Count -ne 2){throw 'per-scene provenance was overwritten'}
'passed per-scene defaults, explicit SourceLOD override, CLI units/order, hashes, output protection, and aggregate provenance'
