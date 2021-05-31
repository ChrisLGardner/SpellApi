package main

import (
	"log"
	"net/http"
	"os"

	"github.com/chrislgardner/spellapi/db"
	"github.com/gorilla/mux"
	"github.com/honeycombio/beeline-go"
	"github.com/honeycombio/beeline-go/wrappers/hnygorilla"
)

var (
	dbUrl string
)

func main() {
	// Initialize beeline. The only required field is WriteKey.
	beeline.Init(beeline.Config{
		WriteKey: os.Getenv("HONEYCOMB_KEY"),
		Dataset:  os.Getenv("HONEYCOMB_DATASET"),
	})
	// ensure everything gets sent off before we exit
	defer beeline.Close()

	dbUrl = os.Getenv("COSMOSDB_URI")
	db, err := db.ConnectDb(dbUrl)
	if err != nil {
		panic(err)
	}

	var spellService SpellService
	if ldApiKey := os.Getenv("LAUNCHDARKLY_KEY"); ldApiKey != "" {
		ldclient, err := NewLaunchDarklyClient(ldApiKey, 5)
		if err != nil {
			panic(err)
		}

		spellService = SpellService{
			store: db,
			flags: ldclient,
		}
	} else {
		spellService = SpellService{
			store: db,
		}
	}

	r := mux.NewRouter()
	r.Use(hnygorilla.Middleware)
	// Routes consist of a path and a handler function.
	r.HandleFunc("/spells/{name}", spellService.GetSpellHandler).Methods("GET")
	r.HandleFunc("/spells", spellService.PostSpellHandler).Methods("POST")

	// Bind to a port and pass our router in
	port := os.Getenv("PORT")
	if port == "" {
		port = "80"
	}
	log.Fatal(http.ListenAndServe(":"+port, r))
}
