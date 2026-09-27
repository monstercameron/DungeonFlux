[CmdletBinding()]
param(
    [switch]$Live,
    [int]$Count = 20,
    [string]$FakeUri = '',
    [int]$TimeoutSeconds = 30,
    [switch]$Json
)

$ErrorActionPreference = 'Stop'
Set-Location (Join-Path $PSScriptRoot '..')

function Get-RequiredEnvironment {
    param([string[]]$Names)
    $missing = @($Names | Where-Object { [string]::IsNullOrWhiteSpace((Get-Item "Env:$($_)" -ErrorAction SilentlyContinue).Value) })
    if ($missing.Count -gt 0) {
        throw "Live probe requires: $($missing -join ', ')"
    }
}

function Get-Percentile {
    param([double[]]$Values, [double]$Percent)
    if ($Values.Count -eq 0) { return $null }
    $ordered = @($Values | Sort-Object)
    $rank = [Math]::Ceiling($ordered.Count * $Percent)
    return [double]$ordered[[Math]::Max(0, $rank - 1)]
}

function New-Client {
    $client = New-Object System.Net.Http.HttpClient
    $client.Timeout = [TimeSpan]::FromSeconds($TimeoutSeconds)
    return $client
}

function Invoke-FirstByte {
    param(
        [System.Net.Http.HttpClient]$Client,
        [System.Net.Http.HttpRequestMessage]$Request,
        [System.Diagnostics.Stopwatch]$Clock,
        [bool]$ReadLine
    )
    $response = $null
    try {
        $response = $Client.SendAsync($Request, [System.Net.Http.HttpCompletionOption]::ResponseHeadersRead).GetAwaiter().GetResult()
        if (-not $response.IsSuccessStatusCode) {
            throw "HTTP $([int]$response.StatusCode) $($response.ReasonPhrase)"
        }
        $stream = $response.Content.ReadAsStreamAsync().GetAwaiter().GetResult()
        $reader = New-Object System.IO.StreamReader($stream)
        if ($ReadLine) {
            while (-not $reader.EndOfStream) {
                $line = $reader.ReadLine()
                if (-not [string]::IsNullOrWhiteSpace($line)) {
                    return [double]$Clock.Elapsed.TotalMilliseconds
                }
            }
        } elseif ($stream.ReadByte() -ge 0) {
            return [double]$Clock.Elapsed.TotalMilliseconds
        }
        throw 'Response contained no first token or audio byte'
    } finally {
        if ($null -ne $response) { $response.Dispose() }
        $Request.Dispose()
    }
}

function New-LlmRequest {
    param([string]$Uri, [string]$Key)
    $request = New-Object System.Net.Http.HttpRequestMessage([System.Net.Http.HttpMethod]::Post, $Uri)
    $request.Headers.Authorization = New-Object System.Net.Http.Headers.AuthenticationHeaderValue('Bearer', $Key)
    $body = @{
        model = if ($env:DF_PROBE_LLM_MODEL) { $env:DF_PROBE_LLM_MODEL } else { 'gpt-6-luna' }
        input = 'Reply with exactly one short sentence: venue probe.'
        stream = $true
        reasoning = @{ effort = 'none' }
    } | ConvertTo-Json -Compress
    $request.Content = New-Object System.Net.Http.StringContent($body, [Text.Encoding]::UTF8, 'application/json')
    return $request
}

function New-TtsRequest {
    param([string]$Uri, [string]$Key)
    $request = New-Object System.Net.Http.HttpRequestMessage([System.Net.Http.HttpMethod]::Post, $Uri)
    $request.Headers.Add('xi-api-key', $Key)
    $body = @{ text = 'Venue probe.'; model_id = 'eleven_flash_v2_5'; output_format = 'mp3_44100_128' } | ConvertTo-Json -Compress
    $request.Content = New-Object System.Net.Http.StringContent($body, [Text.Encoding]::UTF8, 'application/json')
    return $request
}

function New-SilenceWav {
    $sampleRate = 16000
    $sampleCount = 1600
    $bytes = New-Object byte[](44 + ($sampleCount * 2))
    [Text.Encoding]::ASCII.GetBytes('RIFF').CopyTo($bytes, 0)
    [BitConverter]::GetBytes([int32]($bytes.Length - 8)).CopyTo($bytes, 4)
    [Text.Encoding]::ASCII.GetBytes('WAVEfmt ').CopyTo($bytes, 8)
    [BitConverter]::GetBytes([int32]16).CopyTo($bytes, 16)
    [BitConverter]::GetBytes([int16]1).CopyTo($bytes, 20)
    [BitConverter]::GetBytes([int16]1).CopyTo($bytes, 22)
    [BitConverter]::GetBytes([int32]$sampleRate).CopyTo($bytes, 24)
    [BitConverter]::GetBytes([int32]($sampleRate * 2)).CopyTo($bytes, 28)
    [BitConverter]::GetBytes([int16]2).CopyTo($bytes, 32)
    [BitConverter]::GetBytes([int16]16).CopyTo($bytes, 34)
    [Text.Encoding]::ASCII.GetBytes('data').CopyTo($bytes, 36)
    [BitConverter]::GetBytes([int32]($sampleCount * 2)).CopyTo($bytes, 40)
    return $bytes
}

function Invoke-Scribe {
    param([System.Net.Http.HttpClient]$Client, [string]$Uri, [string]$Key)
    $content = New-Object System.Net.Http.MultipartFormDataContent
    $audio = New-Object System.Net.Http.ByteArrayContent (New-SilenceWav)
    $audio.Headers.ContentType = New-Object System.Net.Http.Headers.MediaTypeHeaderValue('audio/wav')
    $content.Add($audio, 'file', 'venue-probe.wav')
    $content.Add((New-Object System.Net.Http.StringContent('scribe_v2')), 'model_id')
    $request = New-Object System.Net.Http.HttpRequestMessage([System.Net.Http.HttpMethod]::Post, $Uri)
    $request.Headers.Add('xi-api-key', $Key)
    $request.Content = $content
    try {
        $response = $Client.SendAsync($request).GetAwaiter().GetResult()
        if (-not $response.IsSuccessStatusCode) { throw "HTTP $([int]$response.StatusCode) $($response.ReasonPhrase)" }
    } finally {
        $request.Dispose()
    }
}

function Invoke-LiveProbe {
    param([int]$Index)
    $client = New-Client
    $clock = [Diagnostics.Stopwatch]::StartNew()
    try {
        $llmUri = if ($env:DF_PROBE_LLM_URL) { $env:DF_PROBE_LLM_URL } else { 'https://api.openai.com/v1/responses' }
        $ttsUri = if ($env:DF_PROBE_TTS_URL) { $env:DF_PROBE_TTS_URL } else { "https://api.elevenlabs.io/v1/text-to-speech/$($env:DF_PROBE_VOICE_ID)/stream" }
        $sttUri = if ($env:DF_PROBE_STT_URL) { $env:DF_PROBE_STT_URL } else { 'https://api.elevenlabs.io/v1/speech-to-text' }
        $llmFirst = Invoke-FirstByte $client (New-LlmRequest $llmUri $env:DF_OPENAI_API_KEY) $clock $true
        $ttsFirst = Invoke-FirstByte $client (New-TtsRequest $ttsUri $env:DF_ELEVENLABS_API_KEY) $clock $false
        Invoke-Scribe $client $sttUri $env:DF_ELEVENLABS_API_KEY
        [pscustomobject]@{ index = $Index; ok = $true; llm_first_ms = $llmFirst; voice_first_ms = $ttsFirst; release_to_voice_ms = [Math]::Round($ttsFirst - $llmFirst, 1); error = $null }
    } catch {
        [pscustomobject]@{ index = $Index; ok = $false; llm_first_ms = $null; voice_first_ms = $null; release_to_voice_ms = $null; error = $_.Exception.Message }
    } finally {
        $client.Dispose()
    }
}

function Invoke-FakeProbe {
    param([int]$Index, [string]$Uri)
    try {
        $payload = @{ index = $Index; mode = 'venue_probe' } | ConvertTo-Json -Compress
        $reply = Invoke-RestMethod -Method Post -Uri $Uri -Body $payload -ContentType 'application/json' -TimeoutSec $TimeoutSeconds
        [pscustomobject]@{ index = $Index; ok = [bool]$reply.ok; llm_first_ms = $reply.llm_first_ms; voice_first_ms = $reply.voice_first_ms; release_to_voice_ms = $reply.release_to_voice_ms; error = $reply.error }
    } catch {
        [pscustomobject]@{ index = $Index; ok = $false; llm_first_ms = $null; voice_first_ms = $null; release_to_voice_ms = $null; error = $_.Exception.Message }
    }
}

if ($Count -lt 1) { throw '-Count must be at least 1.' }
if ($Live) {
    Get-RequiredEnvironment @('DF_OPENAI_API_KEY', 'DF_ELEVENLABS_API_KEY', 'DF_PROBE_VOICE_ID')
    $results = @(1..$Count | ForEach-Object { Invoke-LiveProbe $_ })
} elseif (-not [string]::IsNullOrWhiteSpace($FakeUri)) {
    $results = @(1..$Count | ForEach-Object { Invoke-FakeProbe $_ $FakeUri })
} else {
    $plan = [pscustomobject]@{ mode = 'dry-run'; calls = $Count; calls_per_iteration = @('LLM first token', 'ElevenLabs TTS first audio', 'ElevenLabs Scribe transcription'); live_switch = 'Run with -Live and DF_OPENAI_API_KEY, DF_ELEVENLABS_API_KEY, and DF_PROBE_VOICE_ID.'; decision = 'Live mode is recommended only when every call succeeds and p90 release_to_voice_ms is at most 4000.' }
    if ($Json) { $plan | ConvertTo-Json -Depth 4 } else { $plan | Format-List | Out-String | Write-Output }
    exit 0
}

$successful = @($results | Where-Object ok)
$failures = @($results | Where-Object { -not $_.ok })
$latencies = @($successful | ForEach-Object { [double]$_.release_to_voice_ms })
$report = [pscustomobject]@{
    mode = if ($Live) { 'live' } else { 'fake' }
    calls = $results.Count
    successes = $successful.Count
    failures = $failures.Count
    p50_release_to_voice_ms = Get-Percentile $latencies 0.5
    p90_release_to_voice_ms = Get-Percentile $latencies 0.9
    safe_mode = ($failures.Count -gt 0 -or ($latencies.Count -gt 0 -and (Get-Percentile $latencies 0.9) -gt 4000))
    failure_details = @($failures | ForEach-Object { "#$($_.index): $($_.error)" })
}
if ($Json) { $report | ConvertTo-Json -Depth 5; exit 0 }
$report | Format-List | Write-Output
if ($report.safe_mode) { Write-Output 'Recommendation: Safe Mode (sequence_mode).' } else { Write-Output 'Recommendation: live mode.' }
