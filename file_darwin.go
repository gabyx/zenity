package zenity

import (
	"os/exec"
	"strings"

	"github.com/ncruces/zenity/internal/osa"
)

func SelectFile(options ...Option) (string, error) {
	opts := optsParse(options)
	out, err := osa.Run("file", osa.File{
		Operation: "chooseFile",
		Prompt:    opts.title,
		Location:  opts.filename,
		Type:      appleFilters(opts.filters),
	})
	if err, ok := err.(*exec.ExitError); ok && err.ExitCode() == 1 {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if len(out) > 0 {
		out = out[:len(out)-1]
	}
	return string(out), nil
}

func SelectFileMutiple(options ...Option) ([]string, error) {
	opts := optsParse(options)
	out, err := osa.Run("file", osa.File{
		Operation: "chooseFile",
		Multiple:  true,
		Prompt:    opts.title,
		Location:  opts.filename,
		Type:      appleFilters(opts.filters),
	})
	if err, ok := err.(*exec.ExitError); ok && err.ExitCode() == 1 {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(out) > 0 {
		out = out[:len(out)-1]
	}
	if len(out) == 0 {
		return nil, nil
	}
	return strings.Split(string(out), "\x00"), nil
}

func SelectFileSave(options ...Option) (string, error) {
	opts := optsParse(options)
	out, err := osa.Run("file", osa.File{
		Operation: "chooseFileName",
		Prompt:    opts.title,
		Location:  opts.filename,
	})
	if err, ok := err.(*exec.ExitError); ok && err.ExitCode() == 1 {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if len(out) > 0 {
		out = out[:len(out)-1]
	}
	return string(out), nil
}

func SelectDirectory(options ...Option) (string, error) {
	opts := optsParse(options)
	out, err := osa.Run("file", osa.File{
		Operation: "chooseFolder",
		Prompt:    opts.title,
		Location:  opts.filename,
	})
	if err, ok := err.(*exec.ExitError); ok && err.ExitCode() == 1 {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if len(out) > 0 {
		out = out[:len(out)-1]
	}
	return string(out), nil
}

func appleFilters(filters []FileFilter) []string {
	var filter []string
	for _, f := range filters {
		for _, p := range f.Patterns {
			filter = append(filter, p) // FIXME
		}
	}
	return filter
}
