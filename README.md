# EnvParser

## Overview

EnvParser is a powerful and flexible Go package designed to streamline the process of loading and managing environment variables from `.env` files in Go applications. Built with simplicity and robustness in mind, it provides advanced features like automatic type conversion, variable substitution, AES-encrypted value handling, and support for multiple configuration files. EnvParser is ideal for developers who need a reliable way to handle environment configurations with diverse data types, making it suitable for both simple and complex applications.

## Features

- **Flexible `.env` File Parsing**: Load environment variables from customizable `.env` files with support for user-defined filenames and paths.
- **Variable Substitution**: Resolve variables within values using `${VAR}` or `$VAR` syntax, referencing other environment variables.
- **Automatic Type Conversion**: Convert string values to appropriate Go types, including `string`, `bool`, `int`, `float`, `list`, `tuple`, `dict`, and more.
- **Encrypted Value Support**: Decrypt AES-encrypted values prefixed with `ENC()` or `enc()` using a provided key.
- **Multiple File Support**: Parse additional `.env` files alongside the primary file for layered configuration.
- **Project Root Detection**: Automatically locate the project root using markers like `go.mod`, `.git`, `.project-root`, or `.root`.
- **Debug Mode**: Enable detailed logging for troubleshooting and parsing insights.
- **Functional Options**: Configure the parser with options for filename, root path, extra files, and debug mode.

## Installation

To use EnvParser in your Go project, import it:

```bash
go get github.com/alexanderthegreat96/envparser/v3
```

## Supported Data Types

EnvParser supports the following data types for automatic conversion:

- **`str`** or **`string`**: Plain text strings (e.g., `"hello"` → `hello`).
- **`bool`** or **`boolean`**: Boolean values (`true`, `false`, `1`, `0`, etc.).
- **`int`** or **`integer`**: Integer numbers (e.g., `"123"` → `123`).
- **`float`**: Floating-point numbers (e.g., `"3.14"` → `3.14`). Integers in float format (e.g., `"42.0"`) are converted to `int` if they have no decimal part.
- **`list`** or **`array`**: JSON-style arrays (e.g., `"[1, 2, 3]"` → `[1, 2, 3]`).
- **`tuple`**: Tuple-like structures parsed as arrays (e.g., `"(1, 2, 3)"` → `[1, 2, 3]`).
- **`dict`**, **`map`**, or **`json`**: JSON-style objects (e.g., `"{'key': 'value'}"` → `map[string]interface{"key": "value"}`).

## Example `.env` File

Below is an example `.env` file showcasing various data types supported by EnvParser:

```
# .env
APP_NAME="Just an app"
IS_DEV_MODE=true
REDIS_HOST=hostname
REDIS_PORT=6379
REDIS_PASS=my-pass
REDIS_GLOBAL_CACHE_KEY=ubi-go
UBISOFT_ACCOUNTS=[{"email": "someone@gmail.com", "password": "this"},{"email": "someone-else", "password": "another-one"}]
DATABASES=(alex, someone, else, 12)
DATA={"testing-this": 12, "first-name": "alex"}
LIST_OF_USERS=["michael", "joe", "stan", 123, true, false]
```

## Usage Example

The following Go program demonstrates how to initialize EnvParser and extract all variables from the example `.env` file, showcasing the handling of various data types:

```go
package main

import (
    "fmt"
    "github.com/yourusername/envparser"
)

func main() {
    // Initialize the parser
    env, err := envparser.New(
        envparser.WithFilename(".env"),
        envparser.WithRootPath(true),
        envparser.WithDebug(true),
    )
    if err != nil {
        fmt.Println("Error initializing parser:", err)
        return
    }

    // Check for parsing errors
    if err := env.GetError(); err != "" {
        fmt.Println("Parsing error:", err)
        return
    }

    // Extract specific values with type conversion
    appName, err := env.GetValue("APP_NAME", "string", "DefaultApp")
    if err != nil {
        fmt.Println("Error getting APP_NAME:", err)
        return
    }
    fmt.Printf("APP_NAME: %v (%T)\n", appName, appName)

    isDevMode, err := env.GetValue("IS_DEV_MODE", "bool", false)
    if err != nil {
        fmt.Println("Error getting IS_DEV_MODE:", err)
        return
    }
    fmt.Printf("IS_DEV_MODE: %v (%T)\n", isDevMode, isDevMode)

    redisHost, err := env.GetValue("REDIS_HOST", "string", "")
    if err != nil {
        fmt.Println("Error getting REDIS_HOST:", err)
        return
    }
    fmt.Printf("REDIS_HOST: %v (%T)\n", redisHost, redisHost)

    redisPort, err := env.GetValue("REDIS_PORT", "int", 0)
    if err != nil {
        fmt.Println("Error getting REDIS_PORT:", err)
        return
    }
    fmt.Printf("REDIS_PORT: %v (%T)\n", redisPort, redisPort)

    redisPass, err := env.GetValue("REDIS_PASS", "string", "")
    if err != nil {
        fmt.Println("Error getting REDIS_PASS:", err)
        return
    }
    fmt.Printf("REDIS_PASS: %v (%T)\n", redisPass, redisPass)

    redisCacheKey, err := env.GetValue("REDIS_GLOBAL_CACHE_KEY", "string", "")
    if err != nil {
        fmt.Println("Error getting REDIS_GLOBAL_CACHE_KEY:", err)
        return
    }
    fmt.Printf("REDIS_GLOBAL_CACHE_KEY: %v (%T)\n", redisCacheKey, redisCacheKey)

    ubisoftAccounts, err := env.GetValue("UBISOFT_ACCOUNTS", "list", []interface{}{})
    if err != nil {
        fmt.Println("Error getting UBISOFT_ACCOUNTS:", err)
        return
    }
    fmt.Printf("UBISOFT_ACCOUNTS: %v (%T)\n", ubisoftAccounts, ubisoftAccounts)

    databases, err := env.GetValue("DATABASES", "list", []interface{}{})
    if err != nil {
        fmt.Println("Error getting DATABASES:", err)
        return
    }
    fmt.Printf("DATABASES: %v (%T)\n", databases, databases)

    data, err := env.GetValue("DATA", "dict", map[string]interface{}{})
    if err != nil {
        fmt.Println("Error getting DATA:", err)
        return
    }
    fmt.Printf("DATA: %v (%T)\n", data, data)

    listOfUsers, err := env.GetValue("LIST_OF_USERS", "list", []interface{}{})
    if err != nil {
        fmt.Println("Error getting LIST_OF_USERS:", err)
        return
    }
    fmt.Printf("LIST_OF_USERS: %v (%T)\n", listOfUsers, listOfUsers)

    // Get all variables
    fmt.Println("\nAll Variables:")
    vars := env.GetVars()
    for key, value := range vars {
        fmt.Printf("%s: %v (%T)\n", key, value, value)
    }
}
```

