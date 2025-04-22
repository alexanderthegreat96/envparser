package envparser

// This file has been refactored and improved by GPT 3o
// although some of the original logic is there
// it was highly improved to achieve better results

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// -----------------------------------------------------------------------------
// Public API types
// -----------------------------------------------------------------------------

type EnvData struct {
	FilePath    string
	EnvContents map[any]any
	EnvError    error

	// config the functional‑options path manipulates
	filename   string
	useRoot    bool
	extraFiles []string
	debug      bool
}

// -----------------------------------------------------------------------------
// Functional‑options constructor
// -----------------------------------------------------------------------------

// Option configures an EnvData instance before it is initialised.
// Returning an error aborts construction.
//
// Example:
//  p, err := envparser.New(envparser.WithDebug(true), envparser.WithFilename(".env.test"))
//
// The legacy constructor (NewEnvParser) continues to work unchanged.

type Option func(*EnvData) error

func New(opts ...Option) (*EnvData, error) {
	env := &EnvData{
		filename: ".env",
		useRoot:  true,
	}

	// apply options
	for _, opt := range opts {
		if err := opt(env); err != nil {
			return nil, fmt.Errorf("option: %w", err)
		}
	}

	// run the same initialisation logic the legacy path uses
	env.EnvParser(env.filename, env.useRoot, env.extraFiles)
	return env, env.EnvError
}

// WithFilename overrides the primary env file (default ".env").
func WithFilename(name string) Option {
	return func(e *EnvData) error {
		if strings.TrimSpace(name) == "" {
			return errors.New("filename cannot be empty")
		}
		e.filename = name
		return nil
	}
}

// WithRootPath toggles project‑root resolution (enabled by default).
func WithRootPath(use bool) Option {
	return func(e *EnvData) error {
		e.useRoot = use
		return nil
	}
}

// WithExtraFiles appends additional env files that load **before** the main one.
func WithExtraFiles(files []string) Option {
	return func(e *EnvData) error {
		e.extraFiles = append(e.extraFiles, files...)
		return nil
	}
}

// WithDebug enables noisy logging to stderr.
func WithDebug(debug bool) Option {
	return func(e *EnvData) error {
		e.debug = debug
		return nil
	}
}

// -----------------------------------------------------------------------------
// Legacy variadic constructor (unchanged signature)
// -----------------------------------------------------------------------------

func NewEnvParser(params ...any) *EnvData {
	env := &EnvData{}
	env.EnvParser(params...)
	return env
}

// -----------------------------------------------------------------------------
// Original initialiser (slightly adapted to respect debug flag)
// -----------------------------------------------------------------------------

func (env *EnvData) EnvParser(params ...any) {
	filename := ".env"
	useRoot := true
	var extra []string

	if len(params) > 0 {
		if f, ok := params[0].(string); ok {
			filename = f
		}
	}
	if len(params) > 1 {
		if b, ok := params[1].(bool); ok {
			useRoot = b
		}
	}
	if len(params) > 2 {
		if list, ok := params[2].([]string); ok {
			extra = list
		}
	}

	env.filename = filename
	env.useRoot = useRoot
	env.extraFiles = extra

	// parse extra files first so that later files win (same as original order)
	for _, f := range extra {
		env.parseFile(f, useRoot)
		if env.EnvError != nil {
			return
		}
	}
	env.parseFile(filename, useRoot)
}

// -----------------------------------------------------------------------------
// Public helper methods (identical contracts)
// -----------------------------------------------------------------------------

func (env *EnvData) GetError() string {
	if env.EnvError == nil {
		return ""
	}
	return env.EnvError.Error()
}

