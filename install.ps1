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

# Show title
Write-Host "=== $BINARY_NAME ===" -ForegroundColor Blue

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
    Write-Host "Installed version: $INSTALLED_VERSION" -ForegroundColor Green
}

# Get latest version
Write-Host "[1/4] Checking latest version..." -ForegroundColor Blue
$response = Invoke-RestMethod -Uri $LATEST_RELEASE_URL -Method Get
$LATEST_RELEASE = $response.tag_name
Write-Host "Latest version: $LATEST_RELEASE" -ForegroundColor Green

# Check if installation/update is needed
if ($INSTALLED_VERSION) {
    if ($INSTALLED_VERSION -eq $LATEST_RELEASE) {
        Write-Host "Already up to date" -ForegroundColor Green
        return
    } else {
        Write-Host "New version available ${INSTALLED_VERSION} → ${LATEST_RELEASE}" -ForegroundColor Yellow
    }
} else {
    Write-Host "No installation detected, performing fresh install" -ForegroundColor Yellow
}

# Build download URL
$DOWNLOAD_URL = "https://github.com/$REPO/releases/download/$LATEST_RELEASE/${BINARY_NAME}-windows-amd64-${LATEST_RELEASE}.zip"
Write-Host "Download URL: $DOWNLOAD_URL" -ForegroundColor Yellow

# Create temporary directory
$TMP_DIR = Join-Path $env:TEMP "${BINARY_NAME}-$(Get-Date -Format 'yyyyMMddHHmmss')"
New-Item -ItemType Directory -Path $TMP_DIR -Force | Out-Null

try {
    # Download
    Write-Host "[2/4] Downloading ${BINARY_NAME}..." -ForegroundColor Blue
    $zipFile = "$TMP_DIR\$BINARY_NAME.zip"
    Invoke-WebRequest -Uri $DOWNLOAD_URL -OutFile $zipFile -UseBasicParsing

    # Extract
    Write-Host "[3/4] Extracting..." -ForegroundColor Blue
    Expand-Archive -Path $zipFile -DestinationPath $TMP_DIR -Force

    # Install
    Write-Host "[4/4] Installing to ${INSTALL_DIR}..." -ForegroundColor Blue
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
        Write-Host "Adding ${INSTALL_DIR} to PATH..." -ForegroundColor Yellow
        [Environment]::SetEnvironmentVariable("PATH", "$path;$INSTALL_DIR", "User")
        # Update current session PATH
        $env:PATH += ";$INSTALL_DIR"
        Write-Host "Successfully added to PATH" -ForegroundColor Green
    }

    # Verify installation
    $NEW_VERSION = Get-InstalledVersion
    if ($NEW_VERSION -eq $LATEST_RELEASE) {
        if ($INSTALLED_VERSION) {
            Write-Host "Update successful! ${BINARY_NAME} updated from ${INSTALLED_VERSION} to ${NEW_VERSION}" -ForegroundColor Green
        } else {
            Write-Host "Installation successful! ${BINARY_NAME} ${NEW_VERSION} (amd64) installed to ${INSTALL_DIR}" -ForegroundColor Green
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