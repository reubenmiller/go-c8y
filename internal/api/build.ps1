[cmdletbinding()]
Param()

# Convert the yaml specs to json
if (!(Get-Command yaml2json -ErrorAction SilentlyContinue)) {
    Write-Error "Missing yaml2json. Please install it using 'npm install -g yamljs'"
    return
}

Write-Host "Converting yaml specs to json" -ForegroundColor Gray
$WorkDir = Resolve-Path -Path "$PSScriptRoot/spec" -Relative
yaml2json -s -r -p $WorkDir

. $PSScriptRoot/New-C8yApi.ps1
. $PSScriptRoot/New-C8yApiGoCommand.ps1
. $PSScriptRoot/New-C8yApiGoRootCommand.ps1

$OutputDir = Resolve-path (Join-Path $PSScriptRoot -ChildPath "../../pkg/cmd")

$SpecFiles = Get-ChildItem -Path "$PSScriptRoot/spec" -Filter "*.json"

$ImportStatements = foreach ($iFile in $SpecFiles) {
    Write-Host ("Generating go cli code [{0}]" -f $iFile.Name) -ForegroundColor Gray
    New-C8yApi $iFile.FullName -OutputDir $OutputDir
}
Write-Host "`nUse the following import statements in the root cmd`n"
$ImportStatements

#
# Build project
#
$BinaryDir = Resolve-Path -Path "$PSScriptRoot/../../cmd/c8y"
$OutputDir = "$PSScriptRoot/../../output"

if (!(Test-Path $OutputDir)) {
    $null = New-Item $OutputDir -ItemType Directory
    $OutputDir = Resolve-Path $OutputDir
}
& go build -o "$OutputDir/c8y.exe" "$BinaryDir/main.go"

if ($LASTEXITCODE -ne 0) {
    Write-Error "Failed to build project"
    return
}

# Create code completions
& "$OutputDir/c8y.exe" completion powershell > "$OutputDir/c8y.ps1"

Write-Host "Build successful! $OutputDir"
