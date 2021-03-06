// Copyright 2013 The Walk Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package walk

import (
	"syscall"
	"unsafe"
)

import (
	. "github.com/lxn/go-winapi"
)

// TableViewColumn represents a column in a TableView.
type TableViewColumn struct {
	tv        *TableView
	index     int
	alignment Alignment1D
	format    string
	precision int
	title     string
	visible   bool
	width     int
}

// NewTableViewColumn returns a new TableViewColumn.
func NewTableViewColumn() *TableViewColumn {
	return &TableViewColumn{
		format:  "%v",
		index:   -1,
		visible: true,
		width:   50,
	}
}

// Alignment returns the alignment of the TableViewColumn.
func (tvc *TableViewColumn) Alignment() Alignment1D {
	return tvc.alignment
}

// SetAlignment sets the alignment of the TableViewColumn.
func (tvc *TableViewColumn) SetAlignment(alignment Alignment1D) (err error) {
	if alignment == tvc.alignment {
		return nil
	}

	old := tvc.alignment
	defer func() {
		if err != nil {
			tvc.alignment = old
		}
	}()

	tvc.alignment = alignment

	return tvc.update()
}

// Format returns the format string for converting a value into a string.
func (tvc *TableViewColumn) Format() string {
	return tvc.format
}

// SetFormat sets the format string for converting a value into a string.
func (tvc *TableViewColumn) SetFormat(format string) (err error) {
	if format == tvc.format {
		return nil
	}

	old := tvc.format
	defer func() {
		if err != nil {
			tvc.format = old
		}
	}()

	tvc.format = format

	if tvc.tv == nil {
		return nil
	}

	return tvc.tv.Invalidate()
}

// Precision returns the number of decimal places for formatting float32,
// float64 or big.Rat values.
func (tvc *TableViewColumn) Precision() int {
	return tvc.precision
}

// SetPrecision sets the number of decimal places for formatting float32,
// float64 or big.Rat values.
func (tvc *TableViewColumn) SetPrecision(precision int) (err error) {
	if precision == tvc.precision {
		return nil
	}

	old := tvc.precision
	defer func() {
		if err != nil {
			tvc.precision = old
		}
	}()

	tvc.precision = precision

	if tvc.tv == nil {
		return nil
	}

	return tvc.tv.Invalidate()
}

// Title returns the text to display in the column header.
func (tvc *TableViewColumn) Title() string {
	return tvc.title
}

// SetTitle sets the text to display in the column header.
func (tvc *TableViewColumn) SetTitle(title string) (err error) {
	if title == tvc.title {
		return nil
	}

	old := tvc.title
	defer func() {
		if err != nil {
			tvc.title = old
		}
	}()

	tvc.title = title

	return tvc.update()
}

/*// Visible returns if the column is visible.
func (tvc *TableViewColumn) Visible() bool {
	return tvc.visible
}

// SetVisible sets if the column is visible.
func (tvc *TableViewColumn) SetVisible(visible bool) (err error) {
	if visible == tvc.visible {
		return nil
	}

	old := tvc.visible
	defer func() {
		if err != nil {
			tvc.visible = old
		}
	}()

	tvc.visible = visible

	if tvc.tv == nil {
		return nil
	}

	if visible {
		return tvc.create()
	}

	return tvc.destroy()
}*/

// Width returns the width of the column in pixels.
func (tvc *TableViewColumn) Width() int {
	if tvc.tv == nil || !tvc.visible {
		return tvc.width
	}

	return int(tvc.tv.SendMessage(LVM_GETCOLUMNWIDTH, uintptr(tvc.index), 0))
}

// SetWidth sets the width of the column in pixels.
func (tvc *TableViewColumn) SetWidth(width int) (err error) {
	if width == tvc.Width() {
		return nil
	}

	old := tvc.width
	defer func() {
		if err != nil {
			tvc.width = old
		}
	}()

	tvc.width = width

	return tvc.update()
}

func (tvc *TableViewColumn) create() error {
	var lvc LVCOLUMN

	lvc.Mask = LVCF_FMT | LVCF_WIDTH | LVCF_TEXT | LVCF_SUBITEM
	lvc.ISubItem = int32(tvc.index)
	lvc.PszText = syscall.StringToUTF16Ptr(tvc.title)
	if tvc.width > 0 {
		lvc.Cx = int32(tvc.width)
	} else {
		lvc.Cx = 100
	}

	switch tvc.alignment {
	case AlignCenter:
		lvc.Fmt = 2

	case AlignFar:
		lvc.Fmt = 1
	}

	j := tvc.tv.SendMessage(LVM_INSERTCOLUMN, uintptr(tvc.index), uintptr(unsafe.Pointer(&lvc)))
	if int(j) == -1 {
		return newError("TableView.SetModel: Failed to insert column.")
	}

	return nil
}

func (tvc *TableViewColumn) destroy() error {
	if FALSE == tvc.tv.SendMessage(LVM_DELETECOLUMN, uintptr(tvc.index), 0) {
		return newError("LVM_DELETECOLUMN")
	}

	return nil
}

func (tvc *TableViewColumn) update() error {
	if tvc.tv == nil || !tvc.visible {
		return nil
	}

	lvc := tvc.getLVCOLUMN()

	if FALSE == tvc.tv.SendMessage(LVM_SETCOLUMN, uintptr(tvc.index), uintptr(unsafe.Pointer(lvc))) {
		return newError("LVM_SETCOLUMN")
	}

	return nil
}

func (tvc *TableViewColumn) getLVCOLUMN() *LVCOLUMN {
	var lvc LVCOLUMN

	lvc.Mask = LVCF_FMT | LVCF_WIDTH | LVCF_TEXT | LVCF_SUBITEM
	lvc.ISubItem = int32(tvc.index)
	lvc.PszText = syscall.StringToUTF16Ptr(tvc.title)
	lvc.Cx = int32(tvc.Width())

	switch tvc.alignment {
	case AlignCenter:
		lvc.Fmt = 2

	case AlignFar:
		lvc.Fmt = 1
	}

	return &lvc
}
