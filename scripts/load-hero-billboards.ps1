# Replays prepared hero clips through the authenticated loopback debug API.
# Generate clips with scripts/buildtime billboards; this loader makes no vendor calls.
[CmdletBinding()]
param(
    [Parameter(Mandatory)][string]$AssetDir,
    [Parameter(Mandatory)][string]$Addr,
    [string]$Room = "DF-FAKE",
    [string]$Manifest = "config/hero-billboards.json",
    [string]$SourceDir = "artifacts/media/UI-REPAIR/heroes",
    [string]$Dfctl = "artifacts/build/UI-REPAIR/dfctl.exe"
)
$ErrorActionPreference = "Stop"
if ($Addr -notmatch '^(localhost|127\.0\.0\.1):[0-9]+$') { throw "Use the loopback debug listener." }
if ([string]::IsNullOrWhiteSpace($env:DF_DEBUG_TOKEN)) { throw "Set DF_DEBUG_TOKEN for this review server." }
$catalogue = Get-Content -LiteralPath $Manifest -Raw | ConvertFrom-Json
$destination = [IO.Path]::GetFullPath($AssetDir)
if (-not (Test-Path -LiteralPath $destination -PathType Container)) { throw "AssetDir must already exist in the intended server data directory." }
# Validate every file before copying anything or sending events.
$clips = foreach ($hero in $catalogue.heroes) {
    if ($hero.seat -notin 1,2) { throw "Hero seat must be 1 or 2." }
    foreach ($clip in $hero.clips) {
        if ($clip.sha256 -cnotmatch '^[0-9a-f]{64}$') { throw "Invalid clip SHA256." }
        if ($clip.action -notin 'idle','attack','hit','fall') { throw "Unsupported clip action." }
        $source = Join-Path $SourceDir ($clip.sha256 + '.mp4')
        if ((Get-FileHash -LiteralPath $source -Algorithm SHA256).Hash.ToLowerInvariant() -cne $clip.sha256) { throw "Clip hash mismatch: $source" }
        [pscustomobject]@{ Seat=$hero.seat; Action=$clip.action; Hash=$clip.sha256; Source=$source; Duration=$clip.duration_ms }
    }
}
foreach ($clip in $clips) {
    $filename = $clip.Hash + '.mp4'
    Copy-Item -LiteralPath $clip.Source -Destination (Join-Path $destination $filename) -Force
    $payload = @{slot="billboard:$($clip.Seat):$($clip.Action)"; asset=@{id=$clip.Hash; sha256=$clip.Hash; kind='video'; mime='video/mp4'; url="/assets/$filename"; duration_ms=$clip.Duration}} | ConvertTo-Json -Compress
    & $Dfctl -addr $Addr -room $Room send asset_ready $payload
    if ($LASTEXITCODE -ne 0) { throw "The engine rejected $($clip.Action) for seat $($clip.Seat)." }
}
