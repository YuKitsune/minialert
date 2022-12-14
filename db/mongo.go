package db

import (
	"context"
	"github.com/yukitsune/minialert/config"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"time"
)

func SetupMongoDatabase(cfg config.Database) Repo {
	dbFunc := func(ctx context.Context, cb Callback) error {
		uri, err := cfg.Uri()
		if err != nil {
			return nil
		}

		opts := options.Client().ApplyURI(uri)
		client, err := mongo.NewClient(opts)
		if err != nil {
			return err
		}

		ctx, cancelCtx := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelCtx()

		err = client.Connect(ctx)
		if err != nil {
			return err
		}

		err = client.Ping(ctx, readpref.Primary())
		if err != nil {
			return err
		}

		databaseName, err := cfg.Database()
		if err != nil {
			return err
		}

		db := client.Database(databaseName)

		return cb(ctx, db)
	}

	return &lazyMongoRepo{dbFunc}
}

type lazyMongoRepo struct {
	dbFunc Func
}

func (r *lazyMongoRepo) RegisterCommand(ctx context.Context, guildId string, commandId string, commandName string) error {
	return r.dbFunc(ctx, func(ctx context.Context, db *mongo.Database) error {
		coll := db.Collection(CommandRegistrationsCollection.String())

		reg := CommandRegistration{
			GuildId:     guildId,
			CommandId:   commandId,
			CommandName: commandName,
		}

		filter := bson.D{
			{"guild_id", guildId},
			{"command_id", commandId},
		}

		upsert := bson.M{"$set": reg}

		upsertOpts := options.Update().SetUpsert(true)

		_, err := coll.UpdateOne(ctx, filter, upsert, upsertOpts)
		if err != nil {
			return err
		}

		return nil
	})
}

func (r *lazyMongoRepo) GetRegisteredCommands(ctx context.Context, guildId string) ([]CommandRegistration, error) {

	var commands []CommandRegistration

	err := r.dbFunc(ctx, func(ctx context.Context, db *mongo.Database) error {
		coll := db.Collection(CommandRegistrationsCollection.String())

		filter := bson.D{{"guild_id", guildId}}

		cur, err := coll.Find(ctx, filter)
		if err != nil {
			return err
		}

		err = cur.All(ctx, &commands)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return commands, nil
}

func (r *lazyMongoRepo) GetGuildConfigs(ctx context.Context) (cfgs []GuildConfig, err error) {
	err = r.dbFunc(ctx, func(ctx context.Context, db *mongo.Database) error {
		coll := db.Collection(GuildConfigCollection.String())

		filter := bson.D{{}}

		cur, err := coll.Find(ctx, filter)
		if err != nil {
			return err
		}

		err = cur.All(ctx, &cfgs)

		return err
	})

	return cfgs, err
}

func (r *lazyMongoRepo) GetGuildConfig(ctx context.Context, guildId string) (cfg *GuildConfig, err error) {
	err = r.dbFunc(ctx, func(ctx context.Context, db *mongo.Database) error {
		coll := db.Collection(GuildConfigCollection.String())

		filter := bson.D{{"guild_id", guildId}}

		res := coll.FindOne(ctx, filter)
		if res.Err() != nil {
			return res.Err()
		}

		err = res.Decode(&cfg)
		if err != nil {
			return err
		}

		return nil
	})

	return cfg, err
}

func (r *lazyMongoRepo) SetGuildConfig(ctx context.Context, config *GuildConfig) error {
	return r.dbFunc(ctx, func(ctx context.Context, db *mongo.Database) error {
		coll := db.Collection(GuildConfigCollection.String())

		filter := bson.D{{"guild_id", config.GuildId}}
		upsert := bson.M{"$set": config}
		upsertOpts := options.Update().SetUpsert(true)

		_, err := coll.UpdateMany(ctx, filter, upsert, upsertOpts)
		if err != nil {
			return err
		}

		return nil
	})
}

func (r *lazyMongoRepo) ClearGuildInfo(ctx context.Context, guildId string) error {
	return r.dbFunc(ctx, func(ctx context.Context, db *mongo.Database) error {

		filter := bson.D{{"guild_id", guildId}}

		collections := []CollectionName{
			CommandRegistrationsCollection,
			GuildConfigCollection,
		}

		for _, collection := range collections {
			coll := db.Collection(collection.String())
			_, err := coll.DeleteMany(ctx, filter)
			if err != nil {
				return err
			}
		}

		return nil
	})
}
