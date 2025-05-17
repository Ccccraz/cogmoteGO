<#
.SYNOPSIS
    Build script for Windows platform with version embedding and CGO support
.DESCRIPTION
    Builds Go project with:
    - Version info from git (local mode) or provided parameters (CI mode)
    - CGO enabled with ZMQ library from vcpkg
.PARAMETER CI
    Indicates this is a CI/CD build (will use provided version/commit instead of git)
.PARAMETER Version
    Version to embed in binary (required in CI mode)
.PARAMETER Commit
    Commit hash to embed in binary (required in CI mode)
.EXAMPLE
    # Local development build
    .\build.ps1
.EXAMPLE
    # CI/CD build with specified version
    .\build.ps1 -CI -Version "v0.0.1" -Commit "abc123"
#>

param(
    [switch]$CI,
    [string]$Version,
    [string]$Commit
)

$Target = "windows-amd64"
$ArchiveType = "zip"

$Workspace = $PSScriptRoot | Split-Path -Parent
$OutputDir = "$Workspace\build\$Target"
$DistDir = "$Workspace\dist"
$BinaryName = "cogmoteGO.exe"

# Setup vcpkg paths
$VcpkgRoot = "${Workspace}\vcpkg_installed\x64-windows"

# Verify CI mode parameters
if ($CI) {
    if (-not $Version -or -not $Commit) {
        Write-Host "❌ Need to provide Version and Commit parameters in CI mode" -ForegroundColor Red
        exit 1
    }
    Write-Host "🏗️ Running in CI/CD mode..." -ForegroundColor Cyan
}

# Create necessary directories (force creation of all required directories)
try {
    $null = New-Item -ItemType Directory -Path $OutputDir -Force -ErrorAction Stop
    $null = New-Item -ItemType Directory -Path $DistDir -Force -ErrorAction Stop
}
catch {
    Write-Host "❌ Failed to create dir: $_" -ForegroundColor Red
    exit 1
}

# Get build info
function Get-VersionInfo {
    if ($CI) {
        # CI mode: use provided version info
        $cleanVersion = $Version -replace '^v([\d.]+).*', '$1'
        return @{
            Version = $cleanVersion
            Commit  = $Commit
        }
    }
    else {
        # Local mode: get version info from git
        $gitDesc = git describe --tags 2>$null
        if ($LASTEXITCODE -ne 0 -or -not $gitDesc) {
            return @{
                Version = "dev"
                Commit  = "none"
            }
        }

        $cleanVersion = $gitDesc -replace '^v([\d.]+).*', '$1'
        return @{
            Version = $cleanVersion
            Commit  = git rev-parse --short HEAD
        }
    }
}

$versionInfo = Get-VersionInfo
$version = $versionInfo.Version
$commit = $versionInfo.Commit
$date = Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ"

$ldFlags = @(
    "-w",
    "-X 'github.com/Ccccraz/cogmoteGO/cmd.version=$version'",
    "-X 'github.com/Ccccraz/cogmoteGO/cmd.commit=$commit'",
    "-X 'github.com/Ccccraz/cogmoteGO/cmd.datetime=$date'"
) -join " "

$Env:CGO_ENABLED = "1"
$Env:CGO_CFLAGS = "-I${VcpkgRoot}\include"
$Env:CGO_LDFLAGS = "-L${VcpkgRoot}\lib -l:libzmq-mt-4_3_5.lib"

# Print build info
Write-Host "`n🔧 Setup CGO environment..." -ForegroundColor Cyan
Write-Host "├─ `$Env:CGO_ENABLED = `"1`"" -ForegroundColor DarkGray
Write-Host "├─ `$Env:CGO_CFLAGS = `"-I${VcpkgRoot}\include`"" -ForegroundColor DarkGray
Write-Host "└─ `$Env:CGO_LDFLAGS = `"-L${VcpkgRoot}\lib -l:libzmq-mt-4_3_5.lib`"`n" -ForegroundColor DarkGray

Write-Host "🚀 Start building cogmoteGO..." -ForegroundColor Cyan
Write-Host "├─ version: $version" -ForegroundColor DarkCyan
Write-Host "├─ commit: $commit" -ForegroundColor DarkCyan
Write-Host "├─ datetime: $date" -ForegroundColor DarkCyan

Push-Location $Workspace
try {
    go build -ldflags $ldFlags -o "$OutputDir\$BinaryName" $Workspace 
    
    if ($LASTEXITCODE -ne 0) {
        throw "Build failed with exit code $LASTEXITCODE"
    }

    # copy dependencies
    $dllPath = "$VcpkgRoot\bin\libzmq-mt-4_3_5.dll"
    if (Test-Path $dllPath) {
        Copy-Item -Path $dllPath -Destination $OutputDir
        Write-Host "├─ Already copied dependency: $(Split-Path $dllPath -Leaf)" -ForegroundColor DarkCyan
    }
    else {
        Write-Host "⚠️ Dependencies are not copied: $dllPath" -ForegroundColor Yellow
    }

    # package archive
    $archiveName = "cogmoteGO-$Target-v$version.$ArchiveType"
    Compress-Archive -Path "$OutputDir\*" -DestinationPath "$DistDir\$archiveName" -Force

    Write-Host "└─ ✅ build successful!" -ForegroundColor Green
    Write-Host "   ├─ binary file: $OutputDir\$BinaryName"
    Write-Host "   └─ dist: $DistDir\$archiveName"
}
catch {
    Write-Host "❌ error: $_" -ForegroundColor Red
    exit 1
}
finally {
    Pop-Location
}