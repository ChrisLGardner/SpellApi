package db

import (
	"context"

	"github.com/honeycombio/beeline-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Connect to the specified mongo instance using the context for timeout
func ConnectDb(ctx context.Context, uri string) (*mongo.Client, error) {
	ctx, span := beeline.StartSpan(ctx, "Mongo.Connect")
	defer span.Send()

	beeline.AddField(ctx, "Mongo.Server", uri)

	clientOptions := options.Client().ApplyURI(uri).SetDirect(true)
	c, err := mongo.NewClient(clientOptions)
	if err != nil {
		beeline.AddField(ctx, "Mongo.Client.Error", err)
		return nil, err
	}

	err = c.Connect(ctx)
	if err != nil {
		beeline.AddField(ctx, "Mongo.Connect.Error", err)
		return nil, err
	}

	err = c.Ping(ctx, nil)
	if err != nil {
		beeline.AddField(ctx, "Mongo.Ping.Error", err)
		return nil, err
	}

	return c, nil
}

//	collection := mc.Database("reminders").Collection("reminders")

func runQuery(ctx context.Context, mc *mongo.Collection, query interface{}) ([]bson.M, error) {

	ctx, span := beeline.StartSpan(ctx, "Mongo.RunQuery")
	defer span.Send()

	beeline.AddField(ctx, "Mongo.RunQuery.Collection", mc.Name())
	beeline.AddField(ctx, "Mongo.RunQuery.Database", mc.Database().Name())
	beeline.AddField(ctx, "Mongo.RunQuery.Query", query)

	cursor, err := mc.Find(ctx, query)
	if err != nil {
		beeline.AddField(ctx, "Mongo.RunQuery.Error", err)
		return nil, err
	}

	var results []bson.M
	if err = cursor.All(ctx, &results); err != nil {
		beeline.AddField(ctx, "Mongo.RunQuery.Error", err)
		return nil, err
	}

	beeline.AddField(ctx, "Mongo.RunQuery.Results.Count", len(results))
	beeline.AddField(ctx, "Mongo.RunQuery.Results.Raw", results)

	return results, nil
}

func writeDbObject(ctx context.Context, mc *mongo.Collection, obj interface{}) error {

	ctx, span := beeline.StartSpan(ctx, "Mongo.WriteObject")
	defer span.Send()

	data, err := bson.Marshal(obj)
	if err != nil {
		beeline.AddField(ctx, "Mongo.WriteObject.Error", err)
		return err
	}

	beeline.AddField(ctx, "Mongo.WriteObject.Collection", mc.Name())
	beeline.AddField(ctx, "Mongo.WriteObject.Database", mc.Database().Name())
	beeline.AddField(ctx, "Mongo.WriteObject.Object", data)

	res, err := mc.InsertOne(ctx, data)
	if err != nil {
		beeline.AddField(ctx, "Mongo.WriteObject.Error", err)
		return err
	}

	beeline.AddField(ctx, "Mongo.WriteObject.Id", res.InsertedID)

	return nil
}

func GetSpell(ctx context.Context, mc *mongo.Client, spell bson.M) ([]bson.M, error) {

	ctx, span := beeline.StartSpan(ctx, "Mongo.GetSpell")
	defer span.Send()

	beeline.AddField(ctx, "Mongo.GetSpell.Query", spell)

	collection := mc.Database("spellapi").Collection("spells")

	result, err := runQuery(ctx, collection, spell)
	if err != nil {
		beeline.AddField(ctx, "Mongo.GetSpell.Error", err)
		return nil, err
	}

	beeline.AddField(ctx, "Mongo.GetSpell.Result", result)

	return result, nil
}

func AddSpell(ctx context.Context, mc *mongo.Client, spell []byte) error {

	ctx, span := beeline.StartSpan(ctx, "Mongo.AddSpell")
	defer span.Send()

	beeline.AddField(ctx, "Mongo.AddSpell.Query", spell)

	collection := mc.Database("spellapi").Collection("spells")

	err := writeDbObject(ctx, collection, spell)
	if err != nil {
		beeline.AddField(ctx, "Mongo.AddSpell.Error", err)
		return err
	}

	return nil
}
