package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/honeycombio/beeline-go"
	"github.com/honeycombio/beeline-go/wrappers/hnygorilla"
	"github.com/honeycombio/beeline-go/wrappers/hnynethttp"
)

func main() {
	// Initialize beeline. The only required field is WriteKey.
	beeline.Init(beeline.Config{
		WriteKey: os.Getenv("HONEYCOMB_KEY"),
		Dataset:  os.Getenv("HONEYCOMB_DATASET"),
	})
	// ensure everything gets sent off before we exit
	defer beeline.Close()

	r := mux.NewRouter()
	r.Use(hnygorilla.Middleware)
	// Routes consist of a path and a handler function.
	r.HandleFunc("/", RootHandler)

	// Bind to a port and pass our router in
	log.Fatal(http.ListenAndServe(":8000", hnynethttp.WrapHandler(r)))
}

func RootHandler(w http.ResponseWriter, r *http.Request) {
	ctx, span := beeline.StartSpan(r.Context(), "home")
	defer span.Send()

	beeline.AddField(ctx, "test", "value")
	fmt.Fprint(w, "{\"result\":\"success\"}")
}
