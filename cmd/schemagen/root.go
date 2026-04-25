package main

import (
	"fmt"
	"strings"

	"github.com/nurularifin27/schemagen/dbtype"
	"github.com/nurularifin27/schemagen/entitygen"

	"github.com/spf13/cobra"
)

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "schemagen",
		Short:         "Generate Go entities from database schema",
		Long:          "Schemagen introspects a database schema and syncs Go entity files while preserving manual code outside managed markers.",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		Example: strings.TrimSpace(`
  schemagen init
  schemagen generate --config schemagen.yaml
  schemagen --dsn "postgres://user:pass@localhost:5432/app?sslmode=disable" --driver postgres
  schemagen generate --config schemagen.yaml --tables users,companies --on-conflict=backup
  schemagen completion zsh
		`),
	}

	generateCmd := newGenerateCmd()
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return generateCmd.RunE(generateCmd, args)
	}
	cmd.Flags().AddFlagSet(generateCmd.Flags())

	cmd.AddCommand(generateCmd)
	cmd.AddCommand(newInitCmd())

	cmd.InitDefaultCompletionCmd()
	cmd.CompletionOptions.DisableDefaultCmd = false

	return cmd
}

func newGenerateCmd() *cobra.Command {
	cfg := Config{}

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate Go entities from database schema",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenerate(cmd, &cfg)
		},
	}

	bindGenerateFlags(cmd, &cfg)
	return cmd
}

func bindGenerateFlags(cmd *cobra.Command, cfg *Config) {
	flags := cmd.Flags()
	flags.String("config", defaultConfig, "Path to YAML config")
	flags.String("relations-config", defaultRelationsConfig, "Path to relations YAML config")
	flags.StringVar(&cfg.DSN, "dsn", "", "Database DSN")
	flags.StringVar(&cfg.Driver, "driver", "", "Database driver: postgres, mysql, mariadb, sqlite")
	flags.StringVar(&cfg.Renderer, "renderer", "", "Output renderer: plain, sqlx, gorm")
	flags.StringVar(&cfg.OutDir, "out-dir", "", "Output directory for generated entities")
	flags.StringSliceVar(&cfg.Tables, "tables", nil, "Tables to include")
	flags.StringSliceVar(&cfg.Exclude, "exclude", nil, "Tables to exclude")
	flags.StringVar(&cfg.OnConflict, "on-conflict", "", "Conflict policy for unmanaged files: skip, error, backup, overwrite")
	flags.StringVar(&cfg.DecimalStrategy, "decimal-strategy", "", "Decimal mapping strategy: float64, string")
	flags.StringVar(&cfg.JSONStrategy, "json-strategy", "", "JSON mapping strategy: bytes, rawmessage")
	flags.StringVar(&cfg.NullableStrategy, "nullable-strategy", "", "Nullable mapping strategy: pointer, sqlnull")
}

