# EnvParser

**EnvParser** is a small, zero‑dependency Go library that loads one or more “dotenv” files, performs \${VAR} substitution, and gives you **strongly‑typed** access to your configuration — even when the values are encrypted.

---

## ✨ What’s new in v2

| Quality‑of‑life | Details |
|-----------------|---------|
| 🔧 **Functional‑options constructor** | `New()` now accepts options such as `WithFilename`, `WithExtraFiles`, and `WithDebug` — keeping the old `NewEnvParser()` signature for backward compatibility. |
| 🪄 **Automatic type inference** | Any value read from a file (or returned via `GetVars`) is converted to `bool`, `int`, `float64`, `[]interface{}`, or `map[string]interface{}` whenever possible. |
| 🔒 **Better encryption support** | Values wrapped in `ENC(...)` can be raw **Base‑64** or **AES‑CFB** (16/24/32‑byte key). Use `GetEncryptedValue` and forget the rest. |
| 🐞 **Bug fixes & safety** | Correct boolean parsing (no more “`false` → `true`” bug), nil‑safe `GetError`, stricter AES key length checks. |
| 🧪 **Extended test‑suite** | Public helpers `ConvertInputToType` and `ConvertToSpecificType` are now covered — feel free to use them anywhere in your project. |

---

## Installation

```bash
# Go 1.22+
go get github.com/alexanderthegreat96/envparser/v2
```

---

## Quick start

```go
package main

import (
    "fmt"
    "github.com/alexanderthegreat96/envparser/v2"
)

func main() {
    // Modern style — functional options
    p, err := envparser.New(
        envparser.WithFilename(".env"),         // default is ".env"
        envparser.WithExtraFiles([]string{      // optional — load earlier, lower priority
            ".env.local",
            ".env.secrets",
        }),
        envparser.WithDebug(true),              // noisy logging to stderr
    )
    if err != nil {
        panic(err)
    }

    // Strongly‑typed helpers
    port, _ := p.GetValue("APP_PORT", "int", 8080)
    debug, _ := p.GetValue("DEBUG", "bool", false)

    // Encrypted value (base64 or AES)
    secret, _ := p.GetEncryptedValue("JWT_SECRET", "string", nil, os.Getenv("DECRYPT_KEY"))

    fmt.Println(port, debug, secret)
}
```

Prefer the classic style?  It still works:

```go
p := envparser.NewEnvParser(".env.dev", false, nil) // filename, useRootPath, extraFiles
```

---

## Public API

### Constructors

| Function | Description |
|----------|-------------|
| `New(opts ...Option) (*EnvData, error)` | Typed constructor using the options pattern. |
| `NewEnvParser(params ...interface{}) *EnvData` | Legacy variadic constructor (filename, useRootPath bool, extraFiles []string). |

#### Functional Options

* `WithFilename(name string)` – override main env file (default `.env`).
* `WithRootPath(use bool)` – enable/disable project‑root discovery.
* `WithExtraFiles(files []string)` – prepend additional files (first one wins on duplicate keys).
* `WithDebug(debug bool)` – emit verbose logs.

### Core methods

| Method | Purpose |
|--------|---------|
| `GetVars() map[interface{}]interface{}` | Return a **copy** of all variables with auto‑converted types. |
| `GetValue(key, kind string, def interface{}) (interface{}, error)` | Fetch & convert a single key. `kind` may be `string`, `int`, `float`, `bool`, `list`, `dict`, … |
| `GetEncryptedValue(key, kind string, def interface{}, decryptKey string) (interface{}, error)` | Like `GetValue` but decrypts `ENC(...)` payloads. Leave `decryptKey` empty for pure Base‑64. |
| `GetError() string` | Retrieve (and inspect) the last error, if any. |

### Public conversion helpers

Need type‑coercion elsewhere in your code? Use the exported helpers :

```go
out,  _ := envparser.ConvertInputToType("4.2")        // → float64 4.2
addr, _ := envparser.ConvertToSpecificType("true", "bool") // → bool true
```

Supported `kind` values: `str`, `string`, `bool`, `boolean`, `float`, `int`, `integer`, `list`, `array`, `tuple`, `dict`, `map`, `json`.

---

## Variable substitution

A value can reference another variable defined in **any** earlier‑loaded file **or** your process environment:

```dotenv
API_HOST=localhost
API_URL=http://${API_HOST}:8080
```

`API_URL` resolves to `http://localhost:8080`.

---

## Error handling

Almost every public call returns an `error`. Prefer checking it, but you can also inspect the last one via `parser.GetError()`.

---

## License

MIT © 2025 AlexanderTheGreat96



## Licence
MIT License

Copyright (c) [2024] [alexanderthegreat96]

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

1. The above copyright notice and this permission notice shall be included in
   all copies or substantial portions of the Software.

2. THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
   IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
   FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
   AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
   LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
   OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
   SOFTWARE.
