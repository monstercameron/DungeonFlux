<#
.SYNOPSIS
Checks saved JSON lines or dfctl costs against plan section 0.14 (USD).
.DESCRIPTION
CostReport lines are cumulative snapshots: the last snapshot for each run wins.
Add run/run_id to saved reports for multiple runs; otherwise -Run names the run.
Call records with run, vendor and cost_estimate_usd are summed instead. Do not
mix snapshots and calls for the same run. Use run=buildtime for actual build
spend; absent that, -BuildTimeUSD supplies the plan's $20.50 estimate.
An empty proto report ({}) is rejected because it cannot prove spend is tracked.
Exit codes: 0 within caps, 1 over budget, 2 invalid input or dfctl failure.
.EXAMPLE
./scripts/costcheck.ps1 -InputPath artifacts/test/costs.jsonl
.EXAMPLE
Get-Content artifacts/test/costs.jsonl | ./scripts/costcheck.ps1
.EXAMPLE
./scripts/costcheck.ps1 -DfctlPath artifacts/build/dfctl.exe -Room ABCD -Run rehearsal-1
#>
[CmdletBinding()]
param(
    [string]$InputPath,
    [Parameter(ValueFromPipeline = $true)][string]$JsonLine,
    [string]$DfctlPath = '',
    [string]$Address = '127.0.0.1:9443',
    [string]$Room = '',
    [string]$Run = 'current',
    [decimal]$PerRunCapUSD = 0.62,
    [decimal]$BuildTimeCapUSD = 20.50,
    [decimal]$TotalCapUSD = 39.00,
    [decimal]$BuildTimeUSD = 20.50,
    [switch]$LivePcLoopsOff
)

begin {
    $ErrorActionPreference = 'Stop'
    if (-not $DfctlPath) { $DfctlPath = Join-Path $PSScriptRoot '../artifacts/build/dfctl.exe' }
    $lines = [Collections.Generic.List[string]]::new()

    function Get-Field($Record, [string[]]$Names) {
        foreach ($name in $Names) {
            $property = $Record.PSObject.Properties[$name]
            if ($null -ne $property) { return $property.Value }
        }
        return $null
    }

    function Convert-Money($Value, [string]$Label) {
        if ($null -eq $Value -or $Value -is [bool]) { throw "$Label is missing or invalid." }
        $text = [Convert]::ToString($Value, [Globalization.CultureInfo]::InvariantCulture)
        $amount = [decimal]0
        $style = [Globalization.NumberStyles]::Float
        if (-not [decimal]::TryParse($text, $style, [Globalization.CultureInfo]::InvariantCulture, [ref]$amount)) {
            throw "$Label must be a finite USD amount."
        }
        if ($amount -lt 0) { throw "$Label must be non-negative." }
        return $amount
    }

    function Add-Vendor($Vendors, [string]$Vendor, [decimal]$USD) {
        if ([string]::IsNullOrWhiteSpace($Vendor)) { throw 'A cost is missing its vendor.' }
        if (-not $Vendors.ContainsKey($Vendor)) { $Vendors[$Vendor] = [decimal]0 }
        $Vendors[$Vendor] += $USD
    }

    function Read-Snapshot($Record) {
        $vendors = @{}
        foreach ($entry in @(Get-Field $Record @('vendors'))) {
            if ($null -eq $entry) { continue }
            $usd = Convert-Money (Get-Field $entry @('usd')) 'vendor usd'
            Add-Vendor $vendors (Get-Field $entry @('vendor')) $usd
        }
        $sum = [decimal]0
        foreach ($value in $vendors.Values) { $sum += $value }
        $total = Get-Field $Record @('total_usd', 'totalUsd')
        if ($null -ne $total) {
            $total = Convert-Money $total 'total_usd'
            if ([Math]::Abs($total - $sum) -gt [decimal]0.000001) {
                throw 'total_usd does not equal the vendor costs.'
            }
        } elseif ($vendors.Count -eq 0) {
            throw 'Empty cost report: no evidence that spend is tracked.'
        }
        return $vendors
    }

    function Read-CostLines([string[]]$InputLines, [string]$DefaultRun) {
        $runs = @{}
        foreach ($line in $InputLines) {
            if ([string]::IsNullOrWhiteSpace($line)) { continue }
            $record = ConvertFrom-Json -InputObject $line -ErrorAction Stop
            if ($null -eq $record -or $record -is [array] -or $record -is [string]) {
                throw 'Each line must be a JSON cost object.'
            }
            $runId = Get-Field $record @('run', 'run_id', 'runId')
            if ([string]::IsNullOrWhiteSpace($runId)) { $runId = $DefaultRun }
            $cost = Get-Field $record @('cost_estimate_usd')
            $mode = if ($null -ne $cost) { 'calls' } else { 'snapshot' }
            if ($runs.ContainsKey($runId) -and $runs[$runId].mode -ne $mode) {
                throw "Run $runId mixes cumulative snapshots and individual calls."
            }
            if ($mode -eq 'snapshot') {
                $runs[$runId] = @{ mode = $mode; vendors = (Read-Snapshot $record) }
            } else {
                if (-not $runs.ContainsKey($runId)) {
                    $runs[$runId] = @{ mode = $mode; vendors = @{} }
                }
                Add-Vendor $runs[$runId].vendors (Get-Field $record @('vendor')) (Convert-Money $cost 'cost_estimate_usd')
            }
        }
        if ($runs.Count -eq 0) { throw 'No cost records were supplied.' }
        return $runs
    }

    function Measure-Costs($Runs, [decimal]$RunCap) {
        $vendors = @{}
        $runTotals = @{}
        $failures = [Collections.Generic.List[string]]::new()
        $liveTotal = [decimal]0
        foreach ($runId in ($Runs.Keys | Sort-Object)) {
            $sum = [decimal]0
            foreach ($vendor in $Runs[$runId].vendors.Keys) {
                $amount = $Runs[$runId].vendors[$vendor]
                $sum += $amount
                Add-Vendor $vendors $vendor $amount
            }
            $runTotals[$runId] = $sum
            if ($runId -ne 'buildtime') {
                $liveTotal += $sum
                if ($sum -gt $RunCap) { $failures.Add("Run $runId exceeds the per-run cap.") }
            }
        }
        $build = $BuildTimeUSD
        $buildSource = 'estimate'
        if ($runTotals.ContainsKey('buildtime')) { $build = $runTotals['buildtime']; $buildSource = 'records' }
        if ($build -gt $BuildTimeCapUSD) { $failures.Add('Build time exceeds its cap.') }
        $total = $liveTotal + $build
        if ($total -gt $TotalCapUSD) { $failures.Add('Total spend including build time exceeds its cap.') }
        return [ordered]@{
            ok = ($failures.Count -eq 0); per_vendor_usd = $vendors; per_run_usd = $runTotals
            live_usd = $liveTotal; buildtime_usd = $build; buildtime_source = $buildSource
            total_usd = $total; caps = @{ per_run_usd = $RunCap; buildtime_usd = $BuildTimeCapUSD; total_usd = $TotalCapUSD }
            failures = @($failures.ToArray())
        }
    }
}

process {
    if ($PSBoundParameters.ContainsKey('JsonLine')) { $lines.Add($JsonLine) }
}

end {
    try {
        foreach ($cap in @($PerRunCapUSD, $BuildTimeCapUSD, $TotalCapUSD, $BuildTimeUSD)) {
            if ($cap -lt 0) { throw 'Caps and estimated build spend must be non-negative.' }
        }
        if ([string]::IsNullOrWhiteSpace($Run)) { throw '-Run must name the default run.' }
        if ($LivePcLoopsOff) { $PerRunCapUSD = [decimal]0.27 }
        if ($InputPath) {
            if ($lines.Count -gt 0) { throw 'Use either -InputPath or piped JSON lines.' }
            foreach ($line in [IO.File]::ReadAllLines((Resolve-Path -LiteralPath $InputPath))) { $lines.Add($line) }
        } elseif ($lines.Count -eq 0) {
            if (-not (Test-Path -LiteralPath $DfctlPath -PathType Leaf)) {
                throw 'Build dfctl and pass -DfctlPath, or provide -InputPath/piped JSON lines.'
            }
            $arguments = @('--addr', $Address)
            if ($Room) { $arguments += @('--room', $Room) }
            $arguments += 'costs'
            $output = @(& $DfctlPath @arguments)
            if ($LASTEXITCODE -ne 0) { throw "dfctl costs failed with exit code $LASTEXITCODE." }
            foreach ($line in $output) { $lines.Add([string]$line) }
        }
        $runs = Read-CostLines $lines.ToArray() $Run
        $summary = Measure-Costs $runs $PerRunCapUSD
        $summary | ConvertTo-Json -Depth 8 -Compress
        if (-not $summary.ok) { exit 1 }
        exit 0
    } catch {
        [Console]::Error.WriteLine("costcheck: " + $_.Exception.Message)
        exit 2
    }
}
