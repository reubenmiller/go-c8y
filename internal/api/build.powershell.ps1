[cmdletbinding()]
Param(
    [switch] $SkipGenerate
)

if (!$SkipGenerate) {
    # Convert the yaml specs to json
    if (!(Get-Command yaml2json -ErrorAction SilentlyContinue)) {
        Write-Error "Missing yaml2json. Please install it using 'npm install -g yamljs'"
        return
    }

    Write-Host "Converting yaml specs to json" -ForegroundColor Gray
    $WorkDir = Resolve-Path -Path "$PSScriptRoot/spec" -Relative
    yaml2json -s -r -p $WorkDir

    . $PSScriptRoot/New-C8yPowershellApi.ps1
    . $PSScriptRoot/New-C8yPowershellArguments.ps1
    . $PSScriptRoot/New-C8yApiPowershellCommand.ps1
    . $PSScriptRoot/New-C8yApiPowershellTest.ps1

    $OutputDir = Join-Path $PSScriptRoot -ChildPath "../../pkg/powershell/public"
    if (!(Test-Path $OutputDir)) {
        $null = New-Item -ItemType Directory $OutputDir
    }

    $OutputDir = Resolve-path $OutputDir

    Write-Host "Building the c8y binary"
    $c8yBinary = Resolve-Path "$PSScriptRoot/../../cmd/c8y/main.go"

    if ($IsMacOS) {
        $env:GOARCH = "amd64"
        $env:GOOS = "darwin"
        & go build -ldflags="-s -w" -o "$OutputDir/../c8y" "$c8yBinary"
    } elseif ($IsLinux) {
        $env:GOARCH = "amd64"
        $env:GOOS = "linux"
        & go build -ldflags="-s -w" -o "$OutputDir/../c8y" "$c8yBinary"
    } else {
        $env:GOARCH = "amd64"
        $env:GOOS = "windows"
        & go build -ldflags="-s -w" -o "$OutputDir/../c8y.exe" "$c8yBinary"
    }

    $SpecFiles = Get-ChildItem -Path "$PSScriptRoot/spec" -Filter "*.json"

    $ImportStatements = foreach ($iFile in $SpecFiles) {
        Write-Host ("Generating go cli code [{0}]" -f $iFile.Name) -ForegroundColor Gray
        New-C8yPowershellApi $iFile.FullName -OutputDir $OutputDir
    }
    Write-Host "`nUse the following import statements in the root cmd`n"
    $ImportStatements
}

Write-Host "Build successful! $OutputDir"
