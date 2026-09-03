package theboringoffice_test

import (
	"os"
	"regexp"
	"testing"
)

func TestGetChecksumRegex(t *testing.T) {
	script, err := os.ReadFile("install.ps1")
	if err != nil {
		t.Fatal(err)
	}

	getChecksum := regexp.MustCompile(`(?ms)^function Get-Checksum\([^\n]*\{(.*?)^\}`)
	match := getChecksum.FindStringSubmatch(string(script))
	if len(match) != 2 {
		t.Fatal("Get-Checksum function not found")
	}
	matchExpression := regexp.MustCompile(`(?m)^\s*\$_ -match \(.*\)$`).FindString(match[1])
	if matchExpression == "" {
		t.Fatal("Get-Checksum match expression not found")
	}
	if regexp.MustCompile(`(?:^|\s)-f(?:\s|$)`).MatchString(matchExpression) {
		t.Fatalf("Get-Checksum match expression must not use -f: %s", matchExpression)
	}

	asset := "theboringoffice_0.3.24_windows_amd64.zip"
	checksum := regexp.MustCompile(`^[a-fA-F0-9]{64}\s+\*?` + regexp.QuoteMeta(asset) + `$`)
	hash := "c366c70ea763396f1794230b39ac8a70fcfa6a9e948671262ea39558673a2443"
	tests := []struct {
		name  string
		line  string
		match bool
	}{
		{"GoReleaser checksum", hash + "  " + asset, true},
		{"GNU checksum", hash + " *" + asset, true},
		{"different filename", hash + "  different.zip", false},
		{"63-character hash", hash[:63] + "  " + asset, false},
		{"65-character hash", hash + "0  " + asset, false},
	}
	for _, tc := range tests {
		if got := checksum.MatchString(tc.line); got != tc.match {
			t.Errorf("%s: MatchString(%q) = %t, want %t", tc.name, tc.line, got, tc.match)
		}
	}
}
