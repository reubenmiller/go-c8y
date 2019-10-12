[cmdletbinding()]
Param(

)

. $PSScriptRoot/New-C8yApi.ps1
. $PSScriptRoot/New-C8yApiGoCommand.ps1

$OutputDir = Resolve-path (Join-Path $PSScriptRoot -ChildPath "../../pkg/cmd")

New-C8yApi "$PSScriptRoot/alarms.json" -OutputDir $OutputDir
