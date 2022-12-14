package db

import (
	"context"
	"fmt"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/yukitsune/minialert/slices"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"os"
	"testing"
)

const databaseName string = "minialert_test"

var mongoRepo Repo
var mongoClient *mongo.Client

type testDatabaseConfig struct {
	uri      string
	database string
}

func (c *testDatabaseConfig) Uri() (string, error) {
	return c.uri, nil
}

func (c *testDatabaseConfig) Database() (string, error) {
	return c.database, nil
}

func (c *testDatabaseConfig) UseInMemoryDatabase() bool {
	return false
}

func TestMain(m *testing.M) {

	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not construct pool: %s", err)
	}

	err = pool.Client.Ping()
	if err != nil {
		log.Fatalf("Could not connect to Docker: %s", err)
	}

	// Pull mongodb docker image for version 5.0
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "mongo",
		Tag:        "latest",
		Env: []string{
			// username and password for mongodb superuser
			"MONGO_INITDB_ROOT_USERNAME=root",
			"MONGO_INITDB_ROOT_PASSWORD=password",
		},
	}, func(config *docker.HostConfig) {
		// set AutoRemove to true so that stopped container goes away by itself
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{
			Name: "no",
		}
	})

	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	uri := fmt.Sprintf("mongodb://root:password@localhost:%s", resource.GetPort("27017/tcp"))

	// Exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	err = pool.Retry(func() error {
		var err error
		mongoClient, err = mongo.Connect(
			context.TODO(),
			options.Client().ApplyURI(uri),
		)
		if err != nil {
			return err
		}

		return mongoClient.Ping(context.TODO(), nil)
	})

	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	config := &testDatabaseConfig{
		uri:      uri,
		database: databaseName,
	}

	mongoRepo = SetupMongoDatabase(config)

	// Run tests
	code := m.Run()

	// When you're done, kill and remove the container
	if err = pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}

	// disconnect mongodb client
	if err = mongoClient.Disconnect(context.TODO()); err != nil {
		panic(err)
	}

	os.Exit(code)
}

func TestRegisterCommand(t *testing.T) {
	// Arrange
	ctx := context.Background()
	guildId := "foo"
	commandId := "bar"
	commandName := "baz"

	// Act
	err := mongoRepo.RegisterCommand(ctx, guildId, commandId, commandName)
	if err != nil {
		t.Fail()
	}

	// Assert
	db := mongoClient.Database(databaseName)
	coll := db.Collection("command_registrations")
	res := coll.FindOne(ctx, bson.D{{"guild_id", guildId}, {"command_id", commandId}})

	var reg *CommandRegistration
	if err = res.Decode(&reg); err != nil {
		t.Fail()
	}

	if reg.CommandName != commandName {
		t.Fail()
	}
}

func TestGetRegisterCommands(t *testing.T) {
	// Arrange
	ctx := context.Background()
	guildId := "foo"
	commandId := "bar"
	commandName := "baz"
	err := mongoRepo.RegisterCommand(ctx, guildId, commandId, commandName)
	if err != nil {
		t.Fail()
	}

	// Act
	commands, err := mongoRepo.GetRegisteredCommands(ctx, guildId)
	if err != nil {
		t.Fail()
	}

	// Assert
	if !slices.HasMatching(commands, func(c CommandRegistration) bool {
		return c.CommandId == commandId && c.CommandName == commandName
	}) {
		t.Fail()
	}
}

func TestSetGuildConfig(t *testing.T) {
	// Arrange
	ctx := context.Background()
	guildId := "foo"

	guildConfig := &GuildConfig{
		GuildId:       guildId,
		ScrapeConfigs: []ScrapeConfig{},
	}

	// Act
	err := mongoRepo.SetGuildConfig(ctx, guildConfig)
	if err != nil {
		t.Fail()
	}

	// Assert
	db := mongoClient.Database(databaseName)
	coll := db.Collection("guild_config")
	res := coll.FindOne(ctx, bson.D{{"guild_id", guildId}})

	var foundGuildConfig *GuildConfig
	if err = res.Decode(&foundGuildConfig); err != nil {
		t.Fail()
	}

	if foundGuildConfig.GuildId != guildId {
		t.Fail()
	}

	if len(foundGuildConfig.ScrapeConfigs) != 0 {
		t.Fail()
	}
}

func TestSetGuildConfigAddsScrapeConfigs(t *testing.T) {
	// Arrange
	ctx := context.Background()
	guildId := "foo"

	guildConfig := &GuildConfig{
		GuildId:       guildId,
		ScrapeConfigs: []ScrapeConfig{},
	}
	err := mongoRepo.SetGuildConfig(ctx, guildConfig)
	if err != nil {
		t.Fail()
	}

	// Act
	scrapeConfigName := "My scrape config"
	inhibitedAlertName := "test_alert"
	scrapeConfig := ScrapeConfig{
		Name:                  scrapeConfigName,
		Endpoint:              "http://localhost:1234",
		Username:              "foo",
		Password:              "bar",
		ScrapeIntervalMinutes: 1,
		AlertChannelId:        "123",
		InhibitedAlerts:       []string{inhibitedAlertName},
	}

	guildConfig.ScrapeConfigs = append(guildConfig.ScrapeConfigs, scrapeConfig)

	err = mongoRepo.SetGuildConfig(ctx, guildConfig)
	if err != nil {
		t.Fail()
	}

	// Assert
	db := mongoClient.Database(databaseName)
	coll := db.Collection("guild_config")
	res := coll.FindOne(ctx, bson.D{{"guild_id", guildId}})

	var foundGuildConfig *GuildConfig
	if err = res.Decode(&foundGuildConfig); err != nil {
		t.Fail()
	}

	if foundGuildConfig.GuildId != guildId {
		t.Fail()
	}

	if !slices.HasMatching(foundGuildConfig.ScrapeConfigs, func(config ScrapeConfig) bool {
		return config.Name == scrapeConfigName && slices.Contains(config.InhibitedAlerts, inhibitedAlertName)
	}) {
		t.Fail()
	}
}