### Expected Output

Running the above program with the provided `.env` file will produce output similar to:

```
APP_NAME: Just an app (string)
IS_DEV_MODE: true (bool)
REDIS_HOST: hostname (string)
REDIS_PORT: 6379 (int)
REDIS_PASS: my-pass (string)
REDIS_GLOBAL_CACHE_KEY: ubi-go (string)
UBISOFT_ACCOUNTS: [map[email:someone@gmail.com password:this] map[email:someone-else password:another-one]] ([]interface {})
DATABASES: [alex someone else 12] ([]interface {})
DATA: map[first-name:alex testing-this:12] (map[string]interface {})
LIST_OF_USERS: [michael joe stan 123 true false] ([]interface {})

All Variables:
APP_NAME: Just an app (string)
IS_DEV_MODE: true (bool)
REDIS_HOST: hostname (string)
REDIS_PORT: 6379 (int)
REDIS_PASS: my-pass (string)
REDIS_GLOBAL_CACHE_KEY: ubi-go (string)
UBISOFT_ACCOUNTS: [map[email:someone@gmail.com password:this] map[email:someone-else password:another-one]] ([]interface {})
DATABASES: [alex someone else 12] ([]interface {})
DATA: map[first-name:alex testing-this:12] (map[string]interface {})
LIST_OF_USERS: [michael joe stan 123 true false] ([]interface {})
```

## Configuration Options

EnvParser provides functional options for customization:

- `WithFilename(name string)`: Set a custom filename (e.g., `.env.local`).
- `WithRootPath(use bool)`: Enable or disable project root detection.
- `WithExtraFiles(files []string)`: Specify additional `.env` files to parse.
- `WithDebug(debug bool)`: Enable debug mode for detailed logging.

Example:

```go
env, err := envparser.New(
    envparser.WithFilename(".env.local"),
    envparser.WithRootPath(false),
    envparser.WithExtraFiles([]string{".env.defaults", ".env.secrets"}),
    envparser.WithDebug(true),
)
```

## Variable Substitution

EnvParser supports variable substitution in `.env` files. For example:

```
# .env
BASE_URL=http://localhost
API_URL=${BASE_URL}/api
```

`API_URL` will resolve to `http://localhost/api`.

## Encrypted Values

To handle encrypted values (e.g., `ENCRYPTED_SECRET=ENC(base64encodedvalue)`), use:

```go
secret, err := env.GetEncryptedValue("ENCRYPTED_SECRET", "string", "", "your-16-byte-key")
if err != nil {
    fmt.Println("Error:", err)
    return
}
fmt.Printf("Decrypted Secret: %v\n", secret)
```

*Note*: The encryption key must be 16, 24, or 32 bytes for AES encryption.

## File Parsing Rules

- Lines starting with `#` are treated as comments and ignored.
- Empty lines are skipped.
- Key-value pairs are split on the first `=` character.
- Values enclosed in quotes are unquoted, and escaped quotes are handled properly.
- If a key already exists, the first occurrence takes precedence.
- The parser searches for the project root (if enabled) by looking for markers like `go.mod`, `.git`, `.project-root`, or `.root`.

## Limitations

- Does not support writing to `.env` files.
- Encryption requires a valid AES key for non-base64-only encrypted values.
- JSON parsing assumes valid JSON for `dict` or `list` types; malformed JSON results in errors.
- File I/O errors (e.g., file not found) are captured in `EnvError`.

## Contributing

Contributions are welcome! Please submit a pull request or open an issue on the GitHub repository.

## License

This project is licensed under the MIT License.