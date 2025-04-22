package envparser

import (
	"os"
	"reflect"
	"testing"
)

// -----------------------------------------------------------------------------
// Constructors
// -----------------------------------------------------------------------------

func TestLegacyConstructorInitialisesMap(t *testing.T) {
	env := NewEnvParser()
	if env.EnvContents == nil {
		t.Error("expected EnvContents to be initialised")
	}
}

func TestFunctionalOptionsConstructor(t *testing.T) {
	env, err := New(
		WithFilename(".env"), // file may not exist, we only test construction
		WithRootPath(false),
		WithDebug(true),
	)
	if err != nil && !os.IsNotExist(err) { // ignore ENOENT edge case
		t.Fatalf("unexpected error: %v", err)
	}
	if env.EnvContents == nil {
		t.Error("expected EnvContents to be initialised")
	}
	if !env.debug {
		t.Error("expected debug flag to be true")
	}
}

// -----------------------------------------------------------------------------
// File parsing
// -----------------------------------------------------------------------------

func TestParseValidFile(t *testing.T) {
	testFile := ".env.test"
	content := "TEST_VAR=value\nANOTHER_VAR=another_value"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("cannot create test file: %v", err)
	}
	t.Cleanup(func() { os.Remove(testFile) })

	env := NewEnvParser(testFile, false) // useRootPath=false so cwd is used
	if env.EnvError != nil {
		t.Fatalf("unexpected error: %v", env.EnvError)
	}
	if got := env.EnvContents["TEST_VAR"]; got != "value" {
		t.Errorf("TEST_VAR: want value, got %v", got)
	}
}

func TestNonExistentFile(t *testing.T) {
	env := NewEnvParser("non_existent.env", false)
	if env.EnvError == nil {
		t.Error("expected error for missing file")
	}
}

// -----------------------------------------------------------------------------
// GetValue helpers
// -----------------------------------------------------------------------------

func TestGetValueExisting(t *testing.T) {
	env := NewEnvParser()
	env.EnvContents = map[interface{}]interface{}{"EXISTING_VAR": "some_value"}

	v, err := env.GetValue("EXISTING_VAR", "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "some_value" {
		t.Errorf("want some_value, got %v", v)
	}
}

func TestGetValueDefault(t *testing.T) {
	env := NewEnvParser()
	env.EnvContents = map[interface{}]interface{}{}

	v, err := env.GetValue("NON_EXISTING_VAR", "", "default_value")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "default_value" {
		t.Errorf("want default_value, got %v", v)
	}
}

func TestGetValueConversionError(t *testing.T) {
	env := NewEnvParser()
	env.EnvContents = map[interface{}]interface{}{"INVALID_VAR": "not_an_int"}

	if _, err := env.GetValue("INVALID_VAR", "int", nil); err == nil {
		t.Error("expected conversion error, got nil")
	}
}

// -----------------------------------------------------------------------------
// Variable substitution
// -----------------------------------------------------------------------------

func TestSubstitute(t *testing.T) {
	env := NewEnvParser()
	env.EnvContents = map[interface{}]interface{}{"TEST_VAR": "substituted_value"}

	got := env.substitute("URL is ${TEST_VAR}/some/path")
	want := "URL is substituted_value/some/path"
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

// -----------------------------------------------------------------------------
// Encryption helpers
// -----------------------------------------------------------------------------

func TestBase64EncryptedValue(t *testing.T) {
	env := NewEnvParser()
	enc := "ENC(YXNkamtuYWtqc2Ric2prYmRma2pzaGRiZg==)" // base64('asdjknakjsdbsjkbdfkjshdbf')

	env.EnvContents = map[interface{}]interface{}{"my_encrypted_var": enc}

	got, err := env.GetEncryptedValue("my_encrypted_var", "", "expected_value", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "asdjknakjsdbsjkbdfkjshdbf"
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

// -----------------------------------------------------------------------------
// Public Conversion helpers
// -----------------------------------------------------------------------------

func TestConvertInputToType(t *testing.T) {
	env := NewEnvParser()

	tests := []struct {
		in   string
		want interface{}
	}{
		{"true", true},
		{"42", 42},
		{"3.5", 3.5},
		{"[a,b]", []interface{}{"a", "b"}},
		{"{\"a\":1}", map[string]interface{}{"a": float64(1)}},
	}

	for _, tc := range tests {
		got, err := env.ConvertInputToType(tc.in)
		if err != nil {
			t.Errorf("ConvertInputToType(%q) returned error: %v", tc.in, err)
			continue
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("ConvertInputToType(%q): want %v (%T), got %v (%T)", tc.in, tc.want, tc.want, got, got)
		}
	}
}

func TestConvertToSpecificType(t *testing.T) {
	tests := []struct {
		in   string
		kind string
		want interface{}
	}{
		{"true", "bool", true},
		{"42", "int", 42},
		{"3.14", "float", 3.14},
		{"[x,y]", "list", []interface{}{"x", "y"}},
		{"{\"k\":\"v\"}", "json", map[string]interface{}{"k": "v"}},
	}

	for _, tc := range tests {
		got, err := ConvertToSpecificType(tc.in, tc.kind)
		if err != nil {
			t.Errorf("ConvertToSpecificType(%q,%s) error: %v", tc.in, tc.kind, err)
			continue
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("ConvertToSpecificType(%q,%s): want %v (%T), got %v (%T)", tc.in, tc.kind, tc.want, tc.want, got, got)
		}
	}
}
