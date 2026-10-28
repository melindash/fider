package dbx_test

import (
	"context"
	"testing"

	. "github.com/getfider/fider/app/pkg/assert"
	"github.com/getfider/fider/app/pkg/dbx"
)

func TestMigrate_AppliesOlderUnrecordedVersion(t *testing.T) {
	setupMigrationTest(t)
	ctx := context.Background()

	err := dbx.Migrate(ctx, "/app/pkg/dbx/testdata/mageos_newest_first")
	Expect(err).IsNil()

	err = dbx.Migrate(ctx, "/app/pkg/dbx/testdata/mageos_out_of_order")
	Expect(err).IsNil()

	trx, _ := dbx.BeginTx(ctx)
	defer trx.MustRollback()
	var value string
	err = trx.Scalar(&value, "SELECT description FROM dummy WHERE id = 100")
	Expect(err).IsNil()
	Expect(value).Equals("Applied out of order")
}