func TestSetGuildConfigUpdatesScrapeConfigs(t *testing.T) {
	// Arrange
	ctx := context.Background()
	guildId := "foo"

	scrapeConfigName := "My scrape config"
	inhibitedAlertName := "test_alert"
	scrapeConfig := ScrapeConfig{
		Name:                  scrapeConfigName,
		Endpoint:              "http://localhost:1234",
		Username:              "foo",
		Password:              "bar",
		ScrapeIntervalMinutes: 1,
		AlertChannelId:        "123",
		InhibitedAlerts:       []string{inhibitedAlertName},
	}

	guildConfig := &GuildConfig{
		GuildId:       guildId,
		ScrapeConfigs: []ScrapeConfig{scrapeConfig},
	}
	err := mongoRepo.SetGuildConfig(ctx, guildConfig)
	if err != nil {
		t.Fail()
	}

	// Act
	scrapeConfig.InhibitedAlerts = slices.RemoveMatches(scrapeConfig.InhibitedAlerts, func(alertName string) bool {
		return alertName == inhibitedAlertName
	})

	newEndpoint := "http://localhost:5431"
	scrapeConfig.Endpoint = newEndpoint
	guildConfig.ScrapeConfigs[0] = scrapeConfig

	err = mongoRepo.SetGuildConfig(ctx, guildConfig)
	if err != nil {
		t.Fail()
	}

	// Assert
	db := mongoClient.Database(databaseName)
	coll := db.Collection("guild_config")
	res := coll.FindOne(ctx, bson.D{{"guild_id", guildId}})

	var foundGuildConfig *GuildConfig
	if err = res.Decode(&foundGuildConfig); err != nil {
		t.Fail()
	}

	if foundGuildConfig.GuildId != guildId {
		t.Fail()
	}

	if !slices.HasMatching(foundGuildConfig.ScrapeConfigs, func(config ScrapeConfig) bool {
		return config.Name == scrapeConfigName && config.Endpoint == newEndpoint && len(config.InhibitedAlerts) == 0
	}) {
		t.Fail()
	}
}

func TestGetGuildConfig(t *testing.T) {
	// Arrange
	ctx := context.Background()
	guildId := "foo"

	guildConfig := &GuildConfig{
		GuildId:       guildId,
		ScrapeConfigs: []ScrapeConfig{},
	}

	// Act
	err := mongoRepo.SetGuildConfig(ctx, guildConfig)
	if err != nil {
		t.Fail()
	}

	// Assert
	foundGuildConfig, err := mongoRepo.GetGuildConfig(ctx, guildId)
	if err != nil {
		t.Fail()
	}

	if foundGuildConfig.GuildId != guildId {
		t.Fail()
	}

	if len(foundGuildConfig.ScrapeConfigs) != 0 {
		t.Fail()
	}
}

func TestGetGuildConfigs(t *testing.T) {
	// Arrange
	ctx := context.Background()
	guildId1 := "foo"
	guildId2 := "bar"

	guildConfig1 := &GuildConfig{
		GuildId:       guildId1,
		ScrapeConfigs: []ScrapeConfig{},
	}

	guildConfig2 := &GuildConfig{
		GuildId:       guildId2,
		ScrapeConfigs: []ScrapeConfig{},
	}

	// Act
	err := mongoRepo.SetGuildConfig(ctx, guildConfig1)
	if err != nil {
		t.Fail()
	}

	err = mongoRepo.SetGuildConfig(ctx, guildConfig2)
	if err != nil {
		t.Fail()
	}

	// Assert
	foundGuildConfigs, err := mongoRepo.GetGuildConfigs(ctx)
	if err != nil {
		t.Fail()
	}

	if len(foundGuildConfigs) != 2 {
		t.Fail()
	}

	if !slices.HasMatching(foundGuildConfigs, func(config GuildConfig) bool {
		return config.GuildId == guildId1
	}) {
		t.Fail()
	}

	if !slices.HasMatching(foundGuildConfigs, func(config GuildConfig) bool {
		return config.GuildId == guildId2
	}) {
		t.Fail()
	}
}

func TestClearGuildInfo(t *testing.T) {
	// Arrange
	ctx := context.Background()
	guildId := "foo"
	commandId := "bar"
	commandName := "baz"

	guildConfig := &GuildConfig{
		GuildId:       guildId,
		ScrapeConfigs: []ScrapeConfig{},
	}

	err := mongoRepo.SetGuildConfig(ctx, guildConfig)
	if err != nil {
		t.Fail()
	}

	err = mongoRepo.RegisterCommand(ctx, guildId, commandId, commandName)
	if err != nil {
		t.Fail()
	}

	// Act
	err = mongoRepo.ClearGuildInfo(ctx, guildId)
	if err != nil {
		t.Fail()
	}

	// Assert
	db := mongoClient.Database(databaseName)
	commandsColl := db.Collection("command_registrations")
	commandsCount, err := commandsColl.CountDocuments(ctx, bson.D{{"guild_id", guildId}})
	if commandsCount != 0 {
		t.Fail()
	}

	configColl := db.Collection("guild_config")
	configCount, err := configColl.CountDocuments(ctx, bson.D{{"guild_id", guildId}})
	if configCount != 0 {
		t.Fail()
	}
}
