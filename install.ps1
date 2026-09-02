<#
.SYNOPSIS
    Installs a released beaver binary: no Go toolchain, no administrator rights,
    one command.

.DESCRIPTION
    Resolves a release, downloads the archive built for this machine, verifies
    its SHA-256 against the release's published checksums file, and unpacks
    beaver.exe into the user's local programs directory. Nothing is installed
    until the checksum matches.

    The install directory is added to the user PATH. The machine-wide PATH is
    never touched, which is what keeps this an ordinary-user install.

    Running it again upgrades in place: the binary is replaced and the PATH
    entry is not duplicated.

.PARAMETER Version
    Release to install, with or without the leading v. Defaults to
    $env:BEAVER_VERSION, and to the latest release when that is unset.

.PARAMETER InstallDir
    Where to install. Defaults to $env:BEAVER_INSTALL_DIR, and to
    %LOCALAPPDATA%\Programs\beaver when that is unset.

.EXAMPLE
    irm https://beaverbacklog.com/install.ps1 | iex

.EXAMPLE
    .\install.ps1 -Version 1.0.0

.NOTES
    Nothing here reads stdin or the script's own path, so piping the script into
    PowerShell behaves exactly like running a saved copy. Piped that way there
    is no command line to carry parameters, which is what the two environment
    variables are for:

        $env:BEAVER_VERSION = '1.0.0'; irm https://... | iex
#>
[CmdletBinding()]
param(
    [string]$Version = $env:BEAVER_VERSION,
    [string]$InstallDir = $env:BEAVER_INSTALL_DIR
)

