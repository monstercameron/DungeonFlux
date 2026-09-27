[CmdletBinding()]
param(
    [Parameter(Mandatory = $true, Position = 0)] [string] $InputUrl,
    [Parameter(Mandatory = $true)] [string] $OutputDir,
    [ValidateRange(1, 10)] [int] $Retries = 3,
    [int] $TimeoutSec = 60
)

Set-StrictMode -Version 2.0
$ErrorActionPreference = 'Stop'

function Get-HttpUri([string] $Value) {
    try { $uri = [Uri]$Value } catch { throw "Invalid URL: $Value" }
    if ($uri.Scheme -notin @('http', 'https') -or [string]::IsNullOrEmpty($uri.Host)) {
        throw "Only http and https URLs are supported: $Value"
    }
    return $uri
}

function Get-PageMetadata([string] $Html, [Uri] $PageUri) {
    $result = [ordered]@{ page_url = $PageUri.AbsoluteUri }
    $patterns = @{
        title = '<meta\s+property=["'']og:title["'']\s+content=["'']([^"'']*)'
        description = '<meta\s+property=["'']og:description["'']\s+content=["'']([^"'']*)'
        canonical = '<link\s+rel=["'']canonical["'']\s+href=["'']([^"'']*)'
        license = '<link\s+rel=["'']license["'']\s+href=["'']([^"'']*)'
    }
    foreach ($key in $patterns.Keys) {
        $match = [regex]::Match($Html, $patterns[$key], 'IgnoreCase')
        if ($match.Success) { $result[$key] = [System.Net.WebUtility]::HtmlDecode($match.Groups[1].Value) }
    }
    return [pscustomobject]$result
}

function Find-ContentUrl([string] $Html) {
    $patterns = @(
        '"contentUrl"\s*:\s*"([^"]+\.json(?:\?[^" ]*)?)"',
        "'contentUrl'\s*:\s*'([^']+\.json(?:\?[^' ]*)?)'"
    )
    foreach ($pattern in $patterns) {
        $match = [regex]::Match($Html, $pattern, 'IgnoreCase')
        if ($match.Success) { return [System.Net.WebUtility]::HtmlDecode($match.Groups[1].Value) }
    }
    return $null
}

function Invoke-Text([Uri] $Uri, [int] $Attempts, [int] $Timeout) {
    for ($try = 1; $try -le $Attempts; $try++) {
        try { return (Invoke-WebRequest -UseBasicParsing -Uri $Uri.AbsoluteUri -TimeoutSec $Timeout).Content }
        catch {
            if ($try -eq $Attempts) { throw "GET failed after $Attempts attempts: $($Uri.AbsoluteUri): $($_.Exception.Message)" }
            Start-Sleep -Seconds ([Math]::Min(4, $try))
        }
    }
}

function Get-Bootstrap([Uri] $InputUri, [int] $Attempts, [int] $Timeout) {
    if ($InputUri.AbsolutePath -match '\.json$') {
        return [pscustomobject]@{ manifest = $InputUri; page = $null; html = $null; settings = $null }
    }
    $originalHtml = Invoke-Text $InputUri $Attempts $Timeout
    $content = Find-ContentUrl $originalHtml
    $viewerHtml = $originalHtml
    $viewerUri = $InputUri
    if (-not $content -and $InputUri.Host -eq 'superspl.at' -and $InputUri.AbsolutePath -match '/scene/([^/]+)') {
        $sceneID = $Matches[1]
        $viewerUri = [Uri]("https://superspl.at/s?id=" + $sceneID)
        $viewerHtml = Invoke-Text $viewerUri $Attempts $Timeout
        $content = Find-ContentUrl $viewerHtml
    }
    if (-not $content) { throw "Could not find contentUrl in the SuperSplat page" }
    $settings = Find-ViewerSettings $viewerHtml
    return [pscustomobject]@{ manifest = [Uri]::new($viewerUri, $content); page = $InputUri; html = $originalHtml; settings = $settings }
}

function Find-ViewerSettings([string] $Html) {
    $match = [regex]::Match($Html, '<script[^>]+id=["'']sse-bootstrap["''][^>]*>(.*?)</script>', 'IgnoreCase,Singleline')
    if (-not $match.Success) { return $null }
    try { return (($match.Groups[1].Value | ConvertFrom-Json).settings) } catch { return $null }
}

