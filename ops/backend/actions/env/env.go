package envaction

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	kafkaprobe "flashsale/ops/backend/probe/kafka"
	mysqlprobe "flashsale/ops/backend/probe/mysql"
	redisprobe "flashsale/ops/backend/probe/redis"
	"flashsale/ops/backend/shared/devenv"
	"flashsale/ops/backend/shared/platform"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

const (
	DefaultAppComposeFile = "deploy/compose/docker-compose.app.yml"
	DefaultOpsComposeFile = "deploy/compose/docker-compose.ops.yml"
)

type ComposeOptions struct {
	RepoRoot      string
	EnvFile       string
	ComposeFile   string
	Observability bool
}

type DownOptions struct {
	RepoRoot      string
	EnvFile       string
	ComposeFile   string
	RemoveVolumes bool
}

type OpsComposeOptions struct {
	RepoRoot    string
	ComposeFile string
}

type StartOptions struct {
	RepoRoot      string
	EnvFile       string
	ComposeFile   string
	Observability bool
	SkipMigrate   bool
	SkipSmoke     bool
}

type RestartOptions struct {
	RepoRoot      string
	EnvFile       string
	ComposeFile   string
	RemoveVolumes bool
	Observability bool
	SkipMigrate   bool
	SkipSmoke     bool
}

type MigrationOptions struct {
	RepoRoot string
	EnvFile  string
	Steps    int
	All      bool
}

type SmokeOptions struct {
	RepoRoot   string
	EnvFile    string
	ConfigPath string
}

func Up(opts ComposeOptions) error {
	repoRoot, err := resolveRepoRoot(opts.RepoRoot)
	if err != nil {
		return err
	}
	envPath := devenv.ResolvePath(repoRoot, opts.EnvFile)
	composePath := devenv.ResolvePath(repoRoot, opts.ComposeFile)
	if err := devenv.Load(envPath); err != nil {
		return err
	}
	if err := platform.MustExecutable("docker"); err != nil {
		return err
	}

	cmdArgs := []string{"compose", "--env-file", envPath, "-f", composePath}
	if opts.Observability {
		cmdArgs = append(cmdArgs, "--profile", "observability")
	}
	cmdArgs = append(cmdArgs, "up", "-d")

	fmt.Println("[env.up] docker compose up -d")
	return platform.Run(context.Background(), repoRoot, "docker", cmdArgs...)
}

func Down(opts DownOptions) error {
	repoRoot, err := resolveRepoRoot(opts.RepoRoot)
	if err != nil {
		return err
	}
	envPath := devenv.ResolvePath(repoRoot, opts.EnvFile)
	composePath := devenv.ResolvePath(repoRoot, opts.ComposeFile)
	if err := devenv.Load(envPath); err != nil {
		return err
	}
	if err := platform.MustExecutable("docker"); err != nil {
		return err
	}

	cmdArgs := []string{"compose", "--env-file", envPath, "-f", composePath, "down"}
	if opts.RemoveVolumes {
		cmdArgs = append(cmdArgs, "-v")
	}
	fmt.Println("[env.down] docker compose down")
	return platform.Run(context.Background(), repoRoot, "docker", cmdArgs...)
}

func OpsUp(opts OpsComposeOptions) error {
	repoRoot, err := resolveRepoRoot(opts.RepoRoot)
	if err != nil {
		return err
	}
	composePath := devenv.ResolvePath(repoRoot, opts.ComposeFile)
	if err := platform.MustExecutable("docker"); err != nil {
		return err
	}
	return platform.Run(context.Background(), repoRoot, "docker", "compose", "-f", composePath, "up", "-d")
}

func OpsDown(opts OpsComposeOptions) error {
	repoRoot, err := resolveRepoRoot(opts.RepoRoot)
	if err != nil {
		return err
	}
	composePath := devenv.ResolvePath(repoRoot, opts.ComposeFile)
	if err := platform.MustExecutable("docker"); err != nil {
		return err
	}
	return platform.Run(context.Background(), repoRoot, "docker", "compose", "-f", composePath, "down")
}

func Start(opts StartOptions) error {
	if err := Up(ComposeOptions{
		RepoRoot:      opts.RepoRoot,
		EnvFile:       opts.EnvFile,
		ComposeFile:   opts.ComposeFile,
		Observability: opts.Observability,
	}); err != nil {
		return err
	}
	if !opts.SkipMigrate {
		if err := MigrateUp(MigrationOptions{RepoRoot: opts.RepoRoot, EnvFile: opts.EnvFile, Steps: 1}); err != nil {
			return err
		}
	}
	if !opts.SkipSmoke {
		if err := Smoke(SmokeOptions{RepoRoot: opts.RepoRoot, EnvFile: opts.EnvFile, ConfigPath: "configs/dev.yaml"}); err != nil {
			return err
		}
	}
	return nil
}

func Restart(opts RestartOptions) error {
	if err := Down(DownOptions{
		RepoRoot:      opts.RepoRoot,
		EnvFile:       opts.EnvFile,
		ComposeFile:   opts.ComposeFile,
		RemoveVolumes: opts.RemoveVolumes,
	}); err != nil {
		return err
	}
	return Start(StartOptions{
		RepoRoot:      opts.RepoRoot,
		EnvFile:       opts.EnvFile,
		ComposeFile:   opts.ComposeFile,
		Observability: opts.Observability,
		SkipMigrate:   opts.SkipMigrate,
		SkipSmoke:     opts.SkipSmoke,
	})
}

