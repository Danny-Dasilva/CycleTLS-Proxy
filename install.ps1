# CycleTLS-Proxy Installation Script for Windows
# Downloads latest release from GitHub and installs to %LOCALAPPDATA%
# Adds to PATH environment variable

[CmdletBinding()]
param(
    [string]$Version = "",
    [string]$InstallDir = "",
    [string]$GitHubToken = "",
    [switch]$Help
)

# Configuration
$RepoOwner = "Danny-Dasilva"
$RepoName = "CycleTLS-Proxy"
$BinaryName = "cycletls-proxy.exe"
$DefaultInstallDir = "$env:LOCALAPPDATA\cycletls-proxy"

# Set error action preference
$ErrorActionPreference = "Stop"

# Color functions for better output
function Write-Info {
    param([string]$Message)
    Write-Host "ℹ️  $Message" -ForegroundColor Blue
}

function Write-Success {
    param([string]$Message)
    Write-Host "✅ $Message" -ForegroundColor Green
}

function Write-Warning {
    param([string]$Message)
    Write-Host "⚠️  $Message" -ForegroundColor Yellow
}

function Write-Error {
    param([string]$Message)
    Write-Host "❌ $Message" -ForegroundColor Red
}

function Show-Usage {
    @"
CycleTLS-Proxy Installation Script for Windows

USAGE:
    .\install.ps1 [OPTIONS]

OPTIONS:
    -Version VERSION        Install specific version (default: latest)
    -InstallDir DIRECTORY   Install directory (default: $DefaultInstallDir)
    -GitHubToken TOKEN      GitHub personal access token for private repos
    -Help                   Show this help message

EXAMPLES:
    .\install.ps1                           # Install latest version
    .\install.ps1 -Version v1.2.3          # Install specific version
    .\install.ps1 -InstallDir C:\Tools      # Install to custom directory
    .\install.ps1 -GitHubToken ghp_xxx...   # Use GitHub token

REQUIREMENTS:
    - Windows PowerShell 5.0 or later / PowerShell Core 6.0+
    - Internet connection
    - Administrator privileges (for PATH modification)

"@
}

function Test-Prerequisites {
    Write-Info "Checking prerequisites..."
    
    # Check PowerShell version
    if ($PSVersionTable.PSVersion.Major -lt 5) {
        Write-Error "PowerShell 5.0 or later is required. Current version: $($PSVersionTable.PSVersion)"
        throw "Unsupported PowerShell version"
    }
    
    # Check if running as administrator for PATH modification
    $currentPrincipal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
    $isAdmin = $currentPrincipal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
    
    if (-not $isAdmin) {
        Write-Warning "Not running as administrator. PATH modification may fail."
        Write-Info "Consider running PowerShell as Administrator for full functionality."
    }
    
    Write-Success "Prerequisites check completed"
}

function Get-Architecture {
    $arch = $env:PROCESSOR_ARCHITECTURE
    switch ($arch) {
        "AMD64" { return "amd64" }
        "ARM64" { return "arm64" }
        "x86" { return "386" }
        default {
            Write-Error "Unsupported architecture: $arch"
            throw "Unsupported architecture"
        }
    }
}

function Get-LatestVersion {
    param([string]$Token = "")
    
    $apiUrl = "https://api.github.com/repos/$RepoOwner/$RepoName/releases/latest"
    $headers = @{}
    
    if ($Token) {
        $headers["Authorization"] = "token $Token"
    }
    
    Write-Info "Fetching latest release information..."
    
    try {
        $response = Invoke-RestMethod -Uri $apiUrl -Headers $headers -Method Get
        $version = $response.tag_name
        
        if (-not $version) {
            throw "Could not determine latest version from API response"
        }
        
        return $version
    }
    catch {
        Write-Error "Failed to fetch release information: $($_.Exception.Message)"
        Write-Info "Please specify a version manually with -Version parameter"
        throw
    }
}

