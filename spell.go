package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/honeycombio/beeline-go"
	"go.mongodb.org/mongo-driver/bson"
	"gopkg.in/launchdarkly/go-sdk-common.v2/lduser"
)

const (
	MultipleMatchingSpells = "multiple matching spells found"
	SpellAlreadyExists     = "spell already exists for this system"
)

type Store interface {
	GetSpell(ctx context.Context, search bson.M) ([]bson.M, error)
	AddSpell(ctx context.Context, spell []byte) error
}

type FeatureFlags interface {
	GetUser(ctx context.Context, r http.Request) lduser.User
	GetIntFlag(ctx context.Context, flag string, user lduser.User) int
	GetBoolFlag(ctx context.Context, flag string, user lduser.User) bool
}

type SpellService struct {
	store Store
	flags FeatureFlags
}

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
		Name        string                 `json:"name" bson:"name"`
		Description string                 `json:"description" bson:"description"`
		SpellData   map[string]interface{} `json:"spelldata,omitempty" bson:"spelldata,omitempty"`
		Metadata    SpellMetadata          `json:"metadata" bson:"metadata"`
	}

	temp.Name = strings.Title(s.Name)
	temp.Description = s.Description
	temp.SpellData = s.SpellData
	temp.Metadata = s.Metadata

	return json.Marshal(temp)
}

type SpellMetadata struct {
	System  string `json:"system" bson:"system"`
	Creator string `json:"creator,omitempty" bson:"creator,omitempty"`
}

func (smd SpellMetadata) MarshalJSON() ([]byte, error) {

	var temp struct {
		System string `json:"system"`
	}

	temp.System = smd.System

	return json.Marshal(temp)
}

func FindSpell(ctx context.Context, db Store, name string, query url.Values) (Spell, error) {
	ctx, span := beeline.StartSpan(ctx, "FindSpell")
	defer span.Send()

	beeline.AddField(ctx, "FindSpell.Spellname", name)
	beeline.AddField(ctx, "FindSpell.RawQuery", query)

	bsonQuery := bson.M{
		"name": bson.M{
			"$eq": strings.ToLower(name),
		},
	}

	for k, v := range query {
		if k == "system" {
			bsonQuery["metadata.system"] = bson.M{
				"$eq": v[0],
			}
		} else {
			bsonQuery[(fmt.Sprintf("spelldata.%s", k))] = bson.M{
				"$eq": v,
			}
		}
	}

	beeline.AddField(ctx, "FindSpell.BsonQuery", bsonQuery)

	results, err := db.GetSpell(ctx, bsonQuery)
	if err != nil {
		beeline.AddField(ctx, "FindSpell.Error", err)
		return Spell{}, fmt.Errorf("query failed on DB: %v", err)
	}

	beeline.AddField(ctx, "FindSpell.ResultsCount", len(results))
	beeline.AddField(ctx, "FindSpell.Results", results)

	if len(results) == 0 {
		return Spell{}, nil
	} else if len(results) > 1 {
		return Spell{}, fmt.Errorf(MultipleMatchingSpells)
	}

	var s Spell
	temp, err := bson.Marshal(results[0])
	if err != nil {
		beeline.AddField(ctx, "FindSpell.Error", err)
		return Spell{}, fmt.Errorf("failed to marshall data: %v", err)
	}

	err = bson.Unmarshal(temp, &s)
	if err != nil {
		beeline.AddField(ctx, "FindSpell.error", err)
		return Spell{}, fmt.Errorf("failed to unmarshall data: %v", err)
	}

	return s, nil
}

func AddSpell(ctx context.Context, db Store, spell Spell) error {
	ctx, span := beeline.StartSpan(ctx, "AddSpell")
	defer span.Send()

	beeline.AddField(ctx, "AddSpell.Spell", spell)

	queryValues := url.Values{"system": []string{spell.Metadata.System}}
	exists, err := FindSpell(ctx, db, spell.Name, queryValues)
	if err != nil {
		beeline.AddField(ctx, "AddSpell.Error", err)
		return fmt.Errorf("failed to check for existing spells: %v", err)
	}

	beeline.AddField(ctx, "AddSpell.Existing", exists)

	if exists.Name == spell.Name {
		beeline.AddField(ctx, "AddSpell.Error", SpellAlreadyExists)
		return fmt.Errorf(SpellAlreadyExists)
	}

	bsonSpell, err := bson.Marshal(spell)
	if err != nil {
		beeline.AddField(ctx, "AddSpell.Error", err)
		return fmt.Errorf("failed to marshall data: %v", err)
	}

	err = db.AddSpell(ctx, bsonSpell)
	if err != nil {
		beeline.AddField(ctx, "AddSpell.Error", err)
		return fmt.Errorf("failed to add spell to DB: %v", err)
	}

	return nil
}

func ParseSpell(ctx context.Context, in []byte) (Spell, error) {

	ctx, span := beeline.StartSpan(ctx, "ParseSpell")
	defer span.Send()

	var s Spell
	err := json.Unmarshal(in, &s)
	if err != nil {
		beeline.AddField(ctx, "ParseSpell.Error", err)
		return Spell{}, err
	}

	if s.Name == "" {
		beeline.AddField(ctx, "PostSpellHandler.MissingField", "Name")
		return s, fmt.Errorf("missing required field: name")
	} else if s.Description == "" {
		beeline.AddField(ctx, "PostSpellHandler.MissingField", "Description")
		return s, fmt.Errorf("missing required field: description")
	} else if s.Metadata.System == "" {
		beeline.AddField(ctx, "PostSpellHandler.MissingField", "System")
		return s, fmt.Errorf("missing required field: system")
	}

	return s, nil
}
