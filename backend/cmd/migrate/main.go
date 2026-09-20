package main

import (
	"embed"
	"flag"
	"fmt"
	"os"
	"strings"

	"backend/pkg/config"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func main() {
	var envConf = flag.String("conf", "config/deploy.yml", "config path, eg: -conf ./config/deploy.yml")
	var down = flag.Bool("down", false, "rollback one step")
	flag.Parse()

	conf := config.NewConfig(*envConf)
	dsn := conf.GetString("data.db.user.dsn")
	if i := strings.IndexByte(dsn, '?'); i >= 0 {
		dsn = dsn[:i]
	}
	url := "mysql://" + dsn + "?multiStatements=true"

	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		panic(err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, url)
	if err != nil {
		panic(err)
	}
	defer m.Close()

	if *down {
		if err := m.Steps(-1); err != nil && err != migrate.ErrNoChange {
			fmt.Println("migrate down error:", err)
			os.Exit(1)
		}
		fmt.Println("migrate down success")
		return
	}
	if err := m.Up(); err != nil {
		if err == migrate.ErrNoChange {
			fmt.Println("no change")
			return
		}
		fmt.Println("migrate up error:", err)
		os.Exit(1)
	}
	fmt.Println("migrate up success")
}