function Download-Binary {
    param(
        [string]$Version,
        [string]$Architecture,
        [string]$Token = ""
    )
    
    $platform = "windows_$Architecture"
    $downloadUrl = "https://github.com/$RepoOwner/$RepoName/releases/download/$Version/${BinaryName.Replace('.exe', '')}_${Version}_${platform}.zip"
    
    $tempDir = New-TemporaryFile | ForEach-Object { Remove-Item $_; New-Item -ItemType Directory -Path $_ }
    $archiveFile = Join-Path $tempDir "cycletls-proxy.zip"
    
    $headers = @{}
    if ($Token) {
        $headers["Authorization"] = "token $Token"
    }
    
    Write-Info "Downloading $BinaryName $Version for $platform..."
    Write-Info "URL: $downloadUrl"
    
    try {
        # Download with progress
        $webClient = New-Object System.Net.WebClient
        if ($Token) {
            $webClient.Headers.Add("Authorization", "token $Token")
        }
        
        # Add progress handler
        Register-ObjectEvent -InputObject $webClient -EventName DownloadProgressChanged -Action {
            $progressPercentage = $Event.SourceEventArgs.ProgressPercentage
            Write-Progress -Activity "Downloading CycleTLS-Proxy" -Status "$progressPercentage% Complete" -PercentComplete $progressPercentage
        } | Out-Null
        
        $webClient.DownloadFile($downloadUrl, $archiveFile)
        $webClient.Dispose()
        Write-Progress -Activity "Downloading CycleTLS-Proxy" -Completed
        
        # Verify download
        if (-not (Test-Path $archiveFile) -or (Get-Item $archiveFile).Length -eq 0) {
            throw "Downloaded file is empty or missing"
        }
        
        # Extract archive
        Write-Info "Extracting archive..."
        Expand-Archive -Path $archiveFile -DestinationPath $tempDir -Force
        
        # Find binary
        $binaryPath = Join-Path $tempDir $BinaryName
        if (-not (Test-Path $binaryPath)) {
            # Search for executable in subdirectories
            $foundBinary = Get-ChildItem -Path $tempDir -Recurse -Name "*.exe" | Where-Object { $_ -like "*cycletls*" } | Select-Object -First 1
            if ($foundBinary) {
                $binaryPath = Join-Path $tempDir $foundBinary
            } else {
                Write-Error "Could not find $BinaryName in the archive"
                Write-Info "Archive contents:"
                Get-ChildItem -Path $tempDir -Recurse | ForEach-Object { Write-Host $_.FullName }
                throw "Binary not found in archive"
            }
        }
        
        return $binaryPath
    }
    catch {
        if ($tempDir -and (Test-Path $tempDir)) {
            Remove-Item -Path $tempDir -Recurse -Force -ErrorAction SilentlyContinue
        }
        throw
    }
}

function Install-Binary {
    param(
        [string]$BinaryPath,
        [string]$InstallDirectory
    )
    
    $installPath = Join-Path $InstallDirectory $BinaryName
    
    # Create install directory if it doesn't exist
    if (-not (Test-Path $InstallDirectory)) {
        Write-Info "Creating install directory: $InstallDirectory"
        New-Item -ItemType Directory -Path $InstallDirectory -Force | Out-Null
    }
    
    # Check if we can write to the install directory
    try {
        $testFile = Join-Path $InstallDirectory "test_write.tmp"
        New-Item -ItemType File -Path $testFile -Force | Out-Null
        Remove-Item -Path $testFile -Force
    }
    catch {
        Write-Error "No write permission to install directory: $InstallDirectory"
        throw "Access denied to install directory"
    }
    
    # Install binary
    Write-Info "Installing $BinaryName to $installPath..."
    Copy-Item -Path $BinaryPath -Destination $installPath -Force
    
    # Verify installation
    if (-not (Test-Path $installPath)) {
        throw "Failed to copy binary to $installPath"
    }
    
    Write-Success "$BinaryName installed successfully to $installPath"
    return $installPath
}

