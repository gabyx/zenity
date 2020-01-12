package main

import (
	"flag"
	"os"
	"strings"

	"github.com/ncruces/zenity"
	"github.com/ncruces/zenity/internal/cmd"
)

//go:generate go run github.com/josephspurrier/goversioninfo/cmd/goversioninfo -platform-specific -manifest=win.manifest

var (
	// Application Options
	errorDlg         bool
	infoDlg          bool
	warningDlg       bool
	questionDlg      bool
	fileSelectionDlg bool

	// General options
	title string

	// Message options
	text          string
	iconName      string
	okLabel       string
	cancelLabel   string
	extraButton   string
	noWrap        bool
	ellipsize     bool
	defaultCancel bool

	// File selection options
	save             bool
	multiple         bool
	directory        bool
	confirmOverwrite bool
	confirmCreate    bool
	showHidden       bool
	filename         string
	separator        string
	fileFilters      FileFilters
)

func main() {
	setupFlags()
	flag.Parse()
	validateFlags()
	opts := loadFlags()
	cmd.Command = true

	switch {
	case errorDlg:
		msgResult(zenity.Error(text, opts...))
	case infoDlg:
		msgResult(zenity.Info(text, opts...))
	case warningDlg:
		msgResult(zenity.Warning(text, opts...))
	case questionDlg:
		msgResult(zenity.Question(text, opts...))

	case fileSelectionDlg:
		switch {
		default:
			strResult(zenity.SelectFile(opts...))
		case save:
			strResult(zenity.SelectFileSave(opts...))
		case multiple:
			lstResult(zenity.SelectFileMutiple(opts...))
		}
	}

	flag.Usage()
	os.Exit(-1)
}

func setupFlags() {
	// Application Options

	flag.BoolVar(&errorDlg, "error", false, "Display error dialog")
	flag.BoolVar(&infoDlg, "info", false, "Display info dialog")
	flag.BoolVar(&warningDlg, "warning", false, "Display warning dialog")
	flag.BoolVar(&questionDlg, "question", false, "Display question dialog")
	flag.BoolVar(&fileSelectionDlg, "file-selection", false, "Display file selection dialog")

	// General options

	flag.StringVar(&title, "title", "", "Set the dialog title")

	// Message options

	flag.StringVar(&text, "text", "", "Set the dialog text")
	flag.StringVar(&iconName, "icon-name", "", "Set the dialog icon (error, info, question, warning)")
	flag.StringVar(&okLabel, "ok-label", "", "Set the label of the OK button")
	flag.StringVar(&cancelLabel, "cancel-label", "", "Set the label of the Cancel button")
	flag.StringVar(&extraButton, "extra-button", "", "Add an extra button")
	flag.BoolVar(&noWrap, "no-wrap", false, "Do not enable text wrapping")
	flag.BoolVar(&ellipsize, "ellipsize", false, "Enable ellipsizing in the dialog text")
	flag.BoolVar(&defaultCancel, "default-cancel", false, "Give Cancel button focus by default")

	// File selection options

	flag.BoolVar(&save, "save", false, "Activate save mode")
	flag.BoolVar(&multiple, "multiple", false, "Allow multiple files to be selected")
	flag.BoolVar(&directory, "directory", false, "Activate directory-only selection")
	flag.BoolVar(&confirmOverwrite, "confirm-overwrite", false, "Confirm file selection if filename already exists")
	flag.BoolVar(&confirmCreate, "confirm-create", false, "Confirm file selection if filename does not yet exist (Windows only)")
	flag.BoolVar(&showHidden, "show-hidden", false, "Show hidden files (Windows and macOS only)")
	flag.StringVar(&filename, "filename", "", "Set the filename")
	flag.StringVar(&separator, "separator", "|", "Set output separator character")
	flag.Var(&fileFilters, "file-filter", "Set a filename filter (NAME | PATTERN1 PATTERN2 ...)")
}

func validateFlags() {
	var n int
	if errorDlg {
		n++
	}
	if infoDlg {
		n++
	}
	if warningDlg {
		n++
	}
	if questionDlg {
		n++
	}
	if fileSelectionDlg {
		n++
	}
	if n != 1 {
		flag.Usage()
		os.Exit(-1)
	}
}

func loadFlags() []zenity.Option {
	var options []zenity.Option

	// General options

	options = append(options, zenity.Title(title))

	// Message options

	var icon zenity.MessageIcon
	switch iconName {
	case "error", "dialog-error":
		icon = zenity.ErrorIcon
	case "info", "dialog-information":
		icon = zenity.InfoIcon
	case "question", "dialog-question":
		icon = zenity.QuestionIcon
	case "warning", "dialog-warning":
		icon = zenity.WarningIcon
	}

	options = append(options, zenity.Icon(icon))
	options = append(options, zenity.OKLabel(okLabel))
	options = append(options, zenity.CancelLabel(cancelLabel))
	options = append(options, zenity.ExtraButton(extraButton))
	if noWrap {
		options = append(options, zenity.NoWrap())
	}
	if ellipsize {
		options = append(options, zenity.Ellipsize())
	}
	if defaultCancel {
		options = append(options, zenity.DefaultCancel())
	}

	// File selection options

	options = append(options, fileFilters.Build())
	options = append(options, zenity.Filename(filename))
	if directory {
		options = append(options, zenity.Directory())
	}
	if confirmOverwrite {
		options = append(options, zenity.ConfirmOverwrite())
	}
	if confirmCreate {
		options = append(options, zenity.ConfirmCreate())
	}
	if showHidden {
		options = append(options, zenity.ShowHidden())
	}

	cmd.Separator = separator

	return options
}

func msgResult(ok bool, err error) {
	if err == zenity.ErrExtraButton {
		os.Stdout.WriteString(extraButton)
		os.Stdout.WriteString(cmd.LineBreak)
		os.Exit(1)
	}
	if err != nil {
		os.Stderr.WriteString(err.Error())
		os.Stderr.WriteString(cmd.LineBreak)
		os.Exit(-1)
	}
	if ok {
		os.Exit(0)
	}
	os.Exit(1)
}

func strResult(s string, err error) {
	if err != nil {
		os.Stderr.WriteString(err.Error())
		os.Stderr.WriteString(cmd.LineBreak)
		os.Exit(-1)
	}
	if s == "" {
		os.Exit(1)
	}
	os.Stdout.WriteString(s)
	os.Stdout.WriteString(cmd.LineBreak)
	os.Exit(0)
}

func lstResult(l []string, err error) {
	if err != nil {
		os.Stderr.WriteString(err.Error())
		os.Stderr.WriteString(cmd.LineBreak)
		os.Exit(-1)
	}
	os.Stdout.WriteString(strings.Join(l, separator))
	os.Stdout.WriteString(cmd.LineBreak)
	if l == nil {
		os.Exit(1)
	}
	os.Exit(0)
}

type FileFilters struct {
	zenity.FileFilters
}

func (f *FileFilters) String() string {
	return "filename filter"
}

func (f *FileFilters) Set(s string) error {
	var filter zenity.FileFilter

	if split := strings.SplitN(s, "|", 2); len(split) > 1 {
		filter.Name = split[0]
		s = split[1]
	}

	filter.Patterns = strings.Split(s, " ")
	f.FileFilters = append(f.FileFilters, filter)

	return nil
}