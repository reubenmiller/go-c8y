[cmdletbinding()]
Param(
    [Parameter(
        Mandatory = $true,
        Position = 0)]
    [string] $OutputDir
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

if ($IsMacOS) {
    $env:GOARCH = "amd64"
    $env:GOOS = "darwin"
    & go build -ldflags="-s -w" -o "$OutputDir/${name}" "$c8yBinary"
} elseif ($IsLinux) {
    $env:GOARCH = "amd64"
    $env:GOOS = "linux"
    & go build -ldflags="-s -w" -o "$OutputDir/${name}" "$c8yBinary"
} else {
    $env:GOARCH = "amd64"
    $env:GOOS = "windows"
    & go build -ldflags="-s -w" -o "$OutputDir/${name}.exe" "$c8yBinary"
}

if ($LASTEXITCODE -ne 0) {
    Write-Error "Failed to build project"
    return
}