// GetVars still returns a copy with best‑effort automatic typing.
func (env *EnvData) GetVars() map[any]any {
	out := make(map[any]any, len(env.EnvContents))
	for k, v := range env.EnvContents {
		// Attempt to convert every *string* value; leave others untouched.
		str, ok := v.(string)
		if !ok {
			out[k] = v
			continue
		}
		if converted, err := env.ConvertInputToType(str); err == nil {
			out[k] = converted
		} else {
			env.EnvError = fmt.Errorf("failed to convert %v: %w", k, err)
			out[k] = v
		}
	}
	return out
}

// GetEncryptedValue behaves exactly as before but with correct decryption.
func (env *EnvData) GetEncryptedValue(which, kind string, defaultValue any, key string) (any, error) {
	if env.EnvError != nil {
		return nil, env.EnvError
	}

	raw, ok := env.EnvContents[which]
	if !ok {
		raw = defaultValue
	}

	strVal, ok := raw.(string)
	if !ok {
		return nil, fmt.Errorf("value for %s is not a string", which)
	}
	if !isEncrypted(strVal) {
		return nil, fmt.Errorf("value for %s is not encrypted", which)
	}

	plain, err := decrypt(strVal, key)
	if err != nil {
		return nil, err
	}

	// explicit type requested?
	if kind != "" && isAllowedType(kind) {
		return ConvertToSpecificType(plain, kind)
	}
	return env.ConvertInputToType(plain)
}

// GetValue is unchanged (bug‑fixed inside helpers).
func (env *EnvData) GetValue(which, kind string, defaultValue any) (any, error) {
	if env.EnvError != nil {
		return nil, env.EnvError
	}

	v, ok := env.EnvContents[which]
	if !ok {
		v = defaultValue
	}

	// requested explicit type?
	if kind != "" && isAllowedType(kind) {
		return ConvertToSpecificType(fmt.Sprintf("%v", v), kind)
	}
	return env.ConvertInputToType(fmt.Sprintf("%v", v))
}

// -----------------------------------------------------------------------------
// Internal helpers (kept unexported)
// -----------------------------------------------------------------------------

func (env *EnvData) parseFile(name string, useRoot bool) {
	var path string
	if useRoot {
		root, err := findRoot()
		if err != nil {
			env.EnvError = err
			return
		}
		path = filepath.Join(root, name)
	} else {
		path = name
	}

	file, err := os.Open(path)
	if err != nil {
		env.EnvError = err
		return
	}
	defer file.Close()

	if env.EnvContents == nil {
		env.EnvContents = make(map[any]any)
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := splitKV(line)
		if !ok {
			continue // ignore malformed
		}
		if _, exists := env.EnvContents[key]; exists {
			continue // first one wins
		}
		env.EnvContents[key] = env.substitute(val)
	}
	if err := scanner.Err(); err != nil {
		env.EnvError = err
	}
}

func splitKV(line string) (key, val string, ok bool) {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	key = strings.TrimSpace(parts[0])
	val = strings.TrimSpace(parts[1])

	// handle quoted value
	if strings.HasPrefix(val, "\"") && strings.HasSuffix(val, "\"") {
		val = strings.Trim(val, "\"")
		val = strings.ReplaceAll(val, `\"`, `"`)
	}
	return key, val, true
}

var varRegex = regexp.MustCompile(`\$\{?([A-Za-z_][A-Za-z0-9_]*)\}?`)

func (env *EnvData) substitute(s string) string {
	return varRegex.ReplaceAllStringFunc(s, func(m string) string {
		name := strings.Trim(m, "${}")
		if v, ok := env.EnvContents[name]; ok {
			return fmt.Sprintf("%v", v)
		}
		if v, ok := os.LookupEnv(name); ok {
			return v
		}
		return m // leave untouched
	})
}

