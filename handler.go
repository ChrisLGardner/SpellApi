package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/honeycombio/beeline-go"
)

func (s *SpellService) GetSpellHandler(w http.ResponseWriter, r *http.Request) {
	ctx, span := beeline.StartSpan(r.Context(), "GetSpellHandler")
	defer span.Send()

	vars := mux.Vars(r)
	spellName := vars["name"]
	query := r.URL.Query()

	beeline.AddField(ctx, "GetSpellHandler.SpellName", spellName)
	beeline.AddField(ctx, "GetSpellHandler.Query", query)

	spell, err := FindSpell(ctx, s.store, spellName, query)
	if err != nil && err.Error() == MultipleMatchingSpells {
		beeline.AddField(ctx, "GetSpellHandler.Error", "MultipleMatchingSpells")
		http.Error(w, MultipleMatchingSpells, http.StatusBadRequest)
		return
	} else if err != nil {
		beeline.AddField(ctx, "GetSpellHandler.Error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
		return
	}

	if spell.Name == "" {
		beeline.AddField(ctx, "GetSpellHandler.Error", "NotFound")
		http.Error(w, http.StatusText(http.StatusNotFound),
			http.StatusNotFound)
		return
	}

	json, err := json.Marshal(spell)
	if err != nil {
		beeline.AddField(ctx, "GetSpellHandler.Error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
		return
	}
	fmt.Fprint(w, string(json))
}

func (s *SpellService) PostSpellHandler(w http.ResponseWriter, r *http.Request) {
	ctx, span := beeline.StartSpan(r.Context(), "PostSpell")
	defer span.Send()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		beeline.AddField(ctx, "PostSpellHandler.Error", err)
		http.Error(w, http.StatusText(http.StatusBadRequest),
			http.StatusBadRequest)
		return
	}

	beeline.AddField(ctx, "PostSpellHandler.Raw", string(body))

	spell, err := ParseSpell(ctx, body)
	if err != nil && strings.Contains(err.Error(), "missing required") {
		beeline.AddField(ctx, "PostSpellHandler.Error", "MissingRequiredField")
		resp := fmt.Sprintf("%v: %v", http.StatusText(http.StatusBadRequest), err.Error())
		http.Error(w, resp,
			http.StatusBadRequest)
		return
	} else if err != nil {
		beeline.AddField(ctx, "PostSpellHandler.Error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
		return
	}

	beeline.AddField(ctx, "PostSpellHandler.Parsed", spell)

	err = AddSpell(ctx, s.store, spell)
	if err != nil && err.Error() == SpellAlreadyExists {
		beeline.AddField(ctx, "PostSpellHandler.Error", err)
		http.Error(w, SpellAlreadyExists,
			http.StatusConflict)
		return
	} else if err != nil {
		beeline.AddField(ctx, "PostSpellHandler.Error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	fmt.Fprint(w, "Spell added")
}

func (s *SpellService) DeleteSpellHandler(w http.ResponseWriter, r *http.Request) {
	ctx, span := beeline.StartSpan(r.Context(), "DeleteSpell")
	defer span.Send()

	if deleteEnabled := s.flags.GetBoolFlag(ctx, "delete-spell", s.flags.GetUser(ctx, r)); deleteEnabled {
		beeline.AddField(ctx, "DeleteSpell.Flag", deleteEnabled)
		vars := mux.Vars(r)
		spellName := vars["name"]
		query := r.URL.Query()

		err := DeleteSpell(ctx, s.store, spellName, query)
		if err != nil {
			beeline.AddField(ctx, "GetSpellHandler.Error", "NotFound")
			http.Error(w, http.StatusText(http.StatusNotFound),
				http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusAccepted)
		fmt.Fprint(w, "Spell Removed")

	} else {
		beeline.AddField(ctx, "DeleteSpell.Flag", deleteEnabled)
		http.Error(w, http.StatusText(http.StatusForbidden),
			http.StatusForbidden)
		return
	}

}
