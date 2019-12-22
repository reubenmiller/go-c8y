[cmdletbinding()]
Param(
    [Parameter(
        Mandatory = $true,
        Position = 0)]
    [string] $OutputDir,

    [switch] $CompressOnly,

    # Build binaries for all
    [switch] $All
)

# Create output folder if it does not exist
if (!(Test-Path $OutputDir -PathType Container)) {
    Write-Verbose "Creating output folder [$OutputDir]"
    $null = New-Item -ItemType Directory $OutputDir
}
$OutputDir = Resolve-path $OutputDir

Write-Host "Building the c8y binary"
$c8yBinary = Resolve-Path "$PSScriptRoot/../../cmd/c8y/main.go"

$name = "c8y"

if ($All -or $IsMacOS) {
    Write-Host "Building the c8y binary [MacOS]"
    # $env:GOARCH = "amd64"
    $env:GOOS = "darwin"
    $OutputPath = Join-Path -Path $OutputDir -ChildPath "${name}.macos"
    & go build -ldflags="-s -w" -o "$OutputPath" "$c8yBinary"

    if ($LASTEXITCODE -ne 0) {
        Write-Error "Failed to build project"
        return
    }

    chmod +x "$OutputPath"

    # Compress-Archive -Path $OutputPath -DestinationPath "$OutputDir/c8y.macos.zip" -CompressionLevel Optimal -Force

    if ($CompressOnly -and (Test-Path $OutputPath)) {
        Remove-Item $OutputPath
    }
}

if ($All -or $IsLinux) {
    Write-Host "Building the c8y binary [Linux]"
    # $env:GOARCH = "amd64"
    $env:GOOS = "linux"
    $OutputPath = Join-Path -Path $OutputDir -ChildPath "${name}.linux"
    & go build -ldflags="-s -w" -o "$OutputPath" "$c8yBinary"

    if ($LASTEXITCODE -ne 0) {
        Write-Error "Failed to build project"
        return
    }

    chmod +x "$OutputPath"

    # Compress-Archive -Path $OutputPath -DestinationPath "$OutputDir/c8y.linux.zip" -Force

    if ($CompressOnly -and (Test-Path $OutputPath)) {
        Remove-Item $OutputPath
    }
}

# windows
if ($All -or !($IsMacOS -or $IsLinux)) {
    Write-Host "Building the c8y binary [Windows]"
    # $env:GOARCH = "amd64"
    $env:GOOS = "windows"
    $OutputPath = Join-Path -Path $OutputDir -ChildPath "${name}.windows.exe"
    & go build -ldflags="-s -w" -o "$OutputPath" "$c8yBinary"

    if ($LASTEXITCODE -ne 0) {
        Write-Error "Failed to build project"
        return
    }

    # Compress-Archive -Path $OutputPath -DestinationPath "$OutputDir/c8y.windows.zip" -Force

    if ($CompressOnly -and (Test-Path $OutputPath)) {
        Remove-Item $OutputPath
    }
}
