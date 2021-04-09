package main

import (
	"encoding/json"
	"strings"
)

type Spell struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	SpellData   map[string]interface{} `json:"spelldata,omitempty"`
	Metadata    SpellMetadata          `json:"metadata,omitempty"`
}

func (s *Spell) UnmarshalJSON(data []byte) error {
	var temp struct {
		Name        string                 `json:"name"`
		Description string                 `json:"description"`
		SpellData   map[string]interface{} `json:"spelldata,omitempty"`
		Metadata    SpellMetadata          `json:"metadata,omitempty"`
	}
	err := json.Unmarshal(data, &temp)
	if err != nil {
		return err
	}

	temp.Name = strings.ToLower(temp.Name)
	*s = temp
	return nil
}

func (s Spell) MarshalJSON() ([]byte, error) {

	var temp struct {
		Name        string                 `json:"name"`
		Description string                 `json:"description"`
		SpellData   map[string]interface{} `json:"spelldata,omitempty"`
		Metadata    SpellMetadata          `json:"metadata,omitempty"`
	}

	temp.Name = strings.Title(s.Name)
	temp.Description = s.Description
	temp.SpellData = s.SpellData
	temp.Metadata = s.Metadata

	return json.Marshal(temp)
}

type SpellMetadata struct {
	System  string `json:"system"`
	Creator string `json:"creator,omitempty"`
}

func (smd SpellMetadata) MarshalJSON() ([]byte, error) {

	var temp struct {
		System string
	}

	temp.System = smd.System

	return json.Marshal(temp)
}
