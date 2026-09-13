package state

import (
	"encoding/json"
	"testing"
)

func TestModelSelectionRef(t *testing.T) {
	for _, tc := range []struct {
		model ModelInfo
		want  string
	}{
		{ModelInfo{Provider: "ignored", ID: "ignored", Ref: "opus[1m]"}, "opus[1m]"},
		{ModelInfo{Provider: "anthropic", ID: "claude-sonnet-4-5"}, "anthropic/claude-sonnet-4-5"},
		{ModelInfo{ID: "gpt-5.4"}, "gpt-5.4"},
		{ModelInfo{ID: "namespace/model@preview"}, "namespace/model@preview"},
		{ModelInfo{Ref: " opaque token "}, " opaque token "},
		{ModelInfo{}, ""},
	} {
		if got := tc.model.SelectionRef(); got != tc.want {
			t.Errorf("%+v.SelectionRef() = %q, want %q", tc.model, got, tc.want)
		}
	}
}

func TestModelCatalogJSON(t *testing.T) {
	in := ModelInfo{ID: "opus", Ref: "opus[1m]", Name: "Opus", Description: "Extended context", Disabled: true, IsDefault: true}
	if got := roundTrip(t, in); got != in {
		t.Fatalf("model metadata roundtrip = %+v, want %+v", got, in)
	}
	for _, tc := range []struct {
		value any
		want  string
	}{
		{ModelInfo{ID: "sonnet"}, `{"provider":"","id":"sonnet"}`},
		{in, `{"provider":"","id":"opus","name":"Opus","ref":"opus[1m]","description":"Extended context","disabled":true,"isDefault":true}`},
		{ModelTarget{}, `{}`},
		{ModelTarget{Agent: "research"}, `{"agent":"research"}`},
		{ModelAgentInfo{Name: "research", Description: "Read-only research"}, `{"name":"research","description":"Read-only research"}`},
	} {
		data, err := json.Marshal(tc.value)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != tc.want {
			t.Errorf("JSON = %s, want %s", data, tc.want)
		}
	}
}
