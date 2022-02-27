package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/chrislgardner/spellapi/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"gopkg.in/launchdarkly/go-sdk-common.v2/lduser"
)

const (
	MultipleMatchingSpells = "multiple matching spells found"
	SpellAlreadyExists     = "spell already exists for this system"
)

type FeatureFlags interface {
	GetUser(ctx context.Context, r *http.Request) lduser.User
	GetIntFlag(ctx context.Context, flag string, user lduser.User) int
	GetBoolFlag(ctx context.Context, flag string, user lduser.User) bool
}

type SpellService struct {
	store db.DB
	flags FeatureFlags
}

type Spell struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	SpellData   map[string]interface{} `json:"spelldata,omitempty"`
	Metadata    SpellMetadata          `json:"metadata,omitempty"`
}

type Request struct {
	Data []map[string]interface{} `json:"data"`
}

type Response struct {
	Count           int             `json:"count"`
	TraceId         string          `json:"traceid,omitempty"`
	ResponseCode    int             `json:"responsecode"`
	ResponseMessage string          `json:"responsemessage"`
	Data            []ErrorResponse `json:"data"`
}

type ErrorResponse struct {
	Message      string `json:"message"`
	ResponseCode int    `json:"responsecode"`
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

func (s Spell) String() string {
	json, _ := json.Marshal(s)

	return string(json)
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

func (s SpellMetadata) String() string {
	json, _ := json.Marshal(s)

	return string(json)
}

func FindSpell(ctx context.Context, db db.DB, name string, query url.Values) (Spell, error) {
	tracer := otel.Tracer("Encantus")
	ctx, span := tracer.Start(ctx, "FindSpell")
	defer span.End()

	span.SetAttributes(
		attribute.String("FindSpell.Spellname", name),
		attribute.String("FindSpell.RawQuery", query.Encode()),
	)

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
				"$in": v,
			}
		}
	}

	span.SetAttributes(attribute.String("FindSpell.BsonQuery", fmt.Sprintf("%v", bsonQuery)))

	results, err := db.GetSpell(ctx, bsonQuery)
	if err != nil {
		span.SetAttributes(attribute.String("FindSpell.Error", err.Error()))
		return Spell{}, fmt.Errorf("query failed on DB: %v", err)
	}

	span.SetAttributes(
		attribute.Int("FindSpell.ResultsCount", len(results)),
		attribute.String("FindSpell.Results", fmt.Sprintf("%v", results)),
	)

	if len(results) == 0 {
		return Spell{}, nil
	} else if len(results) > 1 {
		return Spell{}, fmt.Errorf(MultipleMatchingSpells)
	}

	var s Spell
	temp, err := bson.Marshal(results[0])
	if err != nil {
		span.SetAttributes(attribute.String("FindSpell.Error", err.Error()))
		return Spell{}, fmt.Errorf("failed to marshall data: %v", err)
	}

	err = bson.Unmarshal(temp, &s)
	if err != nil {
		span.SetAttributes(attribute.String("FindSpell.error", err.Error()))
		return Spell{}, fmt.Errorf("failed to unmarshall data: %v", err)
	}

	return s, nil
}

func AddSpell(ctx context.Context, db db.DB, spell Spell) error {
	tracer := otel.Tracer("Encantus")
	ctx, span := tracer.Start(ctx, "AddSpell")
	defer span.End()

	span.SetAttributes(attribute.Stringer("AddSpell.Spell", spell))

	queryValues := url.Values{"system": []string{spell.Metadata.System}}
	exists, err := FindSpell(ctx, db, spell.Name, queryValues)
	if err != nil {
		span.SetAttributes(attribute.String("AddSpell.Error", err.Error()))
		return fmt.Errorf("failed to check for existing spells: %v", err)
	}

	span.SetAttributes(attribute.Stringer("AddSpell.Existing", exists))

	if exists.Name == spell.Name {
		span.SetAttributes(attribute.String("AddSpell.Error", SpellAlreadyExists))
		return fmt.Errorf(SpellAlreadyExists)
	}

	bsonSpell, err := bson.Marshal(spell)
	if err != nil {
		span.SetAttributes(attribute.String("AddSpell.Error", err.Error()))
		return fmt.Errorf("failed to marshall data: %v", err)
	}

	err = db.AddSpell(ctx, bsonSpell)
	if err != nil {
		span.SetAttributes(attribute.String("AddSpell.Error", err.Error()))
		return fmt.Errorf("failed to add spell to DB: %v", err)
	}

	return nil
}

func ParseSpell(ctx context.Context, in []byte) (Spell, error) {

	tracer := otel.Tracer("Encantus")
	ctx, span := tracer.Start(ctx, "ParseSpell")
	defer span.End()

	var s Spell
	err := json.Unmarshal(in, &s)
	if err != nil {
		span.SetAttributes(attribute.String("ParseSpell.Error", err.Error()))
		return Spell{}, err
	}

	if s.Name == "" {
		span.SetAttributes(attribute.String("PostSpellHandler.MissingField", "Name"))
		return s, fmt.Errorf("missing required field: name")
	} else if s.Description == "" {
		span.SetAttributes(attribute.String("PostSpellHandler.MissingField", "Description"))
		return s, fmt.Errorf("missing required field: description")
	} else if s.Metadata.System == "" {
		span.SetAttributes(attribute.String("PostSpellHandler.MissingField", "System"))
		return s, fmt.Errorf("missing required field: system")
	}

	return s, nil
}

