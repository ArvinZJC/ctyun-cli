$ErrorActionPreference = "Stop"

$GitHubRoot = "https://github.com/ArvinZJC/ctyun-cli/releases/download/core"
$GiteeRoot = "https://gitee.com/ArvinZJC/ctyun-cli/releases/download/core"
$Channel = if ($env:CTYUN_INSTALL_CHANNEL) { $env:CTYUN_INSTALL_CHANNEL } else { "" }
$Source = if ($env:CTYUN_INSTALL_SOURCE) { $env:CTYUN_INSTALL_SOURCE } else { "auto" }
$InstallDir = if ($env:CTYUN_INSTALL_DIR) { $env:CTYUN_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA "Programs\ctyun-cli" }

function Fail($Message) {
    throw "ctyun install: $Message"
}

function Get-Download($Uri, $OutFile) {
    Invoke-WebRequest -Uri $Uri -OutFile $OutFile -UseBasicParsing
}

function Join-DownloadUrl($Root, $Path) {
    if ($Path -match "^https?://") {
        return $Path
    }
    return "$($Root.TrimEnd('/'))/$Path"
}

switch ($Source) {
    "auto" { $Roots = @($GitHubRoot, $GiteeRoot) }
    "github" { $Roots = @($GitHubRoot) }
    "gitee" { $Roots = @($GiteeRoot) }
    default { Fail "CTYUN_INSTALL_SOURCE must be auto, github, or gitee" }
}
if ($env:CTYUN_INSTALL_BASE_URL) {
    $Roots = @($env:CTYUN_INSTALL_BASE_URL)
}

$GoOS = "windows"
$ArchText = if ($env:PROCESSOR_ARCHITEW6432) { $env:PROCESSOR_ARCHITEW6432 } else { $env:PROCESSOR_ARCHITECTURE }
switch ($ArchText.ToLowerInvariant()) {
    "amd64" { $GoArch = "amd64" }
    "x86_64" { $GoArch = "amd64" }
    "arm64" { $GoArch = "arm64" }
    default { Fail "unsupported architecture: $ArchText" }
}

$TempRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("ctyun-install-" + [System.Guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $TempRoot | Out-Null
try {
    $IndexPath = Join-Path $TempRoot "core-index.json"
    $ResolvedRoot = $null
    foreach ($Candidate in $Roots) {
        try {
            Get-Download "$($Candidate.TrimEnd('/'))/core-index.json" $IndexPath
            $ResolvedRoot = $Candidate
            break
        } catch {
            if ($Candidate -eq $Roots[-1]) {
                throw
            }
        }
    }
    if (-not $ResolvedRoot) {
        Fail "could not download core-index.json"
    }

    $Index = Get-Content -Raw -Path $IndexPath | ConvertFrom-Json
    $Release = $null
    $Artifact = $null
    $ChannelOrder = if ($Channel) { @($Channel) } else { @("stable", "beta", "alpha") }
    foreach ($DesiredChannel in $ChannelOrder) {
        foreach ($CandidateRelease in $Index.releases) {
            if ($CandidateRelease.channel -ne $DesiredChannel) {
                continue
            }
            $CandidateArtifact = @($CandidateRelease.artifacts | Where-Object { $_.os -eq $GoOS -and $_.arch -eq $GoArch })[0]
            if ($CandidateArtifact) {
                $Release = $CandidateRelease
                $Artifact = $CandidateArtifact
                break
            }
        }
        if ($Artifact) {
            break
        }
    }
    if (-not $Artifact) {
        $ChannelLabel = if ($Channel) { "channel $Channel" } else { "any channel" }
        Fail "no ctyun release found for $GoOS/$GoArch on $ChannelLabel"
    }

    $ArchivePath = Join-Path $TempRoot "ctyun.tar.gz"
    Get-Download (Join-DownloadUrl $ResolvedRoot $Artifact.url) $ArchivePath
    $ActualSHA = (Get-FileHash -Algorithm SHA256 -Path $ArchivePath).Hash.ToLowerInvariant()
    if ($ActualSHA -ne $Artifact.sha256) {
        Fail "checksum mismatch for $($Artifact.url)"
    }

    $ExtractDir = Join-Path $TempRoot "extract"
    New-Item -ItemType Directory -Path $ExtractDir | Out-Null
    tar.exe -xzf $ArchivePath -C $ExtractDir
    $BinaryPath = Join-Path $ExtractDir "ctyun.exe"
    if (-not (Test-Path -LiteralPath $BinaryPath)) {
        Fail "archive does not contain ctyun.exe"
    }

    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    $Target = Join-Path $InstallDir "ctyun.exe"
    Copy-Item -LiteralPath $BinaryPath -Destination $Target -Force
    Write-Host "Installed ctyun $($Release.version) to $Target"
    $PathParts = ($env:PATH -split ";") | Where-Object { $_ }
    if ($PathParts -notcontains $InstallDir) {
        Write-Host "Add $InstallDir to PATH before running ctyun from a new shell."
    }
} finally {
    Remove-Item -LiteralPath $TempRoot -Recurse -Force -ErrorAction SilentlyContinue
}