function Get-RelativeResourcePath([Uri] $RootUri, [Uri] $ResourceUri) {
    if ($RootUri.Host -ne $ResourceUri.Host -or $RootUri.Scheme -ne $ResourceUri.Scheme -or $RootUri.Port -ne $ResourceUri.Port) {
        throw "External resource host is not allowed: $($ResourceUri.AbsoluteUri)"
    }
    $rootPath = $RootUri.AbsolutePath
    $slash = $rootPath.LastIndexOf('/')
    $rootDirectory = $rootPath.Substring(0, $slash + 1)
    $relative = [Uri]::new($RootUri, $rootDirectory).MakeRelativeUri($ResourceUri).ToString()
    $relative = [Uri]::UnescapeDataString($relative) -replace '/', '\'
    if ([string]::IsNullOrWhiteSpace($relative) -or $relative -match '(^|\\)\.\.(\\|$)' -or [IO.Path]::IsPathRooted($relative) -or $relative -match '[:\x00-\x1F]') {
        throw "Resource path escapes the manifest directory: $($ResourceUri.AbsoluteUri)"
    }
    return $relative
}

function Find-References($Value, [string] $BaseUrl, [System.Collections.Generic.List[object]] $Found, [string] $Key = '', [bool] $Allowed = $false) {
    if ($null -eq $Value) { return }
    if ($Value -is [string]) {
        if ($Allowed -and $Value -match '(?i)(https?://|\.(json|sog|ply|spz|webp|png|jpe?g|bin|dat)(\?|$))') {
            $Found.Add([pscustomobject]@{ Value = $Value; BaseUrl = $BaseUrl; Key = $Key })
        }
        return
    }
    if ($Value -is [System.Collections.IEnumerable] -and $Value -isnot [pscustomobject]) {
        foreach ($item in $Value) { Find-References $item $BaseUrl $Found $Key $Allowed }
        return
    }
    foreach ($property in $Value.PSObject.Properties) {
        $isResourceField = $property.Name -match '^(?i:filename|filenames|file|files|texture|textures|resource|resources)$'
        Find-References $property.Value $BaseUrl $Found $property.Name $isResourceField
    }
}

function Resolve-Resource([string] $Value, [Uri] $BaseUri) {
    try { $uri = [Uri]::new($BaseUri, $Value) } catch { throw "Invalid resource reference '$Value'" }
    return Get-HttpUri $uri.AbsoluteUri
}

