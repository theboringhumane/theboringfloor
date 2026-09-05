package config

import "testing"

func legacyTestEnv(suffix string) string {
	return "THE" + "BORINGOFFICE_" + suffix
}

func TestEnv(t *testing.T) {
	for _, tc := range []struct {
		name      string
		canonical string
		legacy    string
		want      string
	}{
		{name: "canonical only", canonical: "canonical", want: "canonical"},
		{name: "legacy only", legacy: "legacy", want: "legacy"},
		{name: "both set", canonical: "canonical", legacy: "legacy", want: "canonical"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("THEFLOOR_TEST_ENV", tc.canonical)
			t.Setenv(legacyTestEnv("TEST_ENV"), tc.legacy)
			if got := Env("TEST_ENV"); got != tc.want {
				t.Errorf("Env() = %q, want %q", got, tc.want)
			}
		})
	}

	t.Run("neither set", func(t *testing.T) {
		if got := Env("TEST_ENV_NEITHER_SET"); got != "" {
			t.Errorf("Env() = %q, want empty", got)
		}
	})
}

func TestEnvBool(t *testing.T) {
	for _, value := range []string{"1", "0", "", "true"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("THEFLOOR_TEST_BOOL", value)
			if got, want := EnvBool("TEST_BOOL"), value == "1"; got != want {
				t.Errorf("EnvBool() = %v, want %v", got, want)
			}
		})
	}
}

func TestLookupEnv(t *testing.T) {
	t.Run("unset", func(t *testing.T) {
		got, ok := LookupEnv("TEST_LOOKUP_NEITHER_SET")
		if got != "" || ok {
			t.Errorf("LookupEnv() = (%q, %v), want (\"\", false)", got, ok)
		}
	})

	t.Run("canonical empty wins", func(t *testing.T) {
		t.Setenv("THEFLOOR_TEST_LOOKUP", "")
		t.Setenv(legacyTestEnv("TEST_LOOKUP"), "legacy")
		got, ok := LookupEnv("TEST_LOOKUP")
		if got != "" || !ok {
			t.Errorf("LookupEnv() = (%q, %v), want (\"\", true)", got, ok)
		}
	})
}
