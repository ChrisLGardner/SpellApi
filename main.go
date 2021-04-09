package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
	"github.com/honeycombio/beeline-go"
	"github.com/honeycombio/beeline-go/wrappers/hnygorilla"
)

var (
	spells []Spell
)

func main() {
	// Initialize beeline. The only required field is WriteKey.
	beeline.Init(beeline.Config{
		WriteKey: os.Getenv("HONEYCOMB_KEY"),
		Dataset:  os.Getenv("HONEYCOMB_DATASET"),
		STDOUT:   true,
	})
	// ensure everything gets sent off before we exit
	defer beeline.Close()

	spells = []Spell{}
	r := mux.NewRouter()
	r.Use(hnygorilla.Middleware)
	// Routes consist of a path and a handler function.
	r.HandleFunc("/", RootHandler)
	r.HandleFunc("/spells/{name}", GetSpellHandler).Methods("GET")
	r.HandleFunc("/spells", PostSpellHandler).Methods("POST")

	// Bind to a port and pass our router in
	log.Fatal(http.ListenAndServe(":8000", r))
}

func RootHandler(w http.ResponseWriter, r *http.Request) {
	ctx, span := beeline.StartSpan(r.Context(), "home")
	defer span.Send()

	beeline.AddField(ctx, "test", "value")
	fmt.Fprint(w, "{\"result\":\"success\"}")
}

func GetSpellHandler(w http.ResponseWriter, r *http.Request) {
	ctx, span := beeline.StartSpan(r.Context(), "GetSpell")
	defer span.Send()

	vars := mux.Vars(r)
	spellName := vars["name"]
	query := r.URL.Query()

	beeline.AddField(ctx, "GetSpell.Count", len(spells))
	beeline.AddField(ctx, "GetSpell.SpellName", spellName)
	beeline.AddField(ctx, "GetSpell.Query", query)
	if len(spells) < 1 {
		fmt.Fprint(w, "No Spells")
		return
	}

	for _, s := range spells {
		if s.Name == strings.ToLower(spellName) {
			json, err := json.Marshal(s)
			if err != nil {
				beeline.AddField(ctx, "GetSpell.Error", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError),
					http.StatusInternalServerError)
				return
			}

			fmt.Fprint(w, string(json))
			return
		}
	}

	beeline.AddField(ctx, "GetSpell.Error", "NotFound")
	http.Error(w, http.StatusText(http.StatusNotFound),
		http.StatusNotFound)
}

func PostSpellHandler(w http.ResponseWriter, r *http.Request) {
	ctx, span := beeline.StartSpan(r.Context(), "PostSpell")
	defer span.Send()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		beeline.AddField(ctx, "PostSpell.Error", err)
		http.Error(w, http.StatusText(http.StatusBadRequest),
			http.StatusBadRequest)
		return
	}

	beeline.AddField(ctx, "PostSpell.Raw", string(body))

	var s Spell
	err = json.Unmarshal(body, &s)
	if err != nil {
		beeline.AddField(ctx, "PostSpell.Error", err)
		http.Error(w, http.StatusText(http.StatusBadRequest),
			http.StatusBadRequest)
		return
	}

	beeline.AddField(ctx, "PostSpell.Parsed", s)

	spells = append(spells, s)

	fmt.Fprint(w, "Success")
}
