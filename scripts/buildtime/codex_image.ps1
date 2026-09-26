[CmdletBinding()]
param(
    [Parameter(Mandatory = $true, Position = 0)]
    [ValidateNotNullOrEmpty()]
    [string] $PromptFile,

    [Parameter(Mandatory = $true, Position = 1)]
    [ValidateNotNullOrEmpty()]
    [string] $OutputName
)

$ErrorActionPreference = 'Stop'

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
$promptPath = (Resolve-Path $PromptFile).Path
$name = [IO.Path]::GetFileNameWithoutExtension($OutputName)
if ([string]::IsNullOrWhiteSpace($name) -or $name -ne $OutputName.Replace('.png', '')) {
    throw "OutputName must be a simple PNG filename without a path: $OutputName"
}

$buildtimeRoot = Join-Path $repoRoot 'artifacts\runtime\buildtime'
$stagingRoot = Join-Path $repoRoot 'artifacts\tmp\L-OPS\codex-image'
$stagedPath = Join-Path $stagingRoot ($name + '.png')
$finalPath = Join-Path $buildtimeRoot ($name + '.png')
$reportPath = Join-Path $stagingRoot ($name + '.report.md')

New-Item -ItemType Directory -Force -Path $stagingRoot, $buildtimeRoot | Out-Null
if (Test-Path -LiteralPath $stagedPath) {
    Remove-Item -LiteralPath $stagedPath -Force
}

$prompt = Get-Content -LiteralPath $promptPath -Raw
$instruction = @"
$prompt

Use the built-in image generation tool. Save the final generated PNG at this exact path:
$stagedPath
Generate one opaque raster image only. Do not create alternate files, SVGs, or a report in place of the PNG.
"@

$instruction | & codex exec -m 'gpt-5.6-luna' --dangerously-bypass-approvals-and-sandbox --skip-git-repo-check -C $repoRoot -o $reportPath '-'
if ($LASTEXITCODE -ne 0) {
    throw "codex exec failed for $name with exit code $LASTEXITCODE"
}
if (-not (Test-Path -LiteralPath $stagedPath -PathType Leaf)) {
    throw "Codex completed without creating the expected PNG: $stagedPath"
}

$bytes = [IO.File]::ReadAllBytes($stagedPath)
$pngSignature = [byte[]](137, 80, 78, 71, 13, 10, 26, 10)
if ($bytes.Length -lt $pngSignature.Length) {
    throw "Generated file is too small to be a PNG: $stagedPath"
}
for ($i = 0; $i -lt $pngSignature.Length; $i++) {
    if ($bytes[$i] -ne $pngSignature[$i]) {
        throw "Generated file has an invalid PNG signature: $stagedPath"
    }
}

Copy-Item -LiteralPath $stagedPath -Destination $finalPath -Force
Write-Output "Generated $finalPath"