function Assert-SafeDestination([string] $Root, [string] $Path) {
    $full = [IO.Path]::GetFullPath($Path)
    $prefix = $Root.TrimEnd('\') + '\'
    if (-not $full.StartsWith($prefix, [StringComparison]::OrdinalIgnoreCase)) { throw "Destination escapes output directory: $Path" }
    $cursor = Split-Path -Parent $full
    while ($cursor -and $cursor.StartsWith($prefix, [StringComparison]::OrdinalIgnoreCase)) {
        if (Test-Path -LiteralPath $cursor) {
            $attrs = (Get-Item -LiteralPath $cursor -Force).Attributes
            if (($attrs -band [IO.FileAttributes]::ReparsePoint) -ne 0) { throw "Reparse point in destination path: $cursor" }
        }
        $cursor = Split-Path -Parent $cursor
    }
    return $full
}

function Save-Resource([Uri] $Uri, [string] $Path, [int] $Attempts, [int] $Timeout, [hashtable] $Inventory) {
    $dir = Split-Path -Parent $Path
    if ($dir) { [IO.Directory]::CreateDirectory($dir) | Out-Null }
    $key = $Path.ToLowerInvariant()
    if ((Test-Path -LiteralPath $Path) -and $Inventory.ContainsKey($key)) {
        $hash = (Get-FileHash -Algorithm SHA256 -LiteralPath $Path).Hash.ToLowerInvariant()
        if ($hash -eq $Inventory[$key].sha256 -and $Inventory[$key].url -eq $Uri.AbsoluteUri) { return $Inventory[$key] }
    }
    $part = "$Path.part"
    for ($try = 1; $try -le $Attempts; $try++) {
        try {
            Invoke-WebRequest -UseBasicParsing -Uri $Uri.AbsoluteUri -TimeoutSec $Timeout -OutFile $part
            Move-Item -LiteralPath $part -Destination $Path -Force
            $item = [ordered]@{ path = $Path; url = $Uri.AbsoluteUri; bytes = (Get-Item -LiteralPath $Path).Length; sha256 = (Get-FileHash -Algorithm SHA256 -LiteralPath $Path).Hash.ToLowerInvariant() }
            $Inventory[$key] = [pscustomobject]$item
            return $Inventory[$key]
        } catch {
            if (Test-Path -LiteralPath $part) { Remove-Item -LiteralPath $part -Force -ErrorAction SilentlyContinue }
            if ($try -eq $Attempts) { throw "Download failed after $Attempts attempts: $($Uri.AbsoluteUri): $($_.Exception.Message)" }
            Start-Sleep -Seconds ([Math]::Min(4, $try))
        }
    }
}

function Write-Inventory([hashtable] $Inventory, [string] $Path, [string[]] $Reachable, [Uri] $Manifest) {
    $files = @($Reachable | ForEach-Object { $Inventory[$_.ToLowerInvariant()] } | Where-Object { $null -ne $_ } | Sort-Object path)
    Write-JsonFile ([ordered]@{ generated_utc = [DateTime]::UtcNow.ToString('o'); manifest_url = $Manifest.AbsoluteUri; files = $files }) $Path
}

function Write-JsonFile($Value, [string] $Path) {
    $json = $Value | ConvertTo-Json -Depth 100
    [IO.File]::WriteAllText($Path, $json, [Text.UTF8Encoding]::new($false))
}

$bootstrap = Get-Bootstrap (Get-HttpUri $InputUrl) $Retries $TimeoutSec
$manifestUri = Get-HttpUri $bootstrap.manifest.AbsoluteUri
$root = [IO.Path]::GetFullPath($OutputDir)
[IO.Directory]::CreateDirectory($root) | Out-Null
$inventoryPath = Join-Path $root 'inventory.json'
$inventory = @{}
if (Test-Path -LiteralPath $inventoryPath) {
    try { foreach ($entry in (Get-Content -Raw $inventoryPath | ConvertFrom-Json).files) { $inventory[$entry.path.ToLowerInvariant()] = $entry } } catch { $inventory = @{} }
}
$queue = [System.Collections.Generic.Queue[object]]::new()
$seen = [System.Collections.Generic.HashSet[string]]::new([StringComparer]::OrdinalIgnoreCase)
$manifestName = [IO.Path]::GetFileName($manifestUri.AbsolutePath)
if ([string]::IsNullOrWhiteSpace($manifestName)) { $manifestName = 'manifest.json' }
$queue.Enqueue([pscustomobject]@{ uri = $manifestUri; path = Assert-SafeDestination $root (Join-Path $root $manifestName); json = $true })
$downloaded = [System.Collections.Generic.List[object]]::new()
$reachable = [System.Collections.Generic.HashSet[string]]::new([StringComparer]::OrdinalIgnoreCase)
while ($queue.Count -gt 0) {
    $item = $queue.Dequeue(); $key = $item.uri.AbsoluteUri
    if (-not $seen.Add($key)) { continue }
    $saved = Save-Resource $item.uri $item.path $Retries $TimeoutSec $inventory
    $downloaded.Add($saved)
    $reachable.Add($item.path) | Out-Null
    Write-Inventory $inventory $inventoryPath @($reachable) $manifestUri
    if (-not $item.json) { continue }
    try { $doc = Get-Content -Raw -LiteralPath $item.path | ConvertFrom-Json } catch { throw "Downloaded JSON is invalid: $($item.uri.AbsoluteUri): $($_.Exception.Message)" }
    $refs = [System.Collections.Generic.List[object]]::new(); Find-References $doc $item.uri.AbsoluteUri $refs
    foreach ($ref in $refs) {
        $refUri = Resolve-Resource $ref.Value $item.uri
        $relative = Get-RelativeResourcePath $manifestUri $refUri
        $local = Assert-SafeDestination $root (Join-Path $root $relative)
        $isJson = $refUri.AbsolutePath -match '(?i)\.json$'
        $queue.Enqueue([pscustomobject]@{ uri = $refUri; path = $local; json = $isJson })
    }
}
$files = @($reachable | ForEach-Object { $inventory[$_] } | Sort-Object path)
Write-Inventory $inventory $inventoryPath @($reachable) $manifestUri
$sceneUrl = $null
$pageMetadata = $null
if ($bootstrap.page) {
    $sceneUrl = $bootstrap.page.AbsoluteUri
    if ($bootstrap.html) { $pageMetadata = Get-PageMetadata $bootstrap.html $bootstrap.page }
}
$provenance = [ordered]@{ scene = $sceneUrl; manifest = $manifestUri.AbsoluteUri; page = $pageMetadata; downloaded_utc = [DateTime]::UtcNow.ToString('o'); file_count = $files.Count }
Write-JsonFile $provenance (Join-Path $root 'provenance.json')
if ($bootstrap.settings) { Write-JsonFile $bootstrap.settings (Join-Path $root 'viewer-settings.json') }
Write-Output ("Downloaded {0} files to {1}" -f $files.Count, $root)
