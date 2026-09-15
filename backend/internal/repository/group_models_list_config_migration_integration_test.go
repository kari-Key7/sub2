//go:build integration

package repository

import (
	"context"
	"database/sql"
	"net/url"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	dbmigrations "github.com/Wei-Shaw/sub2api/migrations"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

// 独立测试库仅复现分组结构与迁移账本，避免对集成测试共享表执行删列操作。
func newGroupModelsMigrationDB(t *testing.T) *sql.DB {
	t.Helper()
	ctx := context.Background()
	name := "migration238_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	_, err := integrationDB.ExecContext(ctx, "CREATE DATABASE "+pq.QuoteIdentifier(name))
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := integrationDB.ExecContext(context.Background(), "DROP DATABASE "+pq.QuoteIdentifier(name))
		require.NoError(t, err)
	})
	dsn, err := url.Parse(integrationPostgresDSN)
	require.NoError(t, err)
	dsn.Path = "/" + name
	db, err := sql.Open("postgres", dsn.String())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, db.PingContext(ctx))
	return db
}

func TestMigration238RestoresGroupModelsListConfigWithoutChangingAllowlist(t *testing.T) {
	ctx := context.Background()
	const oldName = "143_group_models_list_config.sql"
	const repairName = "238_restore_group_models_list_config.sql"
	oldSQL, err := dbmigrations.FS.ReadFile(oldName)
	require.NoError(t, err)
	repairSQL, err := dbmigrations.FS.ReadFile(repairName)
	require.NoError(t, err)
	oldFS := fstest.MapFS{oldName: &fstest.MapFile{Data: oldSQL}}
	repairFS := fstest.MapFS{
		oldName:    &fstest.MapFile{Data: oldSQL},
		repairName: &fstest.MapFile{Data: repairSQL},
	}

	for _, state := range []string{"fresh_database", "recorded_143_column_missing", "existing_custom_display"} {
		t.Run(state, func(t *testing.T) {
			db := newGroupModelsMigrationDB(t)
			_, err := db.ExecContext(ctx, `CREATE TABLE groups (id BIGINT PRIMARY KEY, name TEXT NOT NULL)`)
			require.NoError(t, err)
			// 同时覆盖旧版没有 model_allowlist 与较新版保留多种 JSON 值的结构。
			if state != "fresh_database" {
				_, err = db.ExecContext(ctx, `ALTER TABLE groups ADD COLUMN model_allowlist JSONB`)
				require.NoError(t, err)
				require.NoError(t, applyMigrationsFS(ctx, db, oldFS))
				_, err = db.ExecContext(ctx, `
INSERT INTO groups (id, name, model_allowlist, models_list_config) VALUES
    (1, 'configured', '{"enabled":false,"models":["sample-model"]}', '{"enabled":true,"models":["display-model"]}'),
    (2, 'empty', '{}', '{}'),
    (3, 'null', NULL, '{"enabled":false,"models":["disabled-model"]}')`)
				require.NoError(t, err)
				if state == "recorded_143_column_missing" {
					// 复现事故：143 已执行并记账，后续版本删除该列；不伪造或改写旧记录。
					_, err = db.ExecContext(ctx, `ALTER TABLE groups DROP COLUMN models_list_config`)
					require.NoError(t, err)
					var unavailable string
					err = db.QueryRowContext(ctx, `SELECT models_list_config::text FROM groups LIMIT 1`).Scan(&unavailable)
					require.ErrorContains(t, err, "models_list_config")
				}
			} else {
				_, err = db.ExecContext(ctx, `INSERT INTO groups (id, name) VALUES (1, 'fresh')`)
				require.NoError(t, err)
			}

			var oldChecksum string
			var oldAppliedAt time.Time
			if state != "fresh_database" {
				require.NoError(t, db.QueryRowContext(ctx, `SELECT checksum, applied_at FROM schema_migrations WHERE filename = $1`, oldName).Scan(&oldChecksum, &oldAppliedAt))
			}
			// 对比整行而非只看行数，验证原始分组字段和较新 allowlist 均未被覆盖。
			var before, after string
			const preservedRowsSQL = `SELECT jsonb_agg(to_jsonb(g) - 'models_list_config' ORDER BY id)::text FROM groups g`
			require.NoError(t, db.QueryRowContext(ctx, preservedRowsSQL).Scan(&before))
			var originalDisplay string
			if state == "existing_custom_display" {
				require.NoError(t, db.QueryRowContext(ctx, `SELECT jsonb_agg(models_list_config ORDER BY id)::text FROM groups`).Scan(&originalDisplay))
			}

			// 调用真实 runner，证明已记录的 143 会跳过，而新文件名 238 仍会执行。
			require.NoError(t, applyMigrationsFS(ctx, db, repairFS))
			require.NoError(t, db.QueryRowContext(ctx, preservedRowsSQL).Scan(&after))
			require.Equal(t, before, after)
			if state == "existing_custom_display" {
				var display string
				require.NoError(t, db.QueryRowContext(ctx, `SELECT jsonb_agg(models_list_config ORDER BY id)::text FROM groups`).Scan(&display))
				require.JSONEq(t, originalDisplay, display)
			} else {
				var allDefaults bool
				require.NoError(t, db.QueryRowContext(ctx, `SELECT bool_and(models_list_config = '{}'::jsonb) FROM groups`).Scan(&allDefaults))
				require.True(t, allDefaults)
			}
			if state != "fresh_database" {
				var checksum string
				var appliedAt time.Time
				require.NoError(t, db.QueryRowContext(ctx, `SELECT checksum, applied_at FROM schema_migrations WHERE filename = $1`, oldName).Scan(&checksum, &appliedAt))
				require.Equal(t, oldChecksum, checksum)
				require.Equal(t, oldAppliedAt, appliedAt)
			}

			var columnType, nullable, columnDefault string
			require.NoError(t, db.QueryRowContext(ctx, `SELECT data_type, is_nullable, column_default FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'groups' AND column_name = 'models_list_config'`).Scan(&columnType, &nullable, &columnDefault))
			require.Equal(t, "jsonb", columnType)
			require.Equal(t, "NO", nullable)
			require.Equal(t, "'{}'::jsonb", columnDefault)
			var newDisplay string
			require.NoError(t, db.QueryRowContext(ctx, `INSERT INTO groups (id, name) VALUES (4, 'future') RETURNING models_list_config::text`).Scan(&newDisplay))
			require.JSONEq(t, "{}", newDisplay)

			// runner 重启及 SQL 直接重复执行都必须幂等，并保留所有已写入的值。
			const allRowsSQL = `SELECT jsonb_agg(to_jsonb(g) ORDER BY id)::text FROM groups g`
			require.NoError(t, db.QueryRowContext(ctx, allRowsSQL).Scan(&before))
			require.NoError(t, applyMigrationsFS(ctx, db, repairFS))
			_, err = db.ExecContext(ctx, string(repairSQL))
			require.NoError(t, err)
			require.NoError(t, db.QueryRowContext(ctx, allRowsSQL).Scan(&after))
			require.Equal(t, before, after)
			var repairRecords int
			require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE filename = $1`, repairName).Scan(&repairRecords))
			require.Equal(t, 1, repairRecords)
		})
	}
}