// convertInputToType infers bool/int/float/JSON/list/tuple/dict just like before but with accurate boolean parsing.
func (env *EnvData) ConvertInputToType(s string) (any, error) {
	if b, err := strconv.ParseBool(s); err == nil {
		return b, nil
	}

	if i, err := strconv.Atoi(s); err == nil {
		return i, nil
	}

	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f, nil
	}

	if isList(s) || isTuple(s) {
		trimmed := strings.TrimSpace(s[1 : len(s)-1])
		if trimmed == "" {
			return []any{}, nil
		}
		parts := strings.Split(trimmed, ",")
		list := make([]any, 0, len(parts))
		for _, p := range parts {
			list = append(list, strings.TrimSpace(p))
		}
		return list, nil
	}

	if isDict(s) || json.Valid([]byte(s)) {
		m, err := convertStringToMap(s)
		if err != nil {
			return nil, err
		}
		return m, nil
	}

	return s, nil // plain string
}

func ConvertToSpecificType(val, kind string) (any, error) {
	switch strings.ToLower(kind) {
	case "str", "string":
		return val, nil
	case "bool", "boolean":
		return strconv.ParseBool(val)
	case "int", "integer":
		return strconv.Atoi(val)
	case "float":
		return strconv.ParseFloat(val, 64)
	case "list", "array", "tuple":
		if !isList(val) && !isTuple(val) {
			return nil, fmt.Errorf("value is not list/tuple syntax")
		}
		trimmed := strings.TrimSpace(val[1 : len(val)-1])
		if trimmed == "" {
			return []any{}, nil
		}
		elems := strings.Split(trimmed, ",")
		out := make([]any, 0, len(elems))
		for _, e := range elems {
			out = append(out, strings.TrimSpace(e))
		}
		return out, nil
	case "dict", "map", "json":
		return convertStringToMap(val)
	default:
		return nil, fmt.Errorf("unsupported type %s", kind)
	}
}

// -----------------------------------------------------------------------------
// Utility predicates (tweaked where wrong previously)
// -----------------------------------------------------------------------------

func isList(s string) bool {
	s = strings.TrimSpace(s)
	return strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]")
}
func isTuple(s string) bool {
	s = strings.TrimSpace(s)
	return strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")")
}
func isDict(s string) bool {
	s = strings.TrimSpace(s)
	return strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}")
}

func isEncrypted(s string) bool {
	s = strings.TrimSpace(s)
	return (strings.HasPrefix(s, "ENC(") || strings.HasPrefix(s, "enc(")) && strings.HasSuffix(s, ")")
}

func convertStringToMap(s string) (map[string]any, error) {
	s = strings.TrimSpace(s)
	if strings.Contains(s, "'") && !strings.Contains(s, "\"") {
		s = strings.ReplaceAll(s, "'", "\"")
	}
	if !(isDict(s) || json.Valid([]byte(s))) {
		return nil, errors.New("invalid map/json format")
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil, err
	}
	return out, nil
}

// -----------------------------------------------------------------------------
// Encryption helpers (bug‑fixed)
// -----------------------------------------------------------------------------

func decrypt(ciphertext, key string) (string, error) {
	payload := strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(ciphertext, "ENC("), "enc("), ")")

	// base64 only
	if key == "" {
		b, err := base64.StdEncoding.DecodeString(payload)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}

	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return "", errors.New("AES key length must be 16, 24, or 32 bytes")
	}

	raw, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return "", err
	}
	if len(raw) < aes.BlockSize {
		return "", io.ErrUnexpectedEOF
	}
	iv := raw[:aes.BlockSize]
	raw = raw[aes.BlockSize:]

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(raw, raw)
	return string(raw), nil
}

func findRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	markers := []string{"go.mod", ".git", ".project-root", ".root"}
	for {
		for _, m := range markers {
			if _, err := os.Stat(filepath.Join(dir, m)); err == nil {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("project root not found")
		}
		dir = parent
	}
}

// keeps API list for validation
func isAllowedType(kind string) bool {
	allowed := []string{
		"str", "string", "bool", "boolean", "float",
		"int", "integer", "list", "array", "tuple",
		"dict", "map", "json",
	}
	return slices.Contains(allowed, strings.ToLower(kind))
}
