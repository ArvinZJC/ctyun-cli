$ErrorActionPreference = "Stop"

if ([Environment]::OSVersion.Platform -ne [PlatformID]::Win32NT) {
    throw "ctyun install: install.ps1 requires Windows. On macOS, Linux, or WSL, use the Bash installation command in the README."
}

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
                Fail "could not download core-index.json. Check your connection and CTYUN_INSTALL_SOURCE, then rerun the installer. $($_.Exception.Message)"
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
        Fail "no ctyun release found for $GoOS/$GoArch on $ChannelLabel. If you pinned a channel, run Remove-Item Env:CTYUN_INSTALL_CHANNEL and retry."
    }

    $ArchivePath = Join-Path $TempRoot "ctyun.tar.gz"
    Write-Host "Downloading ctyun $($Release.version) ($($Release.channel), $GoOS/$GoArch)..."
    try {
        Get-Download (Join-DownloadUrl $ResolvedRoot $Artifact.url) $ArchivePath
    } catch {
        Fail "could not download $($Artifact.url). Check your connection to the release source and retry. $($_.Exception.Message)"
    }
    $ActualSHA = (Get-FileHash -Algorithm SHA256 -Path $ArchivePath).Hash.ToLowerInvariant()
    if ($ActualSHA -ne $Artifact.sha256) {
        Fail "checksum mismatch for $($Artifact.url). Retry from a trusted release source. If it fails again, report the version and source."
    }

    $ExtractDir = Join-Path $TempRoot "extract"
    New-Item -ItemType Directory -Path $ExtractDir | Out-Null
    if (-not (Get-Command tar.exe -CommandType Application -ErrorAction SilentlyContinue)) {
        Fail "tar.exe is required. Make it available on PATH, open a new PowerShell window, and retry."
    }
    tar.exe -xzf $ArchivePath -C $ExtractDir
    if ($LASTEXITCODE -ne 0) {
        Fail "could not extract archive (tar exit code $LASTEXITCODE). Check the tar diagnostic and free space in the temporary directory, then retry."
    }
    $BinaryPath = Join-Path $ExtractDir "ctyun.exe"
    if (-not (Test-Path -LiteralPath $BinaryPath -PathType Leaf)) {
        Fail "archive does not contain ctyun.exe"
    }

    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    $Target = Join-Path $InstallDir "ctyun.exe"
    if (Test-Path -LiteralPath $Target -PathType Container) {
        Fail "install target is a directory: $Target. Set CTYUN_INSTALL_DIR to another writable directory and retry."
    }
    Copy-Item -LiteralPath $BinaryPath -Destination $Target -Force
    Write-Host "Installed ctyun $($Release.version) to $Target"
    $InstalledFile = Get-Item -LiteralPath $Target
    $Command = Get-Command ctyun -ErrorAction SilentlyContinue
    if ($Command.Path -ne $InstalledFile.FullName) {
        $QuotedDir = [System.Management.Automation.Language.CodeGeneration]::EscapeSingleQuotedStringContent($InstalledFile.DirectoryName)
        Write-Host "For this PowerShell session, run the command below. For future sessions, add the installation directory to your user Path in Environment Variables."
        Write-Host "  `$env:PATH = '$QuotedDir' + [IO.Path]::PathSeparator + `$env:PATH"
    }
    Write-Host "Verify the installation and view the available commands:"
    Write-Host "  ctyun --version"
    Write-Host "  ctyun --help"
    Write-Host "To browse product plugins in all release channels, run:"
    Write-Host "  ctyun plugin list --available --channel all"
} finally {
    Remove-Item -LiteralPath $TempRoot -Recurse -Force -ErrorAction SilentlyContinue
}
