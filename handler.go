package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

func (s *SpellService) GetSpellHandler(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("Encantus")
	ctx, span := tracer.Start(r.Context(), "GetSpellHandler")
	defer span.End()

	vars := mux.Vars(r)
	spellName := vars["name"]
	query := r.URL.Query()

	span.SetAttributes(
		attribute.String("GetSpellHandler.SpellName", spellName),
		attribute.String("GetSpellHandler.Query", query.Encode()),
	)

	spell, err := FindSpell(ctx, s.store, spellName, query)
	if err != nil && err.Error() == MultipleMatchingSpells {
		span.SetAttributes(attribute.String("GetSpellHandler.Error", "MultipleMatchingSpells"))
		http.Error(w, MultipleMatchingSpells, http.StatusBadRequest)
		return
	} else if err != nil {
		span.SetAttributes(attribute.String("GetSpellHandler.Error", err.Error()))
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
		return
	}

	if spell.Name == "" {
		span.SetAttributes(attribute.String("GetSpellHandler.Error", "NotFound"))
		http.Error(w, http.StatusText(http.StatusNotFound),
			http.StatusNotFound)
		return
	}

	json, err := json.Marshal(spell)
	if err != nil {
		span.SetAttributes(attribute.String("GetSpellHandler.Error", err.Error()))
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
		return
	}
	fmt.Fprint(w, string(json))
}

func (s *SpellService) PostSpellHandler(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("Encantus")
	ctx, span := tracer.Start(r.Context(), "PostSpellHandler")
	defer span.End()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		span.SetAttributes(attribute.String("PostSpellHandler.Error", err.Error()))
		http.Error(w, http.StatusText(http.StatusBadRequest),
			http.StatusBadRequest)
		return
	}

	if multipostEnabled := s.flags.GetBoolFlag(ctx, "multipost-spell", s.flags.GetUser(ctx, r)); multipostEnabled {
		span.SetAttributes(attribute.Bool("PostSpellHandler.Multipost.Flag", multipostEnabled))

		var incomingRequest Request
		err = json.NewDecoder(bytes.NewReader(body)).Decode(&incomingRequest)
		if err != nil {
			span.SetAttributes(attribute.String("PostSpellHandler.Error", err.Error()))
			http.Error(w, http.StatusText(http.StatusInternalServerError),
				http.StatusInternalServerError)
			return
		}

		resp := Response{}
		resp.TraceId = span.SpanContext().TraceID().String()
		resp.Data = []ErrorResponse{}
		errorOccured := false

		for _, d := range incomingRequest.Data {

			temp, err := json.Marshal(d)
			if err != nil {
				resp.Data = append(resp.Data, ErrorResponse{err.Error(), http.StatusBadRequest})
				errorOccured = true
				continue
			}

			spell, err := ParseSpell(ctx, temp)
			if err != nil && strings.Contains(err.Error(), "missing required") {
				resp.Data = append(resp.Data, ErrorResponse{err.Error(), http.StatusBadRequest})
				errorOccured = true
				continue

			} else if err != nil {
				resp.Data = append(resp.Data, ErrorResponse{err.Error(), http.StatusInternalServerError})
				errorOccured = true
				continue
			}

			span.SetAttributes(attribute.Stringer("PostSpellHandler.Parsed", spell))

			err = AddSpell(ctx, s.store, spell)
			if err != nil && err.Error() == SpellAlreadyExists {
				resp.Data = append(resp.Data, ErrorResponse{err.Error(), http.StatusConflict})
				errorOccured = true
				continue
			} else if err != nil {
				resp.Data = append(resp.Data, ErrorResponse{err.Error(), http.StatusInternalServerError})
				errorOccured = true
				continue
			}
		}

		resp.Count = len(incomingRequest.Data)
		span.SetAttributes(attribute.Int("PostSpellHandler.SpellCount", resp.Count))

		if errorOccured {
			resp.ResponseCode = http.StatusBadRequest
			resp.ResponseMessage = "Some errors occured while processing input. See Data property for more details."
			span.SetAttributes(attribute.String("PostSpellHandler.Error", "MissingRequiredField"))
			w.WriteHeader(http.StatusBadRequest)
		} else {
			resp.ResponseCode = http.StatusCreated
			resp.ResponseMessage = "Spell(s) added"
			w.WriteHeader(http.StatusCreated)
		}

		responseBytes, err := json.Marshal(resp)
		if err != nil {
			span.SetAttributes(attribute.String("PostSpellHandler.Error", err.Error()))
			http.Error(w, http.StatusText(http.StatusInternalServerError),
				http.StatusInternalServerError)
			return
		}

		w.Write(responseBytes)

	} else {

		span.SetAttributes(attribute.String("PostSpellHandler.Raw", string(body)))

		spell, err := ParseSpell(ctx, body)
		if err != nil && strings.Contains(err.Error(), "missing required") {
			span.SetAttributes(attribute.String("PostSpellHandler.Error", "MissingRequiredField"))
			resp := fmt.Sprintf("%v: %v", http.StatusText(http.StatusBadRequest), err.Error())
			http.Error(w, resp,
				http.StatusBadRequest)
			return
		} else if err != nil {
			span.SetAttributes(attribute.String("PostSpellHandler.Error", err.Error()))
			http.Error(w, http.StatusText(http.StatusInternalServerError),
				http.StatusInternalServerError)
			return
		}

		span.SetAttributes(attribute.Stringer("PostSpellHandler.Parsed", spell))

		err = AddSpell(ctx, s.store, spell)
		if err != nil && err.Error() == SpellAlreadyExists {
			span.SetAttributes(attribute.String("PostSpellHandler.Error", err.Error()))
			http.Error(w, SpellAlreadyExists,
				http.StatusConflict)
			return
		} else if err != nil {
			span.SetAttributes(attribute.String("PostSpellHandler.Error", err.Error()))
			http.Error(w, http.StatusText(http.StatusInternalServerError),
				http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, "Spell added")
	}
}

func (s *SpellService) DeleteSpellHandler(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("Encantus")
	ctx, span := tracer.Start(r.Context(), "DeleteSpellHandler")
	defer span.End()

	if deleteEnabled := s.flags.GetBoolFlag(ctx, "delete-spell", s.flags.GetUser(ctx, r)); deleteEnabled {
		span.SetAttributes(attribute.Bool("DeleteSpellHandler.Flag", deleteEnabled))
		vars := mux.Vars(r)
		spellName := vars["name"]
		query := r.URL.Query()

		err := DeleteSpell(ctx, s.store, spellName, query)
		if err != nil {
			span.SetAttributes(attribute.String("DeleteSpellHandler.Error", "NotFound"))
			http.Error(w, http.StatusText(http.StatusNotFound),
				http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusAccepted)
		fmt.Fprint(w, "Spell Removed")

	} else {
		span.SetAttributes(attribute.Bool("DeleteSpellHandler.Flag", deleteEnabled))
		http.Error(w, http.StatusText(http.StatusForbidden),
			http.StatusForbidden)
		return
	}

}

func (s *SpellService) GetAllSpellHandler(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("Encantus")
	ctx, span := tracer.Start(r.Context(), "GetAllSpellHandler")
	defer span.End()

	query := r.URL.Query()

	span.SetAttributes(attribute.String("GetAllSpellHandler.Query", query.Encode()))

	spells, err := GetAllSpell(ctx, s.store, query)
	if err != nil {
		span.SetAttributes(attribute.String("GetAllSpellHandler.Error", "NotFound"))
		http.Error(w, http.StatusText(http.StatusNotFound),
			http.StatusNotFound)
		return
	}
	json, err := json.Marshal(spells)
	if err != nil {
		span.SetAttributes(attribute.String("GetAllSpellHandler.Error", err.Error()))
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
		return
	}
	fmt.Fprint(w, string(json))

}

func (s *SpellService) GetSpellMetadataHandler(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("Encantus")
	ctx, span := tracer.Start(r.Context(), "GetSpellMetadataHandler")
	defer span.End()

	if metadataEnabled := s.flags.GetBoolFlag(ctx, "get-spell-metadata", s.flags.GetUser(ctx, r)); metadataEnabled {
		span.SetAttributes(attribute.Bool("GetSpellMetadataHandler.Flag", metadataEnabled))

		vars := mux.Vars(r)
		metadataName := vars["name"]

		span.SetAttributes(attribute.String("GetSpellMetadataHandler.MetadataName", metadataName))

		metadata, err := GetSpellMetadata(ctx, s.store, metadataName)
		if err != nil {
			span.SetAttributes(attribute.String("GetSpellMetadataHandler.Error", "NotFound"))
			http.Error(w, http.StatusText(http.StatusNotFound),
				http.StatusNotFound)
			return
		}
		json, err := json.Marshal(metadata)
		if err != nil {
			span.SetAttributes(attribute.String("GetSpellMetadataHandler.Error", err.Error()))
			http.Error(w, http.StatusText(http.StatusInternalServerError),
				http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "{\"%s\":%s}", metadataName, string(json))
	} else {
		span.SetAttributes(attribute.Bool("GetSpellMetadataHandler.Flag", metadataEnabled))
		http.Error(w, http.StatusText(http.StatusForbidden),
			http.StatusForbidden)
		return
	}
}

func (s *SpellService) GetAllSpellMetadataHandler(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("Encantus")
	ctx, span := tracer.Start(r.Context(), "GetAllSpellMetadataHandler")
	defer span.End()

	if metadataEnabled := s.flags.GetBoolFlag(ctx, "get-spell-metadata-names", s.flags.GetUser(ctx, r)); metadataEnabled {
		span.SetAttributes(attribute.Bool("GetAllSpellMetadataHandler.Flag", metadataEnabled))

		query := r.URL.Query()

		span.SetAttributes(attribute.String("GetAllSpellMetadataHandler.Query", query.Encode()))

		metadata, err := GetAllSpellMetadata(ctx, s.store, query)
		if err != nil {
			span.SetAttributes(attribute.String("GetAllSpellMetadataHandler.Error", "NotFound"))
			http.Error(w, http.StatusText(http.StatusNotFound),
				http.StatusNotFound)
			return
		}
		json, err := json.Marshal(metadata)
		if err != nil {
			span.SetAttributes(attribute.String("GetAllSpellMetadataHandler.Error", err.Error()))
			http.Error(w, http.StatusText(http.StatusInternalServerError),
				http.StatusInternalServerError)
			return
		}
		fmt.Fprint(w, string(json))
	} else {
		span.SetAttributes(attribute.Bool("GetAllSpellMetadataHandler.Flag", metadataEnabled))
		http.Error(w, http.StatusText(http.StatusForbidden),
			http.StatusForbidden)
		return
	}
}