func runGenerate(cmd *cobra.Command, cfg *Config) error {
	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return err
	}
	relationsPath, err := cmd.Flags().GetString("relations-config")
	if err != nil {
		return err
	}

	fileCfg := loadConfigIfExists(configPath)
	merged := mergeConfig(fileCfg, *cfg)
	normalizeConfig(&merged)
	relationsCfg := loadRelationsIfExists(relationsPath)
	normalizeRelationsConfig(&relationsCfg)

	if merged.DSN == "" {
		return fmt.Errorf("dsn is required; provide --dsn or set it in schemagen.yaml")
	}
	if !isValidConflictPolicy(merged.OnConflict) {
		return fmt.Errorf("invalid on_conflict %q (supported: skip, error, backup, overwrite)", merged.OnConflict)
	}
	if !isValidRenderer(merged.Renderer) {
		return fmt.Errorf("invalid renderer %q (supported: plain, sqlx, gorm)", merged.Renderer)
	}
	if !isValidDecimalStrategy(merged.DecimalStrategy) {
		return fmt.Errorf("invalid decimal_strategy %q (supported: float64, string)", merged.DecimalStrategy)
	}
	if !isValidJSONStrategy(merged.JSONStrategy) {
		return fmt.Errorf("invalid json_strategy %q (supported: bytes, rawmessage)", merged.JSONStrategy)
	}
	if !isValidNullableStrategy(merged.NullableStrategy) {
		return fmt.Errorf("invalid nullable_strategy %q (supported: pointer, sqlnull)", merged.NullableStrategy)
	}
	merged.TypeOverrides = normalizeTypeOverrides(merged.TypeOverrides)
	if err := validateTypeOverrides(merged.TypeOverrides); err != nil {
		return err
	}
	if err := validateRelationsConfig(relationsCfg); err != nil {
		return err
	}

	db := connectDB(merged.Driver, merged.DSN)
	return entitygen.Generate(db, entitygen.Options{
		Driver:           merged.Driver,
		Renderer:         merged.Renderer,
		OutDir:           merged.OutDir,
		Tables:           merged.Tables,
		ExcludeTables:    merged.Exclude,
		OnConflict:       merged.OnConflict,
		DecimalStrategy:  merged.DecimalStrategy,
		JSONStrategy:     merged.JSONStrategy,
		NullableStrategy: merged.NullableStrategy,
		TypeOverrides:    toDBTypeOverrides(merged.TypeOverrides),
		Relations:        toEntityRelations(relationsCfg),
	})
}

func newInitCmd() *cobra.Command {
	var (
		path          string
		relationsPath string
		force         bool
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Write default schemagen config files",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := writeDefaultConfig(path, force); err != nil {
				return err
			}
			if err := writeDefaultRelationsConfig(relationsPath, force); err != nil {
				return err
			}

			outPath := path
			if strings.TrimSpace(outPath) == "" {
				outPath = defaultConfig
			}
			outRelationsPath := relationsPath
			if strings.TrimSpace(outRelationsPath) == "" {
				outRelationsPath = defaultRelationsConfig
			}

			_, err := fmt.Fprintf(cmd.OutOrStdout(), "wrote %s\nwrote %s\n", outPath, outRelationsPath)
			if err != nil {
				return err
			}
			return nil
		},
		Example: strings.TrimSpace(`
  schemagen init
  schemagen init --path config/schemagen.yaml
  schemagen init --relations-path config/schemagen.relations.yaml
  schemagen init --force
		`),
	}

	cmd.Flags().StringVar(&path, "path", defaultConfig, "Path to generated YAML config")
	cmd.Flags().StringVar(&relationsPath, "relations-path", defaultRelationsConfig, "Path to generated relations YAML config")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite config file if it already exists")

	return cmd
}

func mergeConfig(base, override Config) Config {
	cfg := base
	if override.DSN != "" {
		cfg.DSN = override.DSN
	}
	if override.Driver != "" {
		cfg.Driver = override.Driver
	}
	if override.Renderer != "" {
		cfg.Renderer = override.Renderer
	}
	if override.OutDir != "" {
		cfg.OutDir = override.OutDir
	}
	if len(override.Tables) > 0 {
		cfg.Tables = override.Tables
	}
	if len(override.Exclude) > 0 {
		cfg.Exclude = override.Exclude
	}
	if override.OnConflict != "" {
		cfg.OnConflict = override.OnConflict
	}
	if override.DecimalStrategy != "" {
		cfg.DecimalStrategy = override.DecimalStrategy
	}
	if override.JSONStrategy != "" {
		cfg.JSONStrategy = override.JSONStrategy
	}
	if override.NullableStrategy != "" {
		cfg.NullableStrategy = override.NullableStrategy
	}
	if len(override.TypeOverrides) > 0 {
		cfg.TypeOverrides = override.TypeOverrides
	}
	return cfg
}

func toDBTypeOverrides(overrides []TypeOverrideConfig) []dbtype.Override {
	if len(overrides) == 0 {
		return nil
	}

	result := make([]dbtype.Override, 0, len(overrides))
	for _, override := range overrides {
		result = append(result, dbtype.Override{
			Table:   override.Table,
			Column:  override.Column,
			DBType:  override.DBType,
			GoType:  override.GoType,
			Imports: override.Imports,
		})
	}
	return result
}
