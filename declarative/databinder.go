// Copyright 2012 The Walk Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package declarative

import (
	"github.com/lxn/walk"
)

type DataBinder struct {
	AssignTo       **walk.DataBinder
	DataSource     interface{}
	ErrorPresenter ErrorPresenter
}

func (db DataBinder) create() (*walk.DataBinder, error) {
	if db.DataSource == nil {
		return nil, nil
	}

	b := walk.NewDataBinder()

	if db.ErrorPresenter != nil {
		ep, err := db.ErrorPresenter.Create()
		if err != nil {
			return nil, err
		}
		b.SetErrorPresenter(ep)
	}

	b.SetDataSource(db.DataSource)

	if db.AssignTo != nil {
		*db.AssignTo = b
	}

	return b, nil
}
