# Request Inspection in Results

The `Result` type now includes the `Request *http.Request` field, which allows you to inspect the HTTP request that was (or would have been) sent. This is particularly useful for:

1. **Dry run mode** - Inspect requests without sending them
2. **Debugging** - See exactly what was sent to the server
3. **CLI tools** - Format requests for display (curl, markdown, JSON, etc.)

## Security Note

When dry run mode is enabled, request details are logged for debugging purposes. However, sensitive headers are automatically redacted to prevent credential leakage:

**Redacted headers:**
- `Authorization`
- `Cookie` / `Set-Cookie`
- `X-XSRF-Token` / `X-CSRF-Token`
- `API-Key` / `X-API-Key` / `ApiKey`
- `X-Auth-Token`
- `Proxy-Authorization`

These headers will appear in logs as `[REDACTED]` while still being sent in the actual HTTP request.

### Disabling Redaction for Debugging

⚠️ **WARNING**: Disabling header redaction will expose sensitive credentials in logs. Only use this for debugging in secure environments.

By default, sensitive headers are redacted. If you need to see the actual header values for debugging purposes, you can explicitly disable redaction:

```go
// Enable dry run with visible sensitive headers (for debugging only)
ctx := context.Background()
ctx = c8y_api.WithDryRun(ctx, true)
ctx = c8y_api.WithRedactHeaders(ctx, false) // Disable redaction

result := client.ManagedObjects.Get(ctx, "12345", managedobjects.GetOptions{})
// The log output will now show actual Authorization headers
```

**Best practices:**
- Only disable redaction temporarily during active debugging sessions
- Never commit code with redaction disabled
- Be careful when sharing logs that contain unredacted headers
- Re-enable redaction (the default) as soon as debugging is complete

## Usage Example

### Basic Request Inspection

```go
ctx := c8y_api.WithDryRun(context.Background(), true)
result := client.ManagedObjects.Get(ctx, "12345", managedobjects.GetOptions{})

if result.Request != nil {
    fmt.Println("Method:", result.Request.Method)
    fmt.Println("URL:", result.Request.URL)
    fmt.Println("Headers:", result.Request.Header)
}
```

### Format as Curl Command

```go
func formatAsCurl(req *http.Request) string {
    cmd := "curl -X " + req.Method

    // Add headers
    for key, values := range req.Header {
        for _, value := range values {
            cmd += " \\\n  -H '" + key + ": " + value + "'"
        }
    }

    // Add URL
    cmd += " \\\n  '" + req.URL.String() + "'"

    // Add body for POST/PUT
    if req.Body != nil && (req.Method == "POST" || req.Method == "PUT") {
        body, _ := io.ReadAll(req.Body)
        cmd += " \\\n  -d '" + string(body) + "'"
    }

    return cmd
}

// Usage
result := client.ManagedObjects.Create(ctx, data)
curlCmd := formatAsCurl(result.Request)
fmt.Println(curlCmd)
```

### Format as Markdown

```go
func formatAsMarkdown(req *http.Request) string {
    var md strings.Builder
    
    md.WriteString("## Request Details\n\n")
    md.WriteString(fmt.Sprintf("**Method:** `%s`\n\n", req.Method))
    md.WriteString(fmt.Sprintf("**URL:** `%s`\n\n", req.URL))
    
    md.WriteString("### Headers\n\n")
    md.WriteString("```\n")
    for key, values := range req.Header {
        for _, value := range values {
            md.WriteString(fmt.Sprintf("%s: %s\n", key, value))
        }
    }
    md.WriteString("```\n\n")
    
    if req.Body != nil {
        body, _ := io.ReadAll(req.Body)
        md.WriteString("### Body\n\n")
        md.WriteString("```json\n")
        md.WriteString(string(body))
        md.WriteString("\n```\n")
    }
    
    return md.String()
}
```

### Format as JSON

```go
type RequestInfo struct {
    Method  string            `json:"method"`
    URL     string            `json:"url"`
    Headers map[string]string `json:"headers"`
    Body    json.RawMessage   `json:"body,omitempty"`
}

func formatAsJSON(req *http.Request) ([]byte, error) {
    info := RequestInfo{
        Method:  req.Method,
        URL:     req.URL.String(),
        Headers: make(map[string]string),
    }
    
    for key, values := range req.Header {
        info.Headers[key] = strings.Join(values, ", ")
    }
    
    if req.Body != nil {
        body, _ := io.ReadAll(req.Body)
        info.Body = json.RawMessage(body)
    }
    
    return json.MarshalIndent(info, "", "  ")
}
```

## Use Cases for go-c8y-cli

For go-c8y-cli, you can now:

1. **Show request details before confirmation:**
   ```go
   result := client.ManagedObjects.Delete(ctx, id, options)
   
   fmt.Println("About to execute:")
   fmt.Println(formatAsCurl(result.Request))
   fmt.Print("Continue? (y/n): ")
   ```

2. **Display in different formats based on flags:**
   ```go
   switch outputFormat {
   case "curl":
       fmt.Println(formatAsCurl(result.Request))
   case "markdown":
       fmt.Println(formatAsMarkdown(result.Request))
   case "json":
       json, _ := formatAsJSON(result.Request)
       fmt.Println(string(json))
   }
   ```

3. **Log requests for debugging:**
   ```go
   if debug {
       log.Printf("Request: %s %s", result.Request.Method, result.Request.URL)
       log.Printf("Headers: %v", result.Request.Header)
   }
   ```

## Implementation Details

- The `Request` field is populated **only in dry run mode** to avoid overhead during normal operations
- The field is populated by all execute functions in `core/execute.go`
- It's available for both successful and failed requests
- The request is captured after all middleware has run, so it includes all headers, auth, etc.
- Works with dry run mode (returns the request that *would* have been sent)
- In normal mode (without dry run), `Request` will be `nil`

## Testing

See `test/c8y_api_test/request_inspection_test.go` for complete examples.
