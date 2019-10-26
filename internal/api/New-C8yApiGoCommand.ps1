Function New-C8yApiGoCommand {
    [cmdletbinding()]
    Param(
        [Parameter(
            Position = 0,
            ValueFromPipeline = $true,
            ValueFromPipelineByPropertyName = $true,
            Mandatory = $true
        )]
        [object[]] $Specification,

        [string] $OutputDir = "./"
    )

    $Name = $Specification.name
	$NameCamel = $Name[0].ToString().ToUpperInvariant() + $Name.Substring(1)
	$File = Join-Path -Path $OutputDir -ChildPath ("{0}Cmd.go" -f $Name)


    #
    # Meta information
    #
    $Use = $Specification.alias.go
    $Description = $Specification.description
    $DescriptionLong = $Specification.descriptionLong
    $Examples = foreach ($iExample in $Specification.examples.go) {
        $iExample
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

    $CommandArgs = foreach ($iArg in $ArgumentSources) {
        $ArgParams = @{
            Name = $iArg.name
            Type = $iArg.type
            OptionName = $iArg.alias
            Description = $iArg.description
            Default = $iArg.default
            Required = $iArg.required
        }
        Get-C8yGoArgs @ArgParams
    }

    $RESTPath = $Specification.path
    $RESTMethod = $Specification.method

    #
    # Body
    #
    $RESTBodyBuilder = New-Object System.Text.StringBuilder
    if ($Specification.body) {
        $null = $RESTBodyBuilder.AppendLine('body = getDataFlag(cmd)')

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

                    switch -Regex ($type) {
                        "(\[\]device|application)" {
                            $null = $RESTBodyBuilder.AppendLine(@"
    if v, err := cmd.Flags().GetStringSlice("${argname}"); err == nil {
        for _, iValue := range v {
            if _, exists := body["$($propParts[0])"]; !exists {
                body["$($propParts[0])"] = make(map[string]interface{})
            }
            body["$($propParts[0])"].(map[string]interface{})["$($propParts[1])"] = iValue
        }
    } else {
        return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "${argname}", err))
    }
"@)
                        }

                        default {
                            $null = $RESTBodyBuilder.AppendLine(@"
    if v, err := cmd.Flags().GetString("${argname}") ; err == nil && v != "" {
        if _, exists := body["${argname}"]; !exists {
            body["$($propParts[0])"] = make(map[string]interface{})
        }
        body["$($propParts[0])"].(map[string]interface{})["$($propParts[1])"] = v
    }
"@)
                        }
                    }

                } else {
                    switch -Regex ($type) {
                        "json" {
                            # Do nothing as it is already covered by getDataFlag
                        }
                        "application" {
                            $null = $RESTBodyBuilder.AppendLine(@"
    if v, err := cmd.Flags().GetStringSlice("${argname}") ; err == nil && v != "" {
        if len(v) > 0 {
            body["${prop}"] = v[0]
        }
    }
"@)
                        }
                        default {
                            $null = $RESTBodyBuilder.AppendLine(@"
    if v, err := cmd.Flags().GetString("${argname}") ; err == nil && v != "" {
        body["${prop}"] = v
    }
"@)
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

        switch -Regex ($iPathParameter.type) {
            "(\[\]device|application)" {
                $null = $RESTPathBuilder.AppendLine(@"
    if v, err := cmd.Flags().GetStringSlice("${prop}"); err == nil {
        for _, iValue := range v {
            pathParameters["${prop}"] = iValue
        }
    } else {
        return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "${prop}", err))
    }
"@)
            }
            default {
                $null = $RESTPathBuilder.AppendLine(@"
    if v, err := cmd.Flags().GetString("${prop}"); err == nil {
        pathParameters["${prop}"] = v
    } else {
        return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "${prop}", err))
    }
"@)
            }
        }
    }

    #
    # Query parameters
    #
    $RESTQueryBuilder = New-Object System.Text.StringBuilder
    $null = $RESTQueryBuilder.AppendLine('query := url.Values{}')
    if ($Specification.queryParameters) {
        foreach ($iQueryParameter in $Specification.queryParameters) {
            $prop = $iQueryParameter.name
            $queryParam = $iQueryParameter.property
            if (!$queryParam) {
                $queryParam = $iQueryParameter.name
            }

            switch ($iQueryParameter.type) {
                "boolean" {
                    $null = $RESTQueryBuilder.AppendLine(@"
    if v, err := cmd.Flags().GetBool("${prop}"); err == nil {
        if v {
            query.Add("${queryParam}", "true")
        }
    } else {
        return newUserError("Flag does not exist")
    }
"@)
                }

                "[]device" {
                    $null = $RESTQueryBuilder.AppendLine(@"
    ${prop}Value := getFormattedDeviceSlice(cmd, args, "${prop}")
    if len(${prop}Value) > 0 {
        for _, item := range ${prop}Value {
            if item != "" {
                query.Add("${queryParam}", newIDValue(item).GetID())
            }
        }
    }
"@)
                }

                # Array of strings
                "[]string" {
                    $null = $RESTQueryBuilder.AppendLine(@"
    if v, err := cmd.Flags().GetStringSlice("${prop}"); err == nil {
        if len(v) > 0 {
            for _, item := range v {
                if item != "" {
                    query.Add("${queryParam}", item)
                }
            }
        }
    } else {
        return newUserError("Flag does not exist")
    }
"@)
                }

                default {
                    $null = $RESTQueryBuilder.AppendLine(@"
    if v, err := cmd.Flags().GetString("${prop}"); err == nil {
        if v != "" {
            query.Add("${queryParam}", url.QueryEscape(v))
        }
    } else {
        return newUserError("Flag does not exist")
    }
"@)
                }
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


    #
    # Template
    #
    $Template = @"
// Code generated from specification version 1.0.0: DO NOT EDIT
package cmd

import (
    "context"
    "fmt"
    "net/url"

    "github.com/fatih/color"
    "github.com/reubenmiller/go-c8y/pkg/c8y"
    "github.com/spf13/cobra"
    "github.com/tidwall/pretty"
)

type ${Name}Cmd struct {
    *baseCmd
}

func new${NameCamel}Cmd() *${Name}Cmd {
	ccmd := &${Name}Cmd{}

	cmd := &cobra.Command{
		Use:   "$Use",
		Short: "$Description",
		Long:  ``$DescriptionLong``,
        Example: ``
        $($Examples -join "`n`n")
		``,
		RunE: ccmd.${Name},
    }

    cmd.SilenceUsage = true

    $($CommandArgs.SetFlag -join "`n	")

    // Required flags
    $($CommandArgs.Required -join "`n	")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *${Name}Cmd) ${Name}(cmd *cobra.Command, args []string) error {

    // query parameters
    queryValue := url.QueryEscape("")
    $RESTQueryBuilder

    // body
    var body map[string]interface{}
    $RESTBodyBuilder

    // path parameters
    pathParameters := make(map[string]string)
    $RESTPathBuilder
    path := replacePathParameters("${RESTPath}", pathParameters)

    return n.do${NameCamel}("${RESTMethod}", path, queryValue, body)
}

func (n *${Name}Cmd) do${NameCamel}(method string, path string, query string, body map[string]interface{}) error {
    resp, err := client.SendRequest(
		context.Background(),
		c8y.RequestOptions{
			Method:       method,
            Path:         path,
            Query:        query,
            Body:         body,
            IgnoreAccept: false,
            DryRun:       globalFlagDryRun,
		})

    if err != nil {
        color.Set(color.FgRed, color.Bold)
    }

    if resp != nil && resp.JSONData != nil {
        if globalFlagPrettyPrint {
            fmt.Printf("%s\n", pretty.Pretty([]byte(*resp.JSONData)))
        } else {
            fmt.Printf("%s\n", *resp.JSONData)
        }
    }

    color.Unset()

	if err != nil {
		return newSystemError("command failed", err)
	}
	return nil
}
"@

	# Must not include BOM!
	$Utf8NoBomEncoding = New-Object System.Text.UTF8Encoding $False
	[System.IO.File]::WriteAllLines($File, $Template, $Utf8NoBomEncoding)

	# Auto format code
	& gofmt -w $File
}

Function Get-C8yGoArgs {
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

        [string] $Default
    )

    $NameLocalVariable = $Name[0].ToString().ToLowerInvariant() + $Name.Substring(1) + "Value"

    if ($Required -match "true|yes") {
        $Description = "${Description} (required)"
    }

    $Entry = switch -Regex ($Type) {
        "id" {
            @{
                SetFlag = "addIDFlag(cmd)"
                GetFlag = "GetIDs(cmd, args)"
            }
        }

        "application" {
            @{
                SetFlag = "addApplicationFlag(cmd)"
                # GetFlag = "GetIDs(cmd, args)"
            }
        }

        "json" {
            @{
                SetFlag = "addDataFlag(cmd)"
                GetFlag = "getDataFlag(cmd)"
            }
        }

        "date(from|to|time)" {
            $SetFlag = if ($UseOption) {
                'cmd.Flags().StringP("{0}", "{1}", "{2}", "{3}{4}")' -f $Name, $OptionName, $Default, $Description
            } else {
                'cmd.Flags().String("{0}", "{1}", "{2}")' -f $Name, $Default, $Description
            }

            $GetFlag = @"
    ${NameLocalVariable}, err := cmd.Flags().GetString("$Name");
    if  err != nil {
        return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "$Name", err))
    }
"@
            @{
                SetFlag = $SetFlag
                GetFlag = $GetFlag
            }
        }

        "\[\]string" {
            $SetFlag = if ($UseOption) {
                "cmd.Flags().StringSlice(`"${Name}`", `"${OptionName}`", []string{`"${Default}`"}, `"${Description}`")"
            } else {
                "cmd.Flags().StringSlice(`"${Name}`", []string{`"${Default}`"}, `"${Description}`")"
            }

            $GetFlag = @"
    ${NameLocalVariable}, err := cmd.Flags().GetStringSlice("$Name");
    if  err != nil {
        return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "$Name", err))
    }
"@

            @{
                SetFlag = $SetFlag
                GetFlag = $GetFlag
            }
        }

        "\[\]device" {
            $SetFlag = if ($UseOption) {
                "cmd.Flags().StringSliceP(`"${Name}`", []string{`"${Default}`"}, `"${OptionName}`", `"${Description}`")"
            } else {
                "cmd.Flags().StringSlice(`"${Name}`", []string{`"${Default}`"}, `"${Description}`")"
            }

            $GetFlag = @"
    ${NameLocalVariable} := getFormattedDeviceSlice(cmd, args, "$Name")
"@

            @{
                SetFlag = $SetFlag
                GetFlag = $GetFlag
            }
        }

        "^string$" {
            $SetFlag = if ($UseOption) {
                'cmd.Flags().StringP("{0}", "{1}", "{2}", "{3}")' -f $Name, $OptionName, $Default, $Description
            } else {
                'cmd.Flags().String("{0}", "{1}", "{2}")' -f $Name, $Default, $Description
            }

            $GetFlag = @"
    ${NameLocalVariable}, err := cmd.Flags().GetString("$Name");
    if  err != nil {
        return newUserError("Flag does not exist")
    }
"@


            @{
                SetFlag = $SetFlag
                GetFlag = $GetFlag
            }
        }

        "boolean" {
            if (!$Default) {
                $Default = "false"
            }
            $SetFlag = if ($UseOption) {
                'cmd.Flags().BoolP("{0}", "{1}", {2}, "{3}")' -f $Name, $OptionName, $Default, $Description
            } else {
                'cmd.Flags().Bool("{0}", {1}, "{2}")' -f $Name, $Default, $Description
            }

            $GetFlag = @"
    ${NameLocalVariable}, err := cmd.Flags().GetBool("$Name");
    if  err != nil {
        return newUserError("Flag does not exist")
    }
"@

            @{
                SetFlag = $SetFlag
                GetFlag = $GetFlag
            }
        }
    }

    # Set required flag
    if ($Required -match "true|yes") {
        $Entry | Add-Member -MemberType NoteProperty -Name "Required" -Value "cmd.MarkFlagRequired(`"${Name}`")"
        # $Entry.Required = "cmd.MarkFlagRequired(`"${Name}`")"
    }

    $Entry
}

