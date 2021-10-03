package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/blainemoser/GoCLIInterface/arguments"
	"github.com/blainemoser/MySqlDB/database"
	"github.com/blainemoser/MySqlMigrate/migrate"
)

type Args struct {
	Inputs map[string][]string
	Path   string
}

// Expected is the expected argument names and aliases
func (a Args) Expected() map[string]string {
	return map[string]string{
		"app-name": "app-name",
		"a":        "app-name",
		"migrate":  "migrate",
		"m":        "migrate",
		"create":   "create",
		"c":        "create",
	}
}

func (a *Args) SetInputs(inputs map[string][]string) {
	a.Inputs = inputs
	if err := a.checkInputs(); err != nil {
		log.Fatal(err)
	}
	if err := a.SetPath(); err != nil {
		log.Fatal(err)
	}
}

func (a *Args) checkInputs() error {
	if a.Inputs["app-name"] == nil || len(a.Inputs["app-name"]) < 1 {
		return fmt.Errorf("app-name (a) argument not provided")
	}
	if a.Inputs["create"] != nil && a.Inputs["migrate"] != nil {
		return fmt.Errorf("cannot specify both create and migrate")
	}
	if a.Inputs["migrate"] != nil && a.Inputs["migrate"][0] != "up" && a.Inputs["migrate"][0] != "down" {
		return fmt.Errorf("specify either 'up' or 'down' for migrate")
	}
	if a.Inputs["create"] == nil && a.Inputs["migrate"] == nil {
		return fmt.Errorf("please select one of the following: create, migrate up or migrate down")
	}

	return nil
}

func (a *Args) GetAppName() string {
	return a.Inputs["app-name"][0]
}

func (a *Args) SetPath() error {
	// Note that this assumes that this script is located in the base directory
	// shared by all apps which make use of this programme to manage their migrations.
	// It's also assumed that each app contains a folder called `migrations` to store migration files
	// One can use any method to find a directory or path to use, or just use a hard-coded
	// path if that is more applicable
	appName := a.GetAppName()
	dir, err := getBaseDir()
	if err != nil {
		return err
	}
	a.Path = fmt.Sprintf("%s/%s/migrations", dir, appName)
	return nil
}

func (a *Args) GetAction() (string, error) {
	if a.Inputs["create"] != nil && len(a.Inputs["create"]) > 0 {
		return "create", nil
	}

	if a.Inputs["migrate"] != nil {
		if a.Inputs["migrate"][0] == "up" {
			return "migrate-up", nil
		} else {
			return "migrate-down", nil
		}
	}
	return "", fmt.Errorf("no action could be found")
}

func main() {
	args := os.Args[1:]
	a := &Args{}
	if err := arguments.Inputs(a, args); err != nil {
		log.Fatal(err)
	}
	if err := handle(a); err != nil {
		log.Fatal(err)
	}
}

func connect(a *Args) (*database.Database, error) {
	// These connections settings could - and probably should
	// be sourced from environment variables or configuration files
	db, err := database.MakeSchemaless(&database.Configs{
		Host:     "127.0.0.1",
		Port:     "3306",
		Username: "root",
		Password: "secret",
		// here it's expected that the database name is the same
		// as the app name. if this is not the case, one would need to
		// come up with some mapping rules
		Driver: "mysql",
	})
	if err != nil {
		return nil, err
	}
	return bootSchema(a, &db)
}

func bootSchema(a *Args, db *database.Database) (*database.Database, error) {
	// This will create a new schema if none is found
	existsQ := fmt.Sprintf(
		"SELECT schema_name FROM information_schema.schemata WHERE schema_name = '%s'",
		a.GetAppName(),
	)
	schemaExists, err := db.QueryRaw(existsQ, nil)
	if err != nil {
		return nil, err
	}
	if len(schemaExists) < 1 {
		_, err = db.Exec(fmt.Sprintf("create schema %s", a.GetAppName()), nil)
		if err != nil {
			return nil, err
		}
	}
	db.SetSchema(a.GetAppName())
	return db, nil
}

func handle(a *Args) error {
	db, err := connect(a)
	if err != nil {
		return err
	}
	action, err := a.GetAction()
	if err != nil {
		return err
	}
	switch action {
	case "create":
		return create(a, db)
	case "migrate-up":
		return migrateUp(a, db)
	case "migrate-down":
		return migrateDown(a, db)
	}
	return nil
}

func create(a *Args, db *database.Database) error {
	_, err := migrate.Make(db, a.Path).Create(a.Inputs["create"][0])
	if err != nil {
		return err
	}
	return nil
}

func migrateUp(a *Args, db *database.Database) error {
	err := migrate.Make(db, a.Path).MigrateUp()
	if err != nil {
		return err
	}
	return nil
}

func migrateDown(a *Args, db *database.Database) error {
	err := migrate.Make(db, a.Path).MigrateDown()
	if err != nil {
		return err
	}
	return nil
}

func getBaseDir() (string, error) {
	var dir string
	var err error
	if dir, err = os.Getwd(); err != nil {
		return "", err
	}
	dirS := strings.Split(dir, "/")
	if len(dirS) > 1 {
		dirS = dirS[:len(dirS)-1]
	}
	return strings.Join(dirS, "/"), nil
}
