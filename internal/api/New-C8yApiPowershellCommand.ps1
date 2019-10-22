Function New-C8yApiPowershellCommand {
    [cmdletbinding()]
    Param(
        [Parameter(
            Position = 0,
            ValueFromPipeline = $true,
            ValueFromPipelineByPropertyName = $true,
            Mandatory = $true
        )]
        [object[]] $Specification,

        [string] $Noun,

        [string] $OutputDir = "./"
    )

    $CmdletName = $Specification.alias.powershell
    $Name = $Specification.name
	$NameCamel = $Name[0].ToString().ToUpperInvariant() + $Name.Substring(1)
    $File = Join-Path -Path $OutputDir -ChildPath ("{0}.ps1" -f $CmdletName)
    $ResultType = $Specification.accept
    $ResultItemType = $Specification.collectionType
    $ResultSelectProperty = $Specification.listProperty

    $Verb = $Specification.alias.go

    # Powershell Confirm impact
    $CmdletConfirmImpact = "None";
    if ($CmdletName -notlike "Get*") {
        $CmdletConfirmImpact = "High"
    }

    #
    # Meta information
    #
    $Synopsis = $Specification.description
    $DescriptionLong = $Specification.descriptionLong
    $DocumentationLink = $Specification.link
    $Examples = foreach ($iExample in $Specification.examples.powershell) {
        $iExample
    }

    $CmdletDocStringBuilder = New-Object System.Text.StringBuilder

    if ($Synopsis) {
        $null = $CmdletDocStringBuilder.AppendLine(".SYNOPSIS")
        $null = $CmdletDocStringBuilder.AppendLine("${Synopsis}`n")
    }

    if ($DescriptionLong) {
        $null = $CmdletDocStringBuilder.AppendLine(".DESCRIPTION")
        $null = $CmdletDocStringBuilder.AppendLine("${DescriptionLong}`n")
    }

    #
    # Arguments
    #
    $ArgumentSources = New-Object System.Collections.ArrayList

    if ($Specification.pathParameters) {
        $null = $ArgumentSources.AddRange(([array]$Specification.pathParameters))
    }

    if ($Specification.queryParameters) {
        $null = $ArgumentSources.AddRange(([array]$Specification.queryParameters))
    }
    if ($Specification.body) {
        $null = $ArgumentSources.AddRange(([array]$Specification.body))
    }

    $CmdletParameters = New-Object System.Collections.ArrayList

    foreach ($iArg in $ArgumentSources) {
        $ArgParams = @{
            Name = $iArg.name
            Type = $iArg.type
            OptionName = $iArg.alias
            Description = $iArg.description
            Default = $iArg.default
            Required = $iArg.required
        }
        $item = New-C8yPowershellArguments @ArgParams

        # Parameter definition
        $CurrentParam = New-Object System.Text.StringBuilder
        $null = $CurrentParam.AppendLine("        # {0}" -f ($item.Description))
        $null = $CurrentParam.AppendLine("        [Parameter({0})]" -f ($item.Definition -join ",`n                   "))

        # Validate set
        if ($null -ne $iArg.validationSet) {
            [array] $ValidationSet = $iArg.validationSet | Foreach-Object { "'$_'" }
            $null = $CurrentParam.AppendLine('        [ValidateSet({0})]' -f ($ValidationSet -join ","))
        }

        $null = $CurrentParam.AppendLine('        [{0}]' -f $item.Type)
        $null = $CurrentParam.Append('        ${0}' -f $item.Name)
        $null = $CmdletParameters.Add($CurrentParam)

        # Parameter doc string
        # $null = $CmdletDocStringBuilder.AppendLine((".PARAMETER {0}" -f $item.Name))
        # $null = $CmdletDocStringBuilder.AppendLine("{0}`n" -f $item.Description)
    }

    #
    # Add common parameters
    #
    if ($ResultType -match "collection") {
        $PageSizeParam = New-Object System.Text.StringBuilder
        $null = $PageSizeParam.AppendLine('        # Maximum number of results')
        $null = $PageSizeParam.AppendLine('        [Parameter()]')
        $null = $PageSizeParam.AppendLine('        [AllowNull()]')
        $null = $PageSizeParam.AppendLine('        [AllowEmptyString()]')
        $null = $PageSizeParam.AppendLine('        [ValidateRange(1,2000)]')
        $null = $PageSizeParam.AppendLine('        [int]')
        $null = $PageSizeParam.Append('        $PageSize')
        $null = $CmdletParameters.Add($PageSizeParam)

        # If included, then the original data set will be returned
        $WithTotalPagesParam = New-Object System.Text.StringBuilder
        $null = $WithTotalPagesParam.AppendLine('        # Include total pages statistic')
        $null = $WithTotalPagesParam.AppendLine('        [Parameter()]')
        $null = $WithTotalPagesParam.AppendLine('        [switch]')
        $null = $WithTotalPagesParam.Append('        $WithTotalPages')
        $null = $CmdletParameters.Add($WithTotalPagesParam)

        #
        # Include option to expand pagination results
        #
        $IncludeAllParam = New-Object System.Text.StringBuilder
        $null = $IncludeAllParam.AppendLine('        # Include all results')
        $null = $IncludeAllParam.AppendLine('        [Parameter()]')
        $null = $IncludeAllParam.AppendLine('        [switch]')
        $null = $IncludeAllParam.Append('        $IncludeAll')
        $null = $CmdletParameters.Add($IncludeAllParam)
    }

    $RawParam = New-Object System.Text.StringBuilder
    $null = $RawParam.AppendLine('        # Include raw response including pagination information')
    $null = $RawParam.AppendLine('        [Parameter()]')
    $null = $RawParam.AppendLine('        [switch]')
    $null = $RawParam.Append('        $Raw')
    $null = $CmdletParameters.Add($RawParam)

    # Examples
    foreach ($iExample in $Examples) {
        $null = $CmdletDocStringBuilder.AppendLine(".EXAMPLE")
        $null = $CmdletDocStringBuilder.AppendLine("${iExample}`n")
    }

    # Doc link
    if ($DocumentationLink) {
        $null = $CmdletDocStringBuilder.AppendLine(".LINK " + $DocumentationLink)
    }

    #
    #
    #

    $CmdletRestMethod = $Specification.method
    $CmdletRestPath = $Specification.path

    #
    # Body
    #
    $RESTBodyBuilder = New-Object System.Text.StringBuilder
    if ($Specification.body) {
        $null = $RESTBodyBuilder.AppendLine('$body = @{}')

        foreach ($iArg in $Specification.body) {
            $argname = $iArg.name
            $prop = $iArg.property
            $type = $iArg.type

            if (!$prop) {
                $prop = $iArg.name
            }

            if ($prop) {
                if ($prop.Contains(".")) {
                    [array] $propParts = $prop -split "\."

                    if ($propParts.Count -gt 2) {
                        Write-Warning "TODO: handle nested properties with depth > 2"
                        continue
                    }
                    $rootprop = $propParts[0]
                    $nestedprop = $propParts[1]
                    $null = $RESTBodyBuilder.AppendLine("`$body[`"$rootprop`"] = @{`"`" = `"$nestedprop`"}")
                } else {
                    switch ($type) {
                        "json" {
                            # Do nothing as it is already covered by getDataFlag
                        }
                        default {
                            $null = $RESTBodyBuilder.AppendLine("`$body[`"$prop`"] = ")
                        }
                    }
                }
            }
        }
    }

    #
    # Path Parameters
    #
    $RESTPathBuilder = New-Object System.Text.StringBuilder
    foreach ($iPathParameter in $Specification.pathParameters) {
        $prop = $iPathParameter.name
        $null = $RESTPathBuilder.AppendLine("")
    }

    #
    # Query parameters
    #
    $RESTQueryBuilder = New-Object System.Text.StringBuilder
    if ($Specification.queryParameters) {
        foreach ($iQueryParameter in $Specification.queryParameters) {
            $prop = $iQueryParameter.name
            $queryParam = $iQueryParameter.property
            if (!$queryParam) {
                $queryParam = $iQueryParameter.name
            }

            switch ($iQueryParameter.type) {
                "boolean" {
                    $null = $RESTQueryBuilder.AppendLine("")
                }

                "[]device" {
                    $null = $RESTQueryBuilder.AppendLine("")
                }

                # Array of strings
                "[]string" {
                    $null = $RESTQueryBuilder.AppendLine("")
                }

                default {
                    $null = $RESTQueryBuilder.AppendLine("")
                }
            }
        }

        #
        # Add common options
        #

        $null = $RESTQueryBuilder.AppendLine(@"
    if cmd.Flags().Changed("pageSize") {
        if v, err := cmd.Flags().GetInt("pageSize"); err == nil && v > 0 {
            query.Add("pageSize", fmt.Sprintf("%d", v))
        }
    }

    if cmd.Flags().Changed("withTotalPages") {
        if v, err := cmd.Flags().GetBool("withTotalPages"); err == nil && v {
            query.Add("withTotalPages", "true")
        }
    }
"@)
        #
        # Encode query parameters to a string
        #
        $null = $RESTQueryBuilder.AppendLine(@"
    queryValue, err := url.QueryUnescape(query.Encode())

    if err != nil {
        return newSystemError("Invalid query parameter")
    }
"@)
    }


    #
    # Template
    #

    $Template = @"
# Code generated from specification version 1.0.0: DO NOT EDIT
Function $CmdletName {
<#
$($CmdletDocStringBuilder.ToString())
#>
    [cmdletbinding(SupportsShouldProcess = `$true,
                   PositionalBinding=`$true,
                   HelpUri='$DocumentationLink',
                   ConfirmImpact = '$CmdletConfirmImpact')]
    [Alias()]
    [OutputType([object])]
    Param(
$($CmdletParameters -join ",`n`n")
    )

    Begin {
        $CmdletBeginBlock
    }

    Process {
        # Get the command name
        `$CommandName = `$PSCmdlet.MyInvocation.InvocationName;
        # Get the list of parameters for the command
        `$ParameterList = (Get-Command -Name `$CommandName).Parameters;

        `$Parameters = @{}

        # Grab each parameter value, using Get-Variable
        foreach (`$Name in (`$ParameterList.Keys -notmatch "^Raw$")) {
            `$iParam = Get-Variable -Name `$Name -ErrorAction SilentlyContinue;

            if (`$iParam.Value -is [Switch]) {
                if (`$iParam.Value.IsPresent -and `$iParam) {
                    `$Parameters[`$Name] = `$true
                }
            } elseif (`$iParam.Value -is [hashtable]) {
                `$Parameters[`$Name] = "'{0}'" -f ((ConvertTo-Json `$iParam.Value -Compress) -replace '"', '\"')
            } elseif (`$iParam.Value -is [datetime]) {
                `$Parameters[`$Name] = Format-Date `$iParam.Value
            } else {
                if ("`$iParam" -notmatch "^$") {
                    `$Parameters[`$Name] = `$iParam.Value
                }
            }
        }

        Invoke-Command ``
            -Noun $Noun ``
            -Verb $Verb ``
            -Parameters `$Parameters ``
            -Type "$ResultType" ``
            -ItemType "$ResultItemType" ``
            -ResultProperty "$ResultSelectProperty" ``
            -Raw:`$Raw ``
            -IncludeAll:`$IncludeAll
    }

    End {
        $CmdletEndBlock
    }
}
"@

	# Must not include BOM!
	$Utf8NoBomEncoding = New-Object System.Text.UTF8Encoding $False
	[System.IO.File]::WriteAllLines($File, $Template, $Utf8NoBomEncoding)
}

Function New-C8yApiPowershellProcessBlock {
    [cmdletbinding()]
    Param()

@"
        `$Parameters = @{}

        # Get the command name
        `$CommandName = `$PSCmdlet.MyInvocation.InvocationName;
        # Get the list of parameters for the command
        `$ParameterList = (Get-Command -Name `$CommandName).Parameters;

        # Grab each parameter value, using Get-Variable
        foreach (`$Parameter in `$ParameterList) {
            `$Name = `$Parameter.Values.Name
            `$Value = Get-Variable -Name `$Parameter.Values.Name -ErrorAction SilentlyContinue;

            # Allow for
            if (`$Value -is [Switch]) {
                if (`$Value.IsPresent) {
                    `$Parameter[`$Name] = "`$Value".ToLowerInvariant()
                }
            } else {
                if ("`$Value" -notmatch "^$") {
                    `$Parameter[`$Name] = `$Value
                }
            }

        }

        Invoke-Command -Noun `$Noun -Verb `$Verb -Parameters

"@
}

Function New-C8yApiPowershellTest {
    [cmdletbinding()]
    Param()

    @"
`$BinaryArguments = New-Object arraylist
`$null = `$BinaryArguments.Add("alarms")
`$null = `$BinaryArguments.Add("list")

# Get the command name
`$CommandName = `$PSCmdlet.MyInvocation.InvocationName;
# Get the list of parameters for the command
`$ParameterList = (Get-Command -Name `$CommandName).Parameters;

# Grab each parameter value, using Get-Variable
foreach (`$Parameter in `$ParameterList) {
    `$Name = `$Parameter.Values.Name
    `$Value = Get-Variable -Name `$Parameter.Values.Name -ErrorAction SilentlyContinue;

    # Allow for
    if ("`$Value" -notmatch "^$") {
        `$argName = `$Name[0].ToLowerInvariant() + `$Name.SubString(1)
        `$null = `$BinaryArguments.AddRange(@("--`$Name", `$Value))
    }
}
"@
}