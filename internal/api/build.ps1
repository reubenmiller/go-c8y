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

$OutputDir = Resolve-path (Join-Path $PSScriptRoot -ChildPath "../../pkg/cmd")

$SpecFiles = Get-ChildItem -Path "$PSScriptRoot/spec" -Filter "*.json"

foreach ($iFile in $SpecFiles) {
    Write-Host ("Generating go cli code [{0}]" -f $iFile.Name) -ForegroundColor Gray
    New-C8yApi $iFile.FullName -OutputDir $OutputDir
}