func MigrateUp(opts MigrationOptions) error {
	repoRoot, err := resolveRepoRoot(opts.RepoRoot)
	if err != nil {
		return err
	}
	if strings.TrimSpace(opts.EnvFile) != "-" {
		if err := devenv.Load(devenv.ResolvePath(repoRoot, opts.EnvFile)); err != nil {
			return err
		}
	}
	return applyMigrations(repoRoot, true, 1, false)
}

func MigrateDown(opts MigrationOptions) error {
	if !opts.All && opts.Steps < 1 {
		return fmt.Errorf("steps must be >= 1")
	}
	repoRoot, err := resolveRepoRoot(opts.RepoRoot)
	if err != nil {
		return err
	}
	if strings.TrimSpace(opts.EnvFile) != "-" {
		if err := devenv.Load(devenv.ResolvePath(repoRoot, opts.EnvFile)); err != nil {
			return err
		}
	}
	return applyMigrations(repoRoot, false, opts.Steps, opts.All)
}

func Smoke(opts SmokeOptions) error {
	repoRoot, err := resolveRepoRoot(opts.RepoRoot)
	if err != nil {
		return err
	}
	if strings.TrimSpace(opts.EnvFile) != "-" {
		if err := devenv.Load(devenv.ResolvePath(repoRoot, opts.EnvFile)); err != nil {
			return err
		}
	}
	if err := ensureSmokeEnv(); err != nil {
		return err
	}
	cfg := devenv.ResolvePath(repoRoot, opts.ConfigPath)
	checks := []struct {
		name string
		run  func(string) error
	}{
		{name: "mysql", run: func(path string) error { return mysqlprobe.Run(path, 5*time.Second) }},
		{name: "redis", run: func(path string) error { return redisprobe.Run(path, 5*time.Second) }},
		{name: "kafka", run: func(path string) error { return kafkaprobe.Run(path, 20*time.Second) }},
	}
	for _, check := range checks {
		fmt.Printf("[env.smoke] %s\n", check.name)
		if err := check.run(cfg); err != nil {
			return err
		}
	}
	fmt.Println("[env.smoke] all checks passed")
	return nil
}

func applyMigrations(repoRoot string, up bool, steps int, all bool) error {
	host, port, user, pass, err := devenv.MySQLConfig()
	if err != nil {
		return err
	}
	dbs := []string{"user", "admin", "product", "order", "seckill"}
	for _, db := range dbs {
		name := "flash_" + db
		sourceURL := "file://" + filepath.ToSlash(filepath.Join(repoRoot, "deploy", "migrations", db))
		dsn := buildMigrateDSN(host, port, user, pass, name)
		m, err := migrate.New(sourceURL, dsn)
		if err != nil {
			return fmt.Errorf("init migrate for %s: %w", name, err)
		}
		fmt.Printf("[env.migrate] %s -> %s\n", map[bool]string{true: "up", false: "down"}[up], name)
		if up {
			err = m.Up()
		} else if all {
			err = m.Down()
		} else {
			err = m.Steps(-steps)
		}
		if err != nil && !errors.Is(err, migrate.ErrNoChange) {
			_, _ = m.Close()
			return fmt.Errorf("migrate failed for %s: %w", name, err)
		}
		_, _ = m.Close()
	}
	return nil
}

func buildMigrateDSN(host string, port int, user, pass, db string) string {
	return fmt.Sprintf(
		"mysql://%s:%s@tcp(%s:%d)/%s?multiStatements=true",
		url.QueryEscape(user),
		url.QueryEscape(pass),
		host,
		port,
		db,
	)
}

func ensureSmokeEnv() error {
	host, port, user, pass, err := devenv.MySQLConfig()
	if err != nil {
		return err
	}
	_ = os.Setenv("FLASHSALE_MYSQL_HOST", host)
	_ = os.Setenv("FLASHSALE_MYSQL_PORT", strconv.Itoa(port))
	_ = os.Setenv("FLASHSALE_MYSQL_USER", user)
	_ = os.Setenv("FLASHSALE_MYSQL_PASSWORD", pass)
	if strings.TrimSpace(os.Getenv("FLASHSALE_REDIS_ADDR")) == "" {
		if redisPort := strings.TrimSpace(os.Getenv("FLASH_REDIS_PORT")); redisPort != "" {
			_ = os.Setenv("FLASHSALE_REDIS_ADDR", "127.0.0.1:"+redisPort)
		}
	}
	if strings.TrimSpace(os.Getenv("FLASHSALE_KAFKA_BROKERS")) == "" {
		if kafkaPort := strings.TrimSpace(os.Getenv("FLASH_KAFKA_PORT")); kafkaPort != "" {
			_ = os.Setenv("FLASHSALE_KAFKA_BROKERS", "127.0.0.1:"+kafkaPort)
		}
	}
	return nil
}

func resolveRepoRoot(repoRoot string) (string, error) {
	if strings.TrimSpace(repoRoot) == "" {
		return filepath.Abs(".")
	}
	return filepath.Abs(repoRoot)
}
