param([Parameter(ValueFromRemainingArguments=$true)][string[]]$Args)
$out=$Args[-1]
if (-not $out -or $out -notmatch '\.voxel\.json$') { exit 19 }
$meta=[ordered]@{version='1.1';asset=[ordered]@{generator='stub'};gridBounds=[ordered]@{min=@(0,0,0);max=@(1,1,1)};sceneBounds=[ordered]@{min=@(0,0,0);max=@(1,1,1)};voxelResolution=0.2;leafSize=4;treeDepth=1;numInteriorNodes=1;numMixedLeaves=1;nodeCount=2;leafDataCount=2}
[IO.File]::WriteAllText($out,($meta|ConvertTo-Json -Depth 10),[Text.UTF8Encoding]::new($false))
[IO.File]::WriteAllBytes(($out -replace '(?i)\.voxel\.json$','.voxel.bin'),[byte[]](0..15))
exit 0
