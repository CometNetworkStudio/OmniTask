package migration

import (
	"embed"
	"fmt"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/spf13/viper"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Up 对 full 模式（MySQL）执行版本化迁移。
// lite 模式由 GORM AutoMigrate 处理（golang-migrate 的 sqlite 驱动需 CGO，故不用）。
func Up(conf *viper.Viper) error {
	if conf.GetString("mode") == "lite" {
		return nil
	}
	dsn := conf.GetString("data.db.user.dsn")
	if i := strings.IndexByte(dsn, '?'); i >= 0 {
		dsn = dsn[:i]
	}
	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return err
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, "mysql://"+dsn+"?multiStatements=true")
	if err != nil {
		return err
	}
	defer m.Close()
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}
