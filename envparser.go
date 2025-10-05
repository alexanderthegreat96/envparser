package envparser

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

type EnvData struct {
	FilePath    string
	EnvContents map[string]any
	EnvError    error

	filename   string
	useRoot    bool
	extraFiles []string
	debug      bool
}

type Option func(*EnvData) error

func New(opts ...Option) (*EnvData, error) {
	env := &EnvData{
		filename: ".env",
		useRoot:  true,
	}

	for _, opt := range opts {
		if err := opt(env); err != nil {
			return nil, fmt.Errorf("option: %w", err)
		}
	}

	env.EnvParser(env.filename, env.useRoot, env.extraFiles)
	return env, env.EnvError
}

func WithFilename(name string) Option {
	return func(e *EnvData) error {
		if strings.TrimSpace(name) == "" {
			return errors.New("filename cannot be empty")
		}
		e.filename = name
		return nil
	}
}

func WithRootPath(use bool) Option {
	return func(e *EnvData) error {
		e.useRoot = use
		return nil
	}
}

func WithExtraFiles(files []string) Option {
	return func(e *EnvData) error {
		e.extraFiles = append(e.extraFiles, files...)
		return nil
	}
}

func WithDebug(debug bool) Option {
	return func(e *EnvData) error {
		e.debug = debug
		return nil
	}
}

func NewEnvParser(params ...any) *EnvData {
	env := &EnvData{}
	env.EnvParser(params...)
	return env
}

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

	for _, f := range extra {
		env.parseFile(f, useRoot)
		if env.EnvError != nil {
			return
		}
	}
	env.parseFile(filename, useRoot)
}

func (env *EnvData) GetError() string {
	if env.EnvError == nil {
		return ""
	}
	return env.EnvError.Error()
}

func (env *EnvData) GetVars() map[any]any {
	out := make(map[any]any, len(env.EnvContents))
	for k, v := range env.EnvContents {
		str, ok := v.(string)
		if !ok {
			out[k] = v
			continue
		}
		if converted, err := ConvertInputToType(str); err == nil {
			out[k] = converted
		} else {
			env.EnvError = fmt.Errorf("failed to convert %v: %w", k, err)
			out[k] = v
		}
	}
	return out
}

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

	if kind != "" && isAllowedType(kind) {
		return ConvertToSpecificType(plain, kind)
	}
	return ConvertInputToType(plain)
}

func (env *EnvData) GetValue(which, kind string, defaultValue any) (any, error) {
	if env.EnvError != nil {
		return nil, env.EnvError
	}

	v, ok := env.EnvContents[which]
	if !ok {
		v = defaultValue
	}
	if kind != "" && isAllowedType(kind) {
		return ConvertToSpecificType(fmt.Sprintf("%v", v), kind)
	}
	return ConvertInputToType(fmt.Sprintf("%v", v))
}

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
		env.EnvContents = make(map[string]any)
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

	if strings.HasPrefix(val, "\"") && strings.HasSuffix(val, "\"") {
		val = val[1 : len(val)-1]
		val = strings.ReplaceAll(val, `\"`, `"`)
	}
	return key, val, true
}

var varRegex = regexp.MustCompile(`\$\{?([A-Za-z_][A-Za-z0-9_]*)\}?`)

func (env *EnvData) substitute(s string) string {
	return varRegex.ReplaceAllStringFunc(s, func(m string) string {
		name := strings.TrimSuffix(strings.TrimPrefix(m, "${"), "}")
		if v, ok := env.EnvContents[name]; ok {
			return fmt.Sprintf("%v", v)
		}
		if v, ok := os.LookupEnv(name); ok {
			return v
		}
		return m
	})
}

func ConvertInputToType(s string) (any, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}

	if b, err := strconv.ParseBool(s); err == nil {
		return b, nil
	}
	if i, err := strconv.Atoi(s); err == nil {
		return i, nil
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		if float64(int(f)) == f {
			return int(f), nil
		}
		return f, nil
	}

	if isDict(s) || (strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}")) {
		js := strings.ReplaceAll(s, "'", "\"")

		var out any
		err := json.Unmarshal([]byte(js), &out)
		if err == nil {
			return autoConvertNested(out), nil
		}
		return nil, fmt.Errorf("invalid dict/json (%q): %w", s, err)
	}

	if isList(s) || isTuple(s) {
		js := s
		if isTuple(s) {
			js = "[" + s[1:len(s)-1] + "]"
		}
		js = strings.ReplaceAll(js, "'", "\"")

		if json.Valid([]byte(js)) {
			var out any
			if err := json.Unmarshal([]byte(js), &out); err == nil {
				return autoConvertNested(out), nil
			}
		}

		elems := splitTopLevel(js[1 : len(js)-1])
		arr := make([]any, 0, len(elems))
		for _, e := range elems {
			e = strings.TrimSpace(e)
			if e == "" {
				continue
			}
			v, err := ConvertInputToType(e)
			if err != nil {
				arr = append(arr, e)
			} else {
				arr = append(arr, v)
			}
		}
		return arr, nil
	}

	if json.Valid([]byte(s)) {
		var out any
		if err := json.Unmarshal([]byte(s), &out); err == nil {
			return autoConvertNested(out), nil
		}
	}

	return s, nil
}

func splitTopLevel(s string) []string {
	var parts []string
	var buf strings.Builder
	depth := 0
	inQuote := false
	var quoteChar rune
	escaped := false

	for _, r := range s {
		if inQuote {
			buf.WriteRune(r)
			if escaped {
				escaped = false
			} else if r == '\\' {
				escaped = true
			} else if r == quoteChar {
				inQuote = false
			}
			continue
		}

		switch r {
		case '"', '\'':
			inQuote = true
			quoteChar = r
			buf.WriteRune(r)
		case '{', '[':
			depth++
			buf.WriteRune(r)
		case '}', ']':
			if depth > 0 {
				depth--
			}
			buf.WriteRune(r)
		case ',':
			if depth == 0 {
				parts = append(parts, strings.TrimSpace(buf.String()))
				buf.Reset()
			} else {
				buf.WriteRune(r)
			}
		default:
			buf.WriteRune(r)
		}
	}
	if buf.Len() > 0 {
		parts = append(parts, strings.TrimSpace(buf.String()))
	}
	return parts
}

func autoConvertNested(v any) any {
	switch val := v.(type) {
	case []any:
		for i, e := range val {
			val[i] = autoConvertNested(e)
		}
		return val
	case map[string]any:
		for k, e := range val {
			val[k] = autoConvertNested(e)
		}
		return val
	case float64:
		if float64(int(val)) == val {
			return int(val)
		}
		return val
	default:
		return val
	}
}

func ConvertToSpecificType(val, kind string) (any, error) {
	kind = strings.ToLower(kind)
	switch kind {
	case "str", "string":
		return val, nil
	case "bool", "boolean":
		return strconv.ParseBool(val)
	case "int", "integer":
		return strconv.Atoi(val)
	case "float":
		return strconv.ParseFloat(val, 64)
	case "list", "array", "tuple", "dict", "map", "json":
		return ConvertInputToType(val)
	default:
		return nil, fmt.Errorf("unsupported type %s", kind)
	}
}

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

func decrypt(ciphertext, key string) (string, error) {
	payload := strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(ciphertext, "ENC("), "enc("), ")")

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

func isAllowedType(kind string) bool {
	allowed := []string{
		"str", "string", "bool", "boolean", "float",
		"int", "integer", "list", "array", "tuple",
		"dict", "map", "json",
	}
	return slices.Contains(allowed, strings.ToLower(kind))
}
