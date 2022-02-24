package main_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"testing"

	spellapi "github.com/chrislgardner/spellapi"
	"go.mongodb.org/mongo-driver/bson"
)

var spellJson = []byte(`{
	"name": "Fireball",
	"description": "Does the Big Boom"	
}`)

type TestCaseItem struct {
	input    string
	result   string
	hasError bool
}

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

func TestParseSpell(t *testing.T) {

	testCases := []TestCaseItem{
		{
			"{\"name\":\"test1\",\"description\":\"example\",\"metadata\":{\"system\":\"test\"}}",
			"",
			false,
		},
		{
			"{\"description\":\"example\",\"metadata\":{\"system\":\"test\"}}",
			"missing required field: name",
			true,
		},
		{
			"{\"name\":\"test1\",\"metadata\":{\"system\":\"test\"}}",
			"missing required field: description",
			true,
		},
		{
			"{\"name\":\"test1\",\"description\":\"example\"}",
			"missing required field: system",
			true,
		},
	}

	for _, v := range testCases {

		ctx := context.Background()
		_, err := spellapi.ParseSpell(ctx, []byte(v.input))
		if v.hasError && err != nil {
			if err.Error() != v.result {
				t.Errorf("ParseSpell() err %v, want %v", err, v.result)
			}
		}
	}
}

func TestFindSpell(t *testing.T) {
	
	type NoResults struct {}
	func (s *NoResults) GetSpell(ctx context.Context, search bson.M) ([]bson.M, error){
		return []bson.M{}, nil
	}

	type DBError struct {}
	func (s *DBError) GetSpell(ctx context.Context, search bson.M) ([]bson.M, error){
		return nil, fmt.Errorf("error in DB")
	}

	type MultipleMatches struct {}
	func (s *MultipleMatches) GetSpell(ctx context.Context, search bson.M) ([]bson.M, error){
		return []bson.M{}, nil
	}

	type BadBSONMarshal struct {}
	func (s *BadBSONMarshal) GetSpell(ctx context.Context, search bson.M) ([]bson.M, error){
		return []bson.M{}, nil
	}

	type BadBSONUnmarshal struct {}
	func (s *BadBSONUnmarshal) GetSpell(ctx context.Context, search bson.M) ([]bson.M, error){
		return []bson.M{}, nil
	}

	type SuccessfulResult struct {}
	func (s *SuccessfulResult) GetSpell(ctx context.Context, search bson.M) ([]bson.M, error){
		res := []bson.M{
			{name: "fireball", description: "fire damage", metadata: bson.M{system: "test"}}
		}
		return res, nil
	}

	type args struct {
		ctx   context.Context
		db    spellapi.StoreReader
		name  string
		query url.Values
	}
	tests := []struct {
		name    string
		args    args
		want    spellapi.Spell
		wantErr bool
	}{
		{
			name: "Valid result",
			args: {
				ctx: context.Background(),
				db: SuccessfulResult{}
				name: fireball,
				query: url.Values{},
			},
			want: spellapi.Spell{Name: "fireball", Description: "fire damage", Metadata: spellapi.SpellMetadata{System: "test"}},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := spellapi.FindSpell(tt.args.ctx, tt.args.db, tt.args.name, tt.args.query)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindSpell() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FindSpell() = %v, want %v", got, tt.want)
			}
		})
	}
}
