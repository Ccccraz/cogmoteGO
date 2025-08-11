<#
.SYNOPSIS
    Install or update cogmoteGO on Windows
.DESCRIPTION
    This script downloads and installs the latest version of cogmoteGO for Windows amd64
#>

# Set error handling
$ErrorActionPreference = "Stop"

# Configuration
$REPO = "Ccccraz/cogmoteGO"
$BINARY_NAME = "cogmoteGO"
# Changed to user-specific directory
$INSTALL_DIR = "$env:LOCALAPPDATA\Programs\$BINARY_NAME"
$LATEST_RELEASE_URL = "https://api.github.com/repos/$REPO/releases/latest"
$INSTALLED_CMD = "$INSTALL_DIR\$BINARY_NAME.exe"

# Color definitions
$RED = "`e[31m"
$GREEN = "`e[32m"
$YELLOW = "`e[33m"
$BLUE = "`e[34m"
$NC = "`e[0m" # No Color

# Check if ANSI colors are supported
if ($Host.UI.RawUI -and $Host.UI.RawUI.SupportsVirtualTerminal) {
    $Host.UI.RawUI.UseVirtualTerminal = $true
}

# Show title
Write-Host "${BLUE}=== $BINARY_NAME ===${NC}"

# Get installed version
function Get-InstalledVersion {
    if (Test-Path $INSTALLED_CMD) {
        $version = & $INSTALLED_CMD --version 2>&1 | Select-String -Pattern "(\d+\.\d+\.\d+)" | ForEach-Object { $_.Matches.Groups[1].Value }
        return "v$version"
    }
    return ""
}

$INSTALLED_VERSION = Get-InstalledVersion
if ($INSTALLED_VERSION) {
    Write-Host "Installed version: ${GREEN}${INSTALLED_VERSION}${NC}"
}

# Get latest version
Write-Host "${BLUE}[1/4] ${NC}Checking latest version..."
$response = Invoke-RestMethod -Uri $LATEST_RELEASE_URL -Method Get
$LATEST_RELEASE = $response.tag_name
Write-Host "Latest version: ${GREEN}${LATEST_RELEASE}${NC}"

# Check if installation/update is needed
if ($INSTALLED_VERSION) {
    if ($INSTALLED_VERSION -eq $LATEST_RELEASE) {
        Write-Host "${GREEN}Already up to date${NC}"
        exit 0
    } else {
        Write-Host "${YELLOW}New version available ${INSTALLED_VERSION} → ${LATEST_RELEASE}${NC}"
    }
} else {
    Write-Host "${YELLOW}No installation detected, performing fresh install${NC}"
}

# Build download URL
$DOWNLOAD_URL = "https://github.com/$REPO/releases/download/$LATEST_RELEASE/${BINARY_NAME}-windows-amd64-${LATEST_RELEASE}.zip"
Write-Host "Download URL: ${YELLOW}${DOWNLOAD_URL}${NC}"

# Create temporary directory
$TMP_DIR = Join-Path $env:TEMP "${BINARY_NAME}-$(Get-Date -Format 'yyyyMMddHHmmss')"
New-Item -ItemType Directory -Path $TMP_DIR -Force | Out-Null

try {
    # Download
    Write-Host "${BLUE}[2/4] ${NC}Downloading ${BINARY_NAME}..."
    $zipFile = "$TMP_DIR\$BINARY_NAME.zip"
    Invoke-WebRequest -Uri $DOWNLOAD_URL -OutFile $zipFile -UseBasicParsing

    # Extract
    Write-Host "${BLUE}[3/4] ${NC}Extracting..."
    Expand-Archive -Path $zipFile -DestinationPath $TMP_DIR -Force

    # Install
    Write-Host "${BLUE}[4/4] ${NC}Installing to ${INSTALL_DIR}..."
    if (-not (Test-Path $INSTALL_DIR)) {
        New-Item -ItemType Directory -Path $INSTALL_DIR -Force | Out-Null
    }
    
    # Get all DLL files from the extracted directory
    $dllFiles = Get-ChildItem -Path $TMP_DIR -Filter "*.dll"
    
    # Copy all files (exe and dlls)
    $sourceExe = "$TMP_DIR\$BINARY_NAME.exe"
    if (-not (Test-Path $sourceExe)) {
        throw "Extracted executable not found at $sourceExe"
    }
    
    # Move the executable
    Move-Item -Path $sourceExe -Destination $INSTALLED_CMD -Force
    
    # Copy all DLL files
    if ($dllFiles) {
        foreach ($dll in $dllFiles) {
            $destinationDll = "$INSTALL_DIR\$($dll.Name)"
            Move-Item -Path $dll.FullName -Destination $destinationDll -Force
        }
    }

    # Add to PATH
    $path = [Environment]::GetEnvironmentVariable("PATH", "User")
    if ($path -notlike "*$INSTALL_DIR*") {
        Write-Host "${YELLOW}Adding ${INSTALL_DIR} to PATH...${NC}"
        [Environment]::SetEnvironmentVariable("PATH", "$path;$INSTALL_DIR", "User")
        # Update current session PATH
        $env:PATH += ";$INSTALL_DIR"
        Write-Host "${GREEN}Successfully added to PATH${NC}"
    }

    # Verify installation
    $NEW_VERSION = Get-InstalledVersion
    if ($NEW_VERSION -eq $LATEST_RELEASE) {
        if ($INSTALLED_VERSION) {
            Write-Host "${GREEN}Update successful! ${BINARY_NAME} updated from ${INSTALLED_VERSION} to ${NEW_VERSION}${NC}"
        } else {
            Write-Host "${GREEN}Installation successful! ${BINARY_NAME} ${NEW_VERSION} (amd64) installed to ${INSTALL_DIR}${NC}"
        }
    } else {
        throw "Installation verification failed!"
    }
}
finally {
    # Clean up temporary files
    if (Test-Path $TMP_DIR) {
        Remove-Item -Recurse -Force $TMP_DIR
    }
}
