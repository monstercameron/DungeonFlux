param(
    [string]$Path = (Join-Path (Split-Path -Parent $PSScriptRoot) '.env')
)

if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) {
    return
}

foreach ($line in [IO.File]::ReadLines($Path)) {
    $text = $line.Trim()
    if ([string]::IsNullOrWhiteSpace($text) -or $text.StartsWith('#')) {
        continue
    }
    if ($text.StartsWith('export ')) {
        $text = $text.Substring(7).TrimStart()
    }
    $separator = $text.IndexOf('=')
    if ($separator -le 0) {
        continue
    }
    $name = $text.Substring(0, $separator).Trim()
    if ($name -notmatch '^[A-Za-z_][A-Za-z0-9_]*$') {
        continue
    }
    $value = $text.Substring($separator + 1).Trim()
    if ($value.Length -ge 2 -and (($value.StartsWith('"') -and $value.EndsWith('"')) -or ($value.StartsWith("'") -and $value.EndsWith("'")))) {
        $value = $value.Substring(1, $value.Length - 2)
    }
    [Environment]::SetEnvironmentVariable($name, $value, 'Process')
}
