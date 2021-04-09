package main_test

import (
	"encoding/json"
	"strings"
	"testing"

	spellapi "github.com/chrislgardner/spellapi"
)

var spellJson = []byte(`{
	"name": "Fireball",
	"description": "Does the Big Boom"	
}`)

func TestSpell_Unmarshal(t *testing.T) {
	var got spellapi.Spell
	err := json.Unmarshal(spellJson, &got)
	if err != nil {
		t.Fatalf("Unmarshal() err = %v; want nil", err)
	}

	wantName := "fireball"
	if got.Name != wantName {
		t.Errorf("Spell Name got %v; want %v", got.Name, wantName)
	}

	wantDescription := "Does the Big Boom"
	if got.Description != wantDescription {
		t.Errorf("Spell Description got %v; want %v", got.Description, wantDescription)
	}
}

func TestSpell_Marshal(t *testing.T) {
	want := spellapi.Spell{
		Name:        "fireball",
		Description: "Does the Big Boom",
	}

	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("Marshal() err = %v; want nil", err)
	}

	if !strings.Contains(string(data), "Fireball") {
		t.Errorf("Marshal() returned name %v; Want Fireball", string(data))
	}

	var got spellapi.Spell
	err = json.Unmarshal(data, &got)
	if err != nil {
		t.Fatalf("Marshal() err = %v; want nil", err)
	}

	if got.Description != want.Description {
		t.Errorf("Marshal() returned description %v; Want %v", got.Description, want.Description)
	}
}

func TestSpellMetadata_Marshal(t *testing.T) {
	want := spellapi.SpellMetadata{
		Creator: "TestUser",
		System:  "Example",
	}

	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("Marshal() err = %v; want nil", err)
	}

	var got spellapi.SpellMetadata
	err = json.Unmarshal(data, &got)
	if err != nil {
		t.Fatalf("Marshal() err = %v; want nil", err)
	}

	if got.Creator != "" {
		t.Errorf("Marshal() returned creator %v; Want nil", got.Creator)
	}

	if got.System != want.System {
		t.Errorf("Marshal() returned system %v; Want %v", got.System, want.System)
	}

}