function Add-ToPath {
    param([string]$Directory)
    
    Write-Info "Adding $Directory to PATH..."
    
    try {
        # Get current PATH
        $currentPath = [Environment]::GetEnvironmentVariable("PATH", [EnvironmentVariableTarget]::User)
        
        # Check if directory is already in PATH
        if ($currentPath -split ";" | Where-Object { $_ -eq $Directory }) {
            Write-Info "Directory already in PATH"
            return
        }
        
        # Add to PATH
        $newPath = if ($currentPath) { "$currentPath;$Directory" } else { $Directory }
        [Environment]::SetEnvironmentVariable("PATH", $newPath, [EnvironmentVariableTarget]::User)
        
        # Update current session PATH
        $env:PATH = "$env:PATH;$Directory"
        
        Write-Success "Added $Directory to PATH"
        Write-Warning "Restart your terminal or PowerShell session for PATH changes to take effect"
    }
    catch {
        Write-Warning "Failed to add directory to PATH: $($_.Exception.Message)"
        Write-Info "You can manually add $Directory to your PATH environment variable"
    }
}

function Test-Installation {
    param([string]$InstallPath)
    
    Write-Info "Verifying installation..."
    
    # Check if binary exists
    if (-not (Test-Path $InstallPath)) {
        Write-Error "Binary not found: $InstallPath"
        throw "Installation verification failed"
    }
    
    # Try to run the binary
    Write-Info "Testing binary..."
    try {
        $output = & $InstallPath --version 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-Success "Binary is working correctly"
        } else {
            # Try alternative version flags
            $output = & $InstallPath -v 2>&1
            if ($LASTEXITCODE -eq 0) {
                Write-Success "Binary is working correctly"
            } else {
                Write-Warning "Binary installed but could not verify functionality"
                Write-Info "You can try running: $BinaryName --help"
            }
        }
    }
    catch {
        Write-Warning "Binary installed but could not test functionality: $($_.Exception.Message)"
        Write-Info "You can try running: $BinaryName --help"
    }
}

function Main {
    Write-Host "CycleTLS-Proxy Installation Script for Windows" -ForegroundColor Cyan
    Write-Host "===============================================" -ForegroundColor Cyan
    Write-Host ""
    
    if ($Help) {
        Show-Usage
        return
    }
    
    try {
        # Test prerequisites
        Test-Prerequisites
        
        # Set install directory
        if (-not $InstallDir) {
            $InstallDir = $DefaultInstallDir
        }
        
        # Detect architecture
        $architecture = Get-Architecture
        Write-Info "Detected architecture: $architecture"
        
        # Get version
        if (-not $Version) {
            $Version = Get-LatestVersion -Token $GitHubToken
        }
        Write-Info "Installing version: $Version"
        
        # Download binary
        $binaryPath = Download-Binary -Version $Version -Architecture $architecture -Token $GitHubToken
        
        # Install binary
        $installPath = Install-Binary -BinaryPath $binaryPath -InstallDirectory $InstallDir
        
        # Add to PATH
        Add-ToPath -Directory $InstallDir
        
        # Verify installation
        Test-Installation -InstallPath $installPath
        
        Write-Host ""
        Write-Success "Installation completed successfully!"
        Write-Info "Run '$BinaryName --help' to get started"
        Write-Info "You may need to restart your terminal for PATH changes to take effect"
    }
    catch {
        Write-Error "Installation failed: $($_.Exception.Message)"
        exit 1
    }
    finally {
        # Cleanup temp directory if it exists
        if ($tempDir -and (Test-Path $tempDir)) {
            Remove-Item -Path $tempDir -Recurse -Force -ErrorAction SilentlyContinue
        }
    }
}

# Only run main if script is executed directly (not dot-sourced)
if ($MyInvocation.InvocationName -ne ".") {
    Main
}