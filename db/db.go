package db

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type DB struct {
	*mongo.Client
}

// Connect to the specified mongo instance using the context for timeout
func ConnectDb(uri string) (*DB, error) {
	ctx := context.Background()
	clientOptions := options.Client().ApplyURI(uri).SetDirect(true)
	c, err := mongo.NewClient(clientOptions)
	if err != nil {
		return nil, err
	}

	err = c.Connect(ctx)
	if err != nil {
		return nil, err
	}

	err = c.Ping(ctx, nil)
	if err != nil {
		return nil, err
	}

	return &DB{c}, nil
}

//	collection := mc.Database("reminders").Collection("reminders")

func runQuery(ctx context.Context, mc *mongo.Collection, query interface{}) ([]bson.M, error) {

	tracer := otel.Tracer("Encantus")
	ctx, span := tracer.Start(ctx, "Mongo.RunQuery")
	defer span.End()

	span.SetAttributes(
		attribute.String("Mongo.RunQuery.Collection", mc.Name()),
		attribute.String("Mongo.RunQuery.Database", mc.Database().Name()),
	)

	cursor, err := mc.Find(ctx, query)
	if err != nil {
		span.SetAttributes(attribute.String("Mongo.RunQuery.Error", err.Error()))
		return nil, err
	}

	var results []bson.M
	if err = cursor.All(ctx, &results); err != nil {
		span.SetAttributes(attribute.String("Mongo.RunQuery.Error", err.Error()))
		return nil, err
	}
	span.SetAttributes(
		attribute.Int("Mongo.RunQuery.Results.Count", len(results)),
		attribute.String("Mongo.RunQuery.Results.Raw", fmt.Sprintf("%v", results)),
	)

	return results, nil
}

func writeDbObject(ctx context.Context, mc *mongo.Collection, obj []byte) error {

	tracer := otel.Tracer("Encantus")
	ctx, span := tracer.Start(ctx, "Mongo.WriteObject")
	defer span.End()

	span.SetAttributes(
		attribute.String("Mongo.RunQuery.Collection", mc.Name()),
		attribute.String("Mongo.RunQuery.Database", mc.Database().Name()),
	)

	res, err := mc.InsertOne(ctx, obj)
	if err != nil {
		span.SetAttributes(attribute.String("Mongo.WriteObject.Error", err.Error()))
		return err
	}

	span.SetAttributes(attribute.String("Mongo.WriteObject.Id", fmt.Sprint(res.InsertedID)))

	return nil
}

func deleteDbObject(ctx context.Context, mc *mongo.Collection, query interface{}) error {

	tracer := otel.Tracer("Encantus")
	ctx, span := tracer.Start(ctx, "Mongo.DeleteDbObject")
	defer span.End()

	span.SetAttributes(
		attribute.String("Mongo.RunQuery.Collection", mc.Name()),
		attribute.String("Mongo.RunQuery.Database", mc.Database().Name()),
	)

	deleted, err := mc.DeleteOne(ctx, query)
	if err != nil {
		span.SetAttributes(attribute.String("Mongo.DeleteDbObject.Error", err.Error()))
		return err
	}

	span.SetAttributes(attribute.Int64("Mongo.DeleteDbObject.DeletedCount", deleted.DeletedCount))

	return nil
}

func getDistinctValues(ctx context.Context, mc *mongo.Collection, key string) ([]string, error) {
	tracer := otel.Tracer("Encantus")
	ctx, span := tracer.Start(ctx, "Mongo.getDistinctValues")
	defer span.End()

	span.SetAttributes(
		attribute.String("Mongo.getDistinctValues.Collection", mc.Name()),
		attribute.String("Mongo.getDistinctValues.Database", mc.Database().Name()),
	)

	results, err := mc.Distinct(ctx, key, bson.D{})
	if err != nil {
		span.SetAttributes(attribute.String("Mongo.getDistinctValues.Error", err.Error()))
		return nil, err
	}

	span.SetAttributes(attribute.String("Mongo.getDistinctValues.RawResults", fmt.Sprintf("%v", results)))
	res := make([]string, len(results))
	for i, v := range results {
		res[i] = fmt.Sprintf("%v", v)
	}
	return res, nil
}

func (db *DB) GetSpell(ctx context.Context, search bson.M) ([]bson.M, error) {

	tracer := otel.Tracer("Encantus")
	ctx, span := tracer.Start(ctx, "Mongo.GetSpell")
	defer span.End()

	span.SetAttributes(attribute.String("Mongo.GetSpell.Query", fmt.Sprintf("%v", search)))

	collection := db.Database("spellapi").Collection("spells")

	result, err := runQuery(ctx, collection, search)
	if err != nil {
		span.SetAttributes(attribute.String("Mongo.GetSpell.Error", err.Error()))
		return nil, err
	}

	span.SetAttributes(attribute.String("Mongo.GetSpell.Result", fmt.Sprintf("%v", result)))

	return result, nil
}

func (db *DB) AddSpell(ctx context.Context, spell []byte) error {

	tracer := otel.Tracer("Encantus")
	ctx, span := tracer.Start(ctx, "Mongo.AddSpell")
	defer span.End()

	span.SetAttributes(attribute.String("Mongo.AddSpell.Spell", string(spell)))

	collection := db.Database("spellapi").Collection("spells")

	err := writeDbObject(ctx, collection, spell)
	if err != nil {
		span.SetAttributes(attribute.String("Mongo.AddSpell.Error", err.Error()))
		return err
	}

	return nil
}

func (db *DB) DeleteSpell(ctx context.Context, spell bson.M) error {

	tracer := otel.Tracer("Encantus")
	ctx, span := tracer.Start(ctx, "Mongo.DeleteSpell")
	defer span.End()

	span.SetAttributes(attribute.String("Mongo.DeleteSpell.Spell", fmt.Sprintf("%v", spell)))

	collection := db.Database("spellapi").Collection("spells")

	err := deleteDbObject(ctx, collection, spell)
	if err != nil {
		span.SetAttributes(attribute.String("Mongo.DeleteSpell.Error", err.Error()))
		return err
	}

	return nil
}

func (db *DB) GetMetadataValues(ctx context.Context, metadataName string) ([]string, error) {
	tracer := otel.Tracer("Encantus")
	ctx, span := tracer.Start(ctx, "Mongo.GetMetadataValues")
	defer span.End()

	collection := db.Database("spellapi").Collection("spells")

	values, err := getDistinctValues(ctx, collection, metadataName)
	if err != nil {
		span.SetAttributes(attribute.String("Mongo.GetMetadataValues.Error", err.Error()))
		return nil, err
	}

	return values, nil
}
