[CmdletBinding()]
param(
    [int]$Port = 8443,
    [string]$TaskName = 'DungeonFlux-HumanServer',
    [string]$DataDir = 'artifacts\runtime\human',
    [string]$ConfigPath = '',
    [switch]$SkipGate,
    [switch]$RegisterTask,
    [switch]$StartTask
)

$ErrorActionPreference = 'Stop'
Set-Location (Join-Path $PSScriptRoot '..')

$buildDir = Join-Path $PWD 'artifacts\build\human'
$logDir = Join-Path $PWD 'artifacts\logs\devserver'
New-Item -ItemType Directory -Force -Path $buildDir, $logDir | Out-Null
$binary = Join-Path $buildDir 'devserver.exe'

$env:GOCACHE = Join-Path $PWD 'artifacts\cache\go'
$env:GOTMPDIR = Join-Path $PWD 'artifacts\tmp\ORCH-D'
$env:TMP = $env:GOTMPDIR
$env:TEMP = $env:GOTMPDIR
New-Item -ItemType Directory -Force -Path $env:GOCACHE, $env:GOTMPDIR | Out-Null

go build -o $binary ./scripts/devserver
if ($LASTEXITCODE -ne 0) { throw "go build failed with exit code $LASTEXITCODE" }

$dataPath = if ([IO.Path]::IsPathRooted($DataDir)) { $DataDir } else { Join-Path $PWD $DataDir }
$argumentList = "-port $Port -repo `"$PWD`" -build-dir `"$buildDir`" -data-dir `"$dataPath`" -devlog `"$PWD\docs\devlog.html`" -status `"$PWD\artifacts\logs\devserver\status.json`""
if (-not [string]::IsNullOrWhiteSpace($ConfigPath)) {
    $configPath = if ([IO.Path]::IsPathRooted($ConfigPath)) { $ConfigPath } else { Join-Path $PWD $ConfigPath }
    $argumentList += " -config `"$configPath`""
}
if ($SkipGate) { $argumentList += ' -skip-gate' }
if ($RegisterTask) {
    $action = New-ScheduledTaskAction -Execute $binary -Argument $argumentList -WorkingDirectory $PWD
    $trigger = New-ScheduledTaskTrigger -AtLogOn -User ("{0}\{1}" -f $env:USERDOMAIN, $env:USERNAME)
    $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -MultipleInstances IgnoreNew -ExecutionTimeLimit ([TimeSpan]::Zero) -RestartCount 20 -RestartInterval (New-TimeSpan -Minutes 1)
    Register-ScheduledTask -TaskName $TaskName -Action $action -Trigger $trigger -Settings $settings -Description 'DungeonFlux human test server' -Force | Out-Null
}
if ($StartTask) {
    if (-not $RegisterTask -and -not (Get-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue)) {
        throw "Scheduled task '$TaskName' is not registered"
    }
    Start-ScheduledTask -TaskName $TaskName
}

Write-Output "Built $binary"
if ($RegisterTask) { Write-Output "Registered scheduled task $TaskName" }
if ($StartTask) { Write-Output "Started scheduled task $TaskName" }
