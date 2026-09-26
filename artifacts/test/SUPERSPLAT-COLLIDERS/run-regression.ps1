$ErrorActionPreference='Stop'
$repo=(Resolve-Path (Join-Path $PSScriptRoot '..\..\..')).Path
$out=Join-Path $PSScriptRoot 'output'
if(Test-Path $out){Remove-Item -Recurse -Force $out}
& powershell -NoProfile -ExecutionPolicy Bypass -File (Join-Path $repo 'scripts/generate-supersplat-colliders.ps1') -SceneProfile 64bb46d5 -OutputDir $out -NodePath (Join-Path $PSScriptRoot 'stub-node.ps1') -SourceDir (Join-Path $repo 'artifacts/media/supersplat/64bb46d5') -SourceLOD 1 -VoxelSize 0.2 -Overwrite
if($LASTEXITCODE -ne 0){throw 'stub generation failed'}
$p=Get-Content -Raw (Join-Path $out '64bb46d5.provenance.json')|ConvertFrom-Json
if($p.scene_id -ne '64bb46d5' -or $p.tool_version -ne '3.6.6' -or -not (Test-Path (Join-Path $out '64bb46d5.voxel.bin'))){throw 'provenance or output assertion failed'}
'passed offline stub CLI and provenance'
