// Package zenity provides cross-platform access to simple dialogs that interact
// graphically with the user.
//
// It is inspired by, and closely follows the API of, the zenity program, which
// it uses to provide the functionality on various Unixes. See:
//
// https://help.gnome.org/users/zenity/stable/
//
// This package does not require cgo, and it does not impose any threading or
// initialization requirements.
package zenity

import (
	"context"
	"image/color"
)

type constError string

func (e constError) Error() string { return string(e) }

// ErrCancelOrClosed is returned by certain dialog functions when the cancel
// button is pressed or the dialog is closed.
const ErrCancelOrClosed = constError("Canceled or closed")

type options struct {
	// General options
	title string

	// File selection options
	filename         string
	directory        bool
	confirmOverwrite bool
	confirmCreate    bool
	showHidden       bool
	fileFilters      []FileFilter

	// Color selection options
	color       color.Color
	showPalette bool

	// Message & List options
	icon          DialogIcon
	okLabel       string
	cancelLabel   string
	extraButton   string
	noWrap        bool
	ellipsize     bool
	defaultCancel bool

	// List options
	multipleSelection bool

	// Context for timeout
	ctx context.Context
}

// An Option is an argument passed to dialog functions to customize their
// behavior.
type Option interface {
	apply(*options)
}

type funcOption func(*options)

func (f funcOption) apply(o *options) { f(o) }

func applyOptions(options []Option) (res options) {
	for _, o := range options {
		o.apply(&res)
	}
	return
}

// Title returns an Option to set the dialog title.
func Title(title string) Option {
	return funcOption(func(o *options) { o.title = title })
}

// DialogIcon is the enumeration for dialog icons.
type DialogIcon int

// The stock dialog icons.
const (
	ErrorIcon DialogIcon = iota + 1
	WarningIcon
	InfoIcon
	QuestionIcon
)

// Icon returns an Option to set the dialog icon.
func Icon(icon DialogIcon) Option {
	return funcOption(func(o *options) { o.icon = icon })
}

// Context returns an Option to set a Context that can dismiss the dialog.
//
// Dialogs dismissed by the Context return Context.Err.
func Context(ctx context.Context) Option {
	return funcOption(func(o *options) { o.ctx = ctx })
}
