// Copyright 2014 beego Author. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package orm

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/opentracing/opentracing-go"
)

// database query logger struct.
// if dev mode, use dbQueryLog, or use dbQuerier.
type dbQueryTracable struct {
	dbQueryLog
	span    opentracing.Span
	comment string
}

var _ dbQuerier = new(dbQueryTracable)
var _ txer = new(dbQueryTracable)
var _ txEnder = new(dbQueryTracable)

func (d *dbQueryTracable) CreateSpan(query string) opentracing.Span {
	if d.span == nil {
		return nil
	}

	items := strings.Split(query, " ")
	operationName := fmt.Sprintf("db-%s", items[0])
	span := d.span.Tracer().StartSpan(
		operationName,
		opentracing.ChildOf(d.span.Context()),
	)
	span.LogKV("sql", query)
	//span.SetTag("db.statement", query)

	return span
}

func (d *dbQueryTracable) SetComment(comment string) {
	d.comment = comment
}

func (d *dbQueryTracable) Prepare(query string) (*sql.Stmt, error) {
	span := d.CreateSpan(query)
	if span != nil {
		defer span.Finish()
	}

	var a time.Time
	if Debug {
		a = time.Now()
	}
	stmt, err := d.db.Prepare(query)
	if Debug {
		debugLogQueies(d.alias, "[orm_tracable] db.Prepare", query, a, err)
	}
	return stmt, err
}

func (d *dbQueryTracable) Exec(query string, args ...interface{}) (sql.Result, error) {
	span := d.CreateSpan(query)
	if span != nil {
		defer span.Finish()
	}

	var a time.Time
	if Debug {
		a = time.Now()
	}
	res, err := d.db.Exec(query, args...)
	if Debug {
		debugLogQueies(d.alias, "[orm_tracable] db.Exec", query, a, err, args...)
	}
	return res, err
}

func (d *dbQueryTracable) Query(query string, args ...interface{}) (*sql.Rows, error) {
	span := d.CreateSpan(query)
	if span != nil {
		defer span.Finish()
	}

	var a time.Time
	if Debug {
		a = time.Now()
	}
	res, err := d.db.Query(query, args...)
	if Debug {
		debugLogQueies(d.alias, "[orm_tracable] db.Query", query, a, err, args...)
	}
	return res, err
}

func (d *dbQueryTracable) QueryRow(query string, args ...interface{}) *sql.Row {
	span := d.CreateSpan(query)
	if span != nil {
		defer span.Finish()
	}

	var a time.Time
	if Debug {
		a = time.Now()
	}
	res := d.db.QueryRow(query, args...)
	if Debug {
		debugLogQueies(d.alias, "[orm_tracable] db.QueryRow", query, a, nil, args...)
	}
	return res
}

func (d *dbQueryTracable) Begin() (*sql.Tx, error) {
	span := d.CreateSpan("BEGIN")
	if span != nil {
		defer span.Finish()
	}

	var a time.Time
	if Debug {
		a = time.Now()
	}
	tx, err := d.db.(txer).Begin()
	if Debug {
		debugLogQueies(d.alias, "[orm_tracable] db.Begin", "START TRANSACTION", a, err)
	}
	return tx, err
}

func (d *dbQueryTracable) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	span := d.CreateSpan("BEGIN")
	if span != nil {
		defer span.Finish()
	}

	var a time.Time
	if Debug {
		a = time.Now()
	}
	tx, err := d.db.(txer).BeginTx(ctx, opts)
	if Debug {
		debugLogQueies(d.alias, "[orm_tracable] db.BeginTx", "START TRANSACTION", a, err)
	}
	return tx, err
}

func (d *dbQueryTracable) Commit() error {
	span := d.CreateSpan("COMMIT")
	if span != nil {
		defer span.Finish()
	}

	var a time.Time
	if Debug {
		a = time.Now()
	}
	err := d.db.(txEnder).Commit()
	if Debug {
		debugLogQueies(d.alias, "[orm_tracable] tx.Commit", "COMMIT", a, err)
	}
	return err
}

func (d *dbQueryTracable) Rollback() error {
	span := d.CreateSpan("ROLLBACK")
	if span != nil {
		defer span.Finish()
	}

	var a time.Time
	if Debug {
		a = time.Now()
	}
	err := d.db.(txEnder).Rollback()
	if Debug {
		debugLogQueies(d.alias, "[orm_tracable] tx.Rollback", "ROLLBACK", a, err)
	}
	return err
}

func (d *dbQueryTracable) SetDB(db dbQuerier) {
	d.db = db
}

func newDbQueryTracable(alias *alias, db dbQuerier, span opentracing.Span) dbQuerier {
	d := new(dbQueryTracable)
	d.alias = alias
	d.db = db
	d.span = span
	return d
}