# Piped into iex, the script runs inside the caller's own session. The block
# gives it a scope of its own, so the preferences below and strict mode do not
# outlive the install in an interactive shell.
& {
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$repo = 'builtbystef/beaver-backlog'
$tmp = $null

try {
    # The OS architecture rather than the process one: PowerShell can be running
    # emulated as x64 on an ARM64 machine, which should still get the arm64 build.
    $arch = switch ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture) {
        'X64' { 'amd64' }
        'Arm64' { 'arm64' }
        default { throw "unsupported architecture: $_. beaver publishes Windows builds for x64 and ARM64." }
    }

    # Windows PowerShell 5.1 can still default to protocols github.com refuses.
    [Net.ServicePointManager]::SecurityProtocol =
        [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12

    # Invoke-WebRequest's progress bar costs more time than the download itself
    # and leaves the caller's screen redrawn.
    $ProgressPreference = 'SilentlyContinue'

    # The archive names carry the version, so the tag has to be known before
    # anything can be fetched; the releases API is what names the latest one.
    if ([string]::IsNullOrWhiteSpace($Version)) {
        try {
            $latest = Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases/latest" -UseBasicParsing
        } catch {
            throw "could not reach the GitHub releases API to find the latest release: $($_.Exception.Message)"
        }
        $Version = $latest.tag_name
        if ([string]::IsNullOrWhiteSpace($Version)) {
            throw "the GitHub releases API named no latest release of $repo"
        }
    }

    # Tags are v-prefixed; the names inside a release are not.
    $tag = if ($Version.StartsWith('v')) { $Version } else { "v$Version" }
    $Version = $tag.Substring(1)

    $archive = "beaver_${Version}_windows_${arch}.zip"
    $checksums = "beaver_${Version}_checksums.txt"
    $base = "https://github.com/$repo/releases/download/$tag"

    $tmp = Join-Path ([System.IO.Path]::GetTempPath()) "beaver-install-$([System.IO.Path]::GetRandomFileName())"
    New-Item -ItemType Directory -Path $tmp | Out-Null

    Write-Output "Downloading beaver $Version for windows/$arch"
    $archivePath = Join-Path $tmp $archive
    try {
        Invoke-WebRequest -Uri "$base/$archive" -OutFile $archivePath -UseBasicParsing
    } catch {
        throw "could not download $base/${archive}: check that release $tag exists and publishes a windows/$arch build"
    }
    $checksumsPath = Join-Path $tmp $checksums
    try {
        Invoke-WebRequest -Uri "$base/$checksums" -OutFile $checksumsPath -UseBasicParsing
    } catch {
        throw "could not download $base/$checksums, so the download cannot be verified"
    }

    # One '<hash>  <filename>' line per archive, the same file install.sh reads.
    $want = ''
    foreach ($line in Get-Content -LiteralPath $checksumsPath) {
        $fields = $line.Trim() -split '\s+', 2
        if ($fields.Count -eq 2 -and $fields[1] -eq $archive) {
            $want = $fields[0]
            break
        }
    }
    if ($want -eq '') {
        throw "$checksums has no entry for $archive"
    }
    # Get-FileHash reports upper case and the file is written in lower case; -ne
    # on strings ignores case, so the two are directly comparable.
    $got = (Get-FileHash -LiteralPath $archivePath -Algorithm SHA256).Hash
    if ($got -ne $want) {
        throw "checksum mismatch for ${archive}: got $($got.ToLowerInvariant()), expected $($want.ToLowerInvariant()). Nothing was installed."
    }

    $unpacked = Join-Path $tmp 'unpacked'
    Expand-Archive -LiteralPath $archivePath -DestinationPath $unpacked -Force
    $binary = Join-Path $unpacked 'beaver.exe'
    if (-not (Test-Path -LiteralPath $binary)) {
        throw "$archive does not contain a beaver.exe"
    }

    if ([string]::IsNullOrWhiteSpace($InstallDir)) {
        $programs = [Environment]::GetFolderPath('LocalApplicationData')
        if ([string]::IsNullOrWhiteSpace($programs)) {
            throw 'LOCALAPPDATA is not set: name a destination with BEAVER_INSTALL_DIR'
        }
        $InstallDir = Join-Path (Join-Path $programs 'Programs') 'beaver'
    }
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    Copy-Item -LiteralPath $binary -Destination (Join-Path $InstallDir 'beaver.exe') -Force

    # The user PATH only. Read raw and written back with the kind it already had,
    # so entries like %USERPROFILE%\... survive as the expandable strings they are.
    $environment = 'HKCU:\Environment'
    $key = Get-Item -LiteralPath $environment
    $userPath = [string]$key.GetValue('Path', '', [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)
    $kind = if ($key.GetValueNames() -contains 'Path') {
        $key.GetValueKind('Path')
    } else {
        [Microsoft.Win32.RegistryValueKind]::ExpandString
    }

    $target = $InstallDir.TrimEnd('\')
    $entries = @($userPath -split ';' | Where-Object { $_.Trim() -ne '' })
    $onUserPath = $false
    foreach ($entry in $entries) {
        if ([Environment]::ExpandEnvironmentVariables($entry).Trim().TrimEnd('\') -eq $target) {
            $onUserPath = $true
            break
        }
    }
    if (-not $onUserPath) {
        Set-ItemProperty -LiteralPath $environment -Name 'Path' -Value (($entries + $InstallDir) -join ';') -Type $kind

        # Explorer caches the environment it hands to everything it starts, so
        # without this broadcast a "new shell" would not see the new PATH until
        # the next sign-in. Losing the broadcast is not worth failing over.
        try {
            if (-not ('Beaver.NativeMethods' -as [type])) {
                Add-Type -Namespace 'Beaver' -Name 'NativeMethods' -MemberDefinition @'
[DllImport("user32.dll", SetLastError = true, CharSet = CharSet.Auto)]
public static extern IntPtr SendMessageTimeout(IntPtr hWnd, uint msg, UIntPtr wParam, string lParam, uint flags, uint timeout, out UIntPtr result);
'@
            }
            $ignored = [UIntPtr]::Zero
            # HWND_BROADCAST, WM_SETTINGCHANGE, SMTO_ABORTIFHUNG, five seconds.
            [void][Beaver.NativeMethods]::SendMessageTimeout(
                [IntPtr]0xffff, 0x1A, [UIntPtr]::Zero, 'Environment', 0x2, 5000, [ref]$ignored)
        } catch {
            Write-Verbose "could not broadcast the PATH change: $($_.Exception.Message)"
        }
    }

    Write-Output "Installed beaver $Version to $(Join-Path $InstallDir 'beaver.exe')"

    $inSession = @($env:Path -split ';' | Where-Object { $_.Trim() -ne '' } |
            ForEach-Object { $_.Trim().TrimEnd('\') }) -contains $target
    if (-not $inSession) {
        Write-Output ''
        Write-Output "$InstallDir is on your user PATH but not this session's. Open a new shell to run beaver."
    }
} catch {
    # Never exit: in that same session, exit closes the caller's window before
    # the message can be read. A terminating error leaves an interactive shell
    # open with the message on screen, and still fails `powershell -File`
    # with a non-zero exit code.
    throw "install.ps1: $($_.Exception.Message)"
} finally {
    if ($tmp -and (Test-Path -LiteralPath $tmp)) {
        Remove-Item -LiteralPath $tmp -Recurse -Force
    }
}
}
