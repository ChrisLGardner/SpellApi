package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/honeycombio/beeline-go"
	"github.com/honeycombio/beeline-go/wrappers/hnygorilla"
)

var (
	spells []Spell
	dbUrl  string
)

func main() {
	// Initialize beeline. The only required field is WriteKey.
	beeline.Init(beeline.Config{
		WriteKey: os.Getenv("HONEYCOMB_KEY"),
		Dataset:  os.Getenv("HONEYCOMB_DATASET"),
	})
	// ensure everything gets sent off before we exit
	defer beeline.Close()

	spells = []Spell{}
	dbUrl = os.Getenv("COSMOSDB_URI")
	r := mux.NewRouter()
	r.Use(hnygorilla.Middleware)
	// Routes consist of a path and a handler function.
	r.HandleFunc("/", RootHandler)
	r.HandleFunc("/spells/{name}", GetSpellHandler).Methods("GET")
	r.HandleFunc("/spells", PostSpellHandler).Methods("POST")

	// Bind to a port and pass our router in
	port := os.Getenv("PORT")
	if port == "" {
		port = "80"
	}
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func RootHandler(w http.ResponseWriter, r *http.Request) {
	ctx, span := beeline.StartSpan(r.Context(), "home")
	defer span.Send()

	beeline.AddField(ctx, "test", "value")
	fmt.Fprint(w, "{\"result\":\"success\"}")
}

func GetSpellHandler(w http.ResponseWriter, r *http.Request) {
	ctx, span := beeline.StartSpan(r.Context(), "GetSpellHandler")
	defer span.Send()

	vars := mux.Vars(r)
	spellName := vars["name"]
	query := r.URL.Query()

	beeline.AddField(ctx, "GetSpellHandler.SpellName", spellName)
	beeline.AddField(ctx, "GetSpellHandler.Query", query)

	spell, err := FindSpell(ctx, spellName, query)
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

func PostSpellHandler(w http.ResponseWriter, r *http.Request) {
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

	var s Spell
	err = json.Unmarshal(body, &s)
	if err != nil {
		beeline.AddField(ctx, "PostSpellHandler.Error", err)
		http.Error(w, http.StatusText(http.StatusBadRequest),
			http.StatusBadRequest)
		return
	}

	beeline.AddField(ctx, "PostSpellHandler.Parsed", s)

	err = AddSpell(ctx, s)
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
