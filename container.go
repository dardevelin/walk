// Copyright 2010 The Walk Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package walk

import "unsafe"

import . "github.com/lxn/go-winapi"

type Margins struct {
	HNear, VNear, HFar, VFar int
}

type Layout interface {
	Container() Container
	SetContainer(value Container)
	Margins() Margins
	SetMargins(value Margins) error
	Spacing() int
	SetSpacing(value int) error
	LayoutFlags() LayoutFlags
	MinSize() Size
	Update(reset bool) error
}

func shouldLayoutWidget(widget Widget) bool {
	if widget == nil {
		return false
	}

	_, isSpacer := widget.(*Spacer)

	return isSpacer || widget.BaseWidget().visible
}

type Container interface {
	Widget
	Children() *WidgetList
	Layout() Layout
	SetLayout(value Layout) error
	DataBinder() *DataBinder
	SetDataBinder(dbm *DataBinder)
}

type RootWidget interface {
	Container
	Run() int
}

type ContainerBase struct {
	WidgetBase
	layout     Layout
	children   *WidgetList
	dataBinder *DataBinder
	persistent bool
}

func (cb *ContainerBase) LayoutFlags() LayoutFlags {
	if cb.layout == nil {
		return 0
	}

	return cb.layout.LayoutFlags()
}

func (cb *ContainerBase) MinSizeHint() Size {
	if cb.layout == nil {
		return Size{}
	}

	return cb.layout.MinSize()
}

func (cb *ContainerBase) SizeHint() Size {
	return Size{100, 100}
}

func (cb *ContainerBase) SetEnabled(enabled bool) {
	cb.WidgetBase.SetEnabled(enabled)

	setDescendantsEnabled(cb.widget, enabled)
}

func (cb *ContainerBase) SetFont(f *Font) {
	cb.WidgetBase.SetFont(f)

	setDescendantsFont(cb.widget, f)
}

func (cb *ContainerBase) Children() *WidgetList {
	return cb.children
}

func (cb *ContainerBase) Layout() Layout {
	return cb.layout
}

func (cb *ContainerBase) SetLayout(value Layout) error {
	if cb.layout != value {
		if cb.layout != nil {
			cb.layout.SetContainer(nil)
		}

		cb.layout = value

		if value != nil && value.Container() != Container(cb) {
			value.SetContainer(cb)
		}
	}

	return nil
}

func (cb *ContainerBase) DataBinder() *DataBinder {
	return cb.dataBinder
}

func (cb *ContainerBase) SetDataBinder(db *DataBinder) {
	if db == cb.dataBinder {
		return
	}

	if cb.dataBinder != nil {
		cb.dataBinder.SetBoundWidgets(nil)
	}

	cb.dataBinder = db

	if db != nil {
		var boundWidgets []Widget

		walkDescendants(cb.widget, func(w Widget) bool {
			if w.BaseWidget().Handle() == cb.hWnd {
				return true
			}

			if c, ok := w.(Container); ok && c.DataBinder() != nil {
				return false
			}

			for _, prop := range w.BaseWidget().name2Property {
				if _, ok := prop.Source().(string); ok {
					boundWidgets = append(boundWidgets, w)
					break
				}
			}

			return true
		})

		db.SetBoundWidgets(boundWidgets)
	}
}

func (cb *ContainerBase) forEachPersistableChild(f func(p Persistable) error) error {
	if cb.children == nil {
		return nil
	}

	for _, child := range cb.children.items {
		if persistable, ok := child.(Persistable); ok && persistable.Persistent() {
			if err := f(persistable); err != nil {
				return err
			}
		}
	}

	return nil
}

func (cb *ContainerBase) Persistent() bool {
	return cb.persistent
}

func (cb *ContainerBase) SetPersistent(value bool) {
	cb.persistent = value
}

func (cb *ContainerBase) SaveState() error {
	return cb.forEachPersistableChild(func(p Persistable) error {
		return p.SaveState()
	})
}

func (cb *ContainerBase) RestoreState() error {
	return cb.forEachPersistableChild(func(p Persistable) error {
		return p.RestoreState()
	})
}

func (cb *ContainerBase) SetSuspended(suspend bool) {
	wasSuspended := cb.Suspended()

	cb.WidgetBase.SetSuspended(suspend)

	if !suspend && wasSuspended && cb.layout != nil {
		cb.layout.Update(true)
	}

	walkDescendants(cb, func(w Widget) bool {
		if _, ok := w.(Container); !ok {
			return false
		}

		if w.BaseWidget() != cb.BaseWidget() {
			w.SetSuspended(suspend)
		}

		w.Invalidate()

		return true
	})
}

func (cb *ContainerBase) WndProc(hwnd HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_COMMAND:
		if lParam == 0 {
			switch HIWORD(uint32(wParam)) {
			case 0:
				cmdId := LOWORD(uint32(wParam))
				switch cmdId {
				case IDOK, IDCANCEL:
					root := rootWidget(cb)
					if root == nil {
						break
					}

					dlg, ok := root.(dialogish)
					if !ok {
						break
					}

					var button *PushButton
					if cmdId == IDOK {
						button = dlg.DefaultButton()
					} else {
						button = dlg.CancelButton()
					}

					if button != nil && button.Visible() && button.Enabled() {
						button.raiseClicked()
					}

					break
				}

				// Menu
				actionId := uint16(LOWORD(uint32(wParam)))
				if action, ok := actionsById[actionId]; ok {
					action.raiseTriggered()
					return 0
				}

			case 1:
				// Accelerator
			}
		} else {
			// The widget that sent the notification shall handle it itself.
			hWnd := HWND(lParam)
			if widget := widgetFromHWND(hWnd); widget != nil {
				widget.WndProc(hwnd, msg, wParam, lParam)
				return 0
			}
		}

	case WM_NOTIFY:
		nmh := (*NMHDR)(unsafe.Pointer(lParam))
		if widget := widgetFromHWND(nmh.HwndFrom); widget != nil {
			// The widget that sent the notification shall handle it itself.
			return widget.WndProc(hwnd, msg, wParam, lParam)
		}

	case WM_SIZE, WM_SIZING:
		if cb.layout != nil {
			cb.layout.Update(false)
		}
	}

	return cb.WidgetBase.WndProc(hwnd, msg, wParam, lParam)
}

func (cb *ContainerBase) onInsertingWidget(index int, widget Widget) (err error) {
	return nil
}

func (cb *ContainerBase) onInsertedWidget(index int, widget Widget) (err error) {
	if parent := widget.Parent(); parent == nil || parent.BaseWidget().hWnd != cb.hWnd {
		err = widget.SetParent(cb.widget.(Container))
		if err != nil {
			return
		}
	}

	if cb.layout != nil {
		cb.layout.Update(true)
	}

	return
}

func (cb *ContainerBase) onRemovingWidget(index int, widget Widget) (err error) {
	if widget.Parent() == nil {
		return
	}

	if widget.Parent().BaseWidget().hWnd == cb.hWnd {
		err = widget.SetParent(nil)
	}

	return
}

func (cb *ContainerBase) onRemovedWidget(index int, widget Widget) (err error) {
	if cb.layout != nil {
		cb.layout.Update(true)
	}

	return
}

func (cb *ContainerBase) onClearingWidgets() (err error) {
	for _, widget := range cb.children.items {
		if widget.Parent().BaseWidget().hWnd == cb.hWnd {
			if err = widget.SetParent(nil); err != nil {
				return
			}
		}
	}

	return
}

func (cb *ContainerBase) onClearedWidgets() (err error) {
	if cb.layout != nil {
		cb.layout.Update(true)
	}

	return
}
