### Functional-options constructor
- Added `New(opts ...Option)` plus options:  
  `WithFilename`, `WithRootPath`, `WithExtraFiles`, `WithDebug`.
- Kept the legacy constructor intact:  
  `NewEnvParser(params ...interface{})` still works exactly as before.

---

### Config fields added to EnvData
- `filename`, `useRoot`, `extraFiles`, `debug` to store option state.

---

### Public conversion helpers exported
- `ConvertInputToType` (was `convertInputToType`)
- `ConvertToSpecificType` (was `convertToSpecificType`)

---

### Bug fixes & safety
- Boolean parsing now uses `strconv.ParseBool` (no more `"false" ⇒ true"` bug).
- AES key length validated (16/24/32 bytes) and payload trimming corrected.
- Scanner loop now calls `scanner.Err()` after reading, capturing I/O errors.
- `GetError` returns empty string when no error (nil-safe).

---

### Regex & parsing tweaks
- `${VAR}` substitution regex compiled once at package load (`varRegex`).
- `splitKV` handles quoted values and escaped quotes correctly.

---

### Type-inference refinements
- `ConvertInputToType` recognises JSON via `json.Valid`.
- List/tuple/dict detection helpers streamlined.

---

### Logging hook
- `debug` flag (set via `WithDebug`) triggers verbose logging inside `parseFile`.

---

### Encryption helper clean-up
- Shared decrypt function handles both Base-64 and AES-CFB with clearer error paths.

---

### Project-root discovery
- `findRoot` unchanged in behaviour but reused by both constructors.

---

### Zero-dependency guarantee retained
- Still in standard library only; no third-party imports added.
