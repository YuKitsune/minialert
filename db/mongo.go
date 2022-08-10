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
	opts := options.Client().ApplyURI(cfg.Uri())

	dbFunc := func(ctx context.Context, cb Callback) error {
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

		db := client.Database(cfg.Database())

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

		filter := bson.D{{"guild_id", guildId}}
		update := toInterfaceSlice(reg)
		upsertOpts := options.Update().SetUpsert(true)

		_, err := coll.UpdateMany(ctx, filter, update, upsertOpts)
		if err != nil {
			return err
		}

		return nil
	})
}

func (r *lazyMongoRepo) GetRegisteredCommand(ctx context.Context, guildId string) ([]CommandRegistration, error) {

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

func (r *lazyMongoRepo) SetAlertsChannel(ctx context.Context, guildId string, channelId string) error {
	return r.dbFunc(ctx, func(ctx context.Context, db *mongo.Database) error {
		coll := db.Collection(AlertChannelsCollection.String())

		alertsChannel := AlertsChannel{
			GuildId:   guildId,
			ChannelId: channelId,
		}

		filter := bson.D{{"guild_id", guildId}}
		update := toInterfaceSlice(alertsChannel)
		upsertOpts := options.Update().SetUpsert(true)

		_, err := coll.UpdateMany(ctx, filter, update, upsertOpts)
		if err != nil {
			return err
		}

		return nil
	})
}

func (r *lazyMongoRepo) GetAlertsChannel(ctx context.Context, guildId string) (*AlertsChannel, error) {

	var channel AlertsChannel

	err := r.dbFunc(ctx, func(ctx context.Context, db *mongo.Database) error {
		coll := db.Collection(AlertChannelsCollection.String())

		filter := bson.D{{"guild_id", guildId}}
		res := coll.FindOne(ctx, filter)
		if res.Err() != nil {
			return res.Err()
		}

		err := res.Decode(&channel)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &channel, nil
}

func (r *lazyMongoRepo) SetAdminUser(ctx context.Context, guildId string, adminId string) error {
	return r.dbFunc(ctx, func(ctx context.Context, db *mongo.Database) error {
		coll := db.Collection(AdminUsersCollection.String())

		adminUser := AdminUser{
			GuildId: guildId,
			UserlId: adminId,
		}

		filter := bson.D{{"guild_id", guildId}}
		update := toInterfaceSlice(adminUser)
		upsertOpts := options.Update().SetUpsert(true)

		_, err := coll.UpdateMany(ctx, filter, update, upsertOpts)
		if err != nil {
			return err
		}

		return nil
	})
}

func (r *lazyMongoRepo) CreateInhibition(ctx context.Context, guildId string, alertName string) error {
	return r.dbFunc(ctx, func(ctx context.Context, db *mongo.Database) error {
		coll := db.Collection(InhibitionsCollection.String())

		inhibition := Inhibition{
			GuildId:   guildId,
			AlertName: alertName,
		}

		_, err := coll.InsertOne(ctx, toInterfaceSlice(inhibition))
		if err != nil {
			return err
		}

		return nil
	})
}

func (r *lazyMongoRepo) GetInhibitions(ctx context.Context, guildId string) ([]Inhibition, error) {

	var inhibitions []Inhibition

	err := r.dbFunc(ctx, func(ctx context.Context, db *mongo.Database) error {
		coll := db.Collection(InhibitionsCollection.String())

		filter := bson.D{{"guild_id", guildId}}
		cur, err := coll.Find(ctx, filter)
		if err != nil {
			return err
		}

		err = cur.All(ctx, &inhibitions)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return inhibitions, nil
}

func (r *lazyMongoRepo) DeleteInhibition(ctx context.Context, guildId string, alertName string) error {
	return r.dbFunc(ctx, func(ctx context.Context, db *mongo.Database) error {
		coll := db.Collection(InhibitionsCollection.String())

		filter := bson.D{{"guild_id", guildId}, {"alert_name", alertName}}

		_, err := coll.DeleteOne(ctx, filter)
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
			AlertChannelsCollection,
			AdminUsersCollection,
			InhibitionsCollection,
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

func toInterfaceSlice[v any](things ...v) []interface{} {

	var interfaces []interface{}
	for _, thing := range things {
		interfaces = append(interfaces, thing)
	}

	return interfaces
}
