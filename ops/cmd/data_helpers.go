package main

import (
	"database/sql"

	dataaction "flashsale/ops/backend/actions/data"
)

func openDB(database string) (*sql.DB, func(), error) {
	return dataaction.OpenDB(database)
}