func DeleteSpell(ctx context.Context, db db.DB, spell string, query url.Values) error {
	tracer := otel.Tracer("Encantus")
	ctx, span := tracer.Start(ctx, "DeleteSpell")
	defer span.End()

	span.SetAttributes(attribute.String("DeleteSpell.SpellName", spell))

	exists, err := FindSpell(ctx, db, spell, query)
	if err != nil {
		span.SetAttributes(attribute.String("DeleteSpell.Error", err.Error()))
		return fmt.Errorf("failed to check for existing spells: %v", err)
	}

	span.SetAttributes(attribute.Stringer("DeleteSpell.Existing", exists))

	bsonQuery := bson.M{
		"name": bson.M{
			"$eq": exists.Name,
		},
		"metadata.system": bson.M{
			"$eq": exists.Metadata.System,
		},
	}

	err = db.DeleteSpell(ctx, bsonQuery)
	if err != nil {
		span.SetAttributes(attribute.String("DeleteSpell.Error", err.Error()))
		return fmt.Errorf("failed to delete spell from DB: %v", err)
	}

	return nil
}

func GetAllSpell(ctx context.Context, db db.DB, query url.Values) ([]Spell, error) {
	tracer := otel.Tracer("Encantus")
	ctx, span := tracer.Start(ctx, "GetAllSpell")
	defer span.End()

	span.SetAttributes(attribute.String("GetAllSpell.RawQuery", query.Encode()))

	bsonQuery := bson.M{}

	for k, v := range query {
		if k == "system" {
			bsonQuery["metadata.system"] = bson.M{
				"$eq": v[0],
			}
		} else {
			bsonQuery[(fmt.Sprintf("spelldata.%s", k))] = bson.M{
				"$in": v,
			}
		}
	}

	span.SetAttributes(attribute.String("GetAllSpell.BsonQuery", fmt.Sprintf("%v", bsonQuery)))

	results, err := db.GetSpell(ctx, bsonQuery)
	if err != nil {
		span.SetAttributes(attribute.String("GetAllSpell.Error", err.Error()))
		return []Spell{}, fmt.Errorf("query failed on DB: %v", err)
	}

	span.SetAttributes(attribute.Int("GetAllSpell.ResultsCount", len(results)))
	span.SetAttributes(attribute.String("GetAllSpell.Results", fmt.Sprintf("%v", results)))

	if len(results) == 0 {
		return []Spell{}, nil
	}

	var s []Spell

	for _, v := range results {
		temp, err := bson.Marshal(v)
		if err != nil {
			span.SetAttributes(attribute.String("GetAllSpell.Error", err.Error()))
			return []Spell{}, fmt.Errorf("failed to marshall data: %v", err)
		}

		var tempSpell Spell
		err = bson.Unmarshal(temp, &tempSpell)
		if err != nil {
			span.SetAttributes(attribute.String("GetAllSpell.error", err.Error()))
			return []Spell{}, fmt.Errorf("failed to unmarshall data: %v", err)
		}
		s = append(s, tempSpell)
	}

	return s, nil
}

func GetSpellMetadata(ctx context.Context, db db.DB, metadataName string) ([]string, error) {

	tracer := otel.Tracer("Encantus")
	ctx, span := tracer.Start(ctx, "GetSpellMetadata")
	defer span.End()

	span.SetAttributes(attribute.String("GetSpellMetadata.MetadataName", metadataName))
	if metadataName == "system" {
		metadataName = "metadata.system"
	} else {
		metadataName = fmt.Sprintf("spelldata.%s", metadataName)
	}

	results, err := db.GetMetadataValues(ctx, metadataName)
	if err != nil {
		span.SetAttributes(attribute.String("GetSpellMetadata.error", err.Error()))
		return nil, fmt.Errorf("failed to get metadata: %v", err)
	}

	span.SetAttributes(attribute.String("GetSpellMetadata.Results", fmt.Sprintf("%v", results)))

	return results, nil
}

func GetAllSpellMetadata(ctx context.Context, db db.DB, query url.Values) ([]string, error) {
	tracer := otel.Tracer("Encantus")
	ctx, span := tracer.Start(ctx, "GetAllSpellMetadata")
	defer span.End()

	span.SetAttributes(attribute.String("GetAllSpellMetadata.RawQuery", query.Encode()))

	res, err := db.GetMetadataNames(ctx)
	if err != nil {
		span.SetAttributes(attribute.String("GetAllSpellMetadata.error", err.Error()))
		return nil, fmt.Errorf("failed to get metadata: %v", err)
	}

	return res, nil
}
