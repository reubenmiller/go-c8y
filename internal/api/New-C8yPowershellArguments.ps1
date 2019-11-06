Function New-C8yPowershellArguments {
    [cmdletbinding()]
    Param(
        [Parameter(
            Mandatory = $true,
            Position = 0
        )]
        [string] $Name,

        [Parameter(
            Mandatory = $true,
            Position = 1
        )]
        [string] $Type,

        [string] $Required,

        [string] $OptionName,

        [string] $Description,

        [string] $Default,

        [switch] $ReadFromPipeline
    )

    # Format variable name
    $NameLocalVariable = $Name[0].ToString().ToUpperInvariant() + $Name.Substring(1)

    $ParameterDefinition = New-Object System.Collections.ArrayList

    if ($Required -match "true|yes") {
        $null = $ParameterDefinition.Add("Mandatory = `$true")
        $Description = "${Description} (required)"
    }

    # TODO: Do we need to add Position = x? to the ParameterDefinition

    # Add alias
    if ($UseOption) {
        $null = $ParameterDefinition.Add("Alias = `"$OptionName`"")
    }

    # Add Piped argument
    if ($Type -match "(device|source|id)" -or $ReadFromPipeline) {
        $null = $ParameterDefinition.Add("ValueFromPipeline=`$true")
        $null = $ParameterDefinition.Add("ValueFromPipelineByPropertyName=`$true")
    }

    # Type Definition
    $DataType = switch -Regex ($Type) {
        "id" { "string" }
        "json" { "hashtable" }
        "date(from|to|time)" { "string" }
        "\[\]string" { "string[]" }
        "\[\]device" { "object[]" }
        "^string$" { "string" }
        "boolean" { "switch" }
        "application" { "object[]" }
        "integer" { "long" }
        "tenant" { "object[]" }
        "strings" { "string" }
        "file" { "string" }
        "set" { "object[]" }
        default {
            Write-Error "Unsupported Type. $_"
        }
    }

    New-Object psobject -Property @{
        Name = $NameLocalVariable
        Type = $DataType
        Definition = $ParameterDefinition
        Description = "$Description"

    }
}
