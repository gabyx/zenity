package zenity

import (
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

var (
	getOpenFileName             = comdlg32.NewProc("GetOpenFileNameW")
	getSaveFileName             = comdlg32.NewProc("GetSaveFileNameW")
	commDlgExtendedError        = comdlg32.NewProc("CommDlgExtendedError")
	shBrowseForFolder           = shell32.NewProc("SHBrowseForFolderW")
	shGetPathFromIDListEx       = shell32.NewProc("SHGetPathFromIDListEx")
	shCreateItemFromParsingName = shell32.NewProc("SHCreateItemFromParsingName")
)

func SelectFile(options ...Option) (string, error) {
	var args _OPENFILENAME
	args.StructSize = uint32(unsafe.Sizeof(args))
	args.Flags = 0x80008 // OFN_NOCHANGEDIR|OFN_EXPLORER

	opts := optsParse(options)
	if opts.title != "" {
		args.Title = syscall.StringToUTF16Ptr(opts.title)
	}
	if opts.filters != nil {
		args.Filter = &windowsFilters(opts.filters)[0]
	}

	res := [32768]uint16{}
	args.File = &res[0]
	args.MaxFile = uint32(len(res))
	args.InitialDir = initDirAndName(opts.filename, res[:])

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	n, _, _ := getOpenFileName.Call(uintptr(unsafe.Pointer(&args)))
	if n == 0 {
		return "", commDlgError()
	}
	return syscall.UTF16ToString(res[:]), nil
}

func SelectFileMutiple(options ...Option) ([]string, error) {
	var args _OPENFILENAME
	args.StructSize = uint32(unsafe.Sizeof(args))
	args.Flags = 0x80208 // OFN_NOCHANGEDIR|OFN_ALLOWMULTISELECT|OFN_EXPLORER

	opts := optsParse(options)
	if opts.title != "" {
		args.Title = syscall.StringToUTF16Ptr(opts.title)
	}
	if opts.filters != nil {
		args.Filter = &windowsFilters(opts.filters)[0]
	}

	res := [32768 + 1024*256]uint16{}
	args.File = &res[0]
	args.MaxFile = uint32(len(res))
	args.InitialDir = initDirAndName(opts.filename, res[:])

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	n, _, _ := getOpenFileName.Call(uintptr(unsafe.Pointer(&args)))
	if n == 0 {
		return nil, commDlgError()
	}

	var i int
	var nul bool
	var split []string
	for j, p := range res {
		if p == 0 {
			if nul {
				break
			}
			if i < j {
				split = append(split, string(utf16.Decode(res[i:j])))
			}
			i = j + 1
			nul = true
		} else {
			nul = false
		}
	}
	if len := len(split) - 1; len > 0 {
		base := split[0]
		for i := 0; i < len; i++ {
			split[i] = filepath.Join(base, string(split[i+1]))
		}
		split = split[:len]
	}
	return split, nil
}

func SelectFileSave(options ...Option) (string, error) {
	var args _OPENFILENAME
	args.StructSize = uint32(unsafe.Sizeof(args))
	args.Flags = 0x80008 // OFN_NOCHANGEDIR|OFN_EXPLORER

	opts := optsParse(options)
	if opts.title != "" {
		args.Title = syscall.StringToUTF16Ptr(opts.title)
	}
	if opts.overwrite {
		args.Flags |= 0x2 // OFN_OVERWRITEPROMPT
	}
	if opts.filters != nil {
		args.Filter = &windowsFilters(opts.filters)[0]
	}

	res := [32768]uint16{}
	args.File = &res[0]
	args.MaxFile = uint32(len(res))
	args.InitialDir = initDirAndName(opts.filename, res[:])

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	n, _, _ := getSaveFileName.Call(uintptr(unsafe.Pointer(&args)))
	if n == 0 {
		return "", commDlgError()
	}
	return syscall.UTF16ToString(res[:]), nil
}

func SelectDirectory(options ...Option) (string, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	hr, _, _ := coInitializeEx.Call(0, 0x6) // COINIT_APARTMENTTHREADED|COINIT_DISABLE_OLE1DDE
	if hr != 0x80010106 {                   // RPC_E_CHANGED_MODE
		if int32(hr) < 0 {
			return "", syscall.Errno(hr)
		}
		defer coUninitialize.Call()
	}

	opts := optsParse(options)

	var dialog *_IFileOpenDialog
	hr, _, _ = coCreateInstance.Call(
		_CLSID_FileOpenDialog, 0, 0x17, // CLSCTX_ALL
		_IID_IFileOpenDialog, uintptr(unsafe.Pointer(&dialog)))
	if int32(hr) < 0 {
		return browseForFolder(opts.title)
	}
	defer dialog.Call(dialog.vtbl.Release)

	var flgs int
	hr, _, _ = dialog.Call(dialog.vtbl.GetOptions, uintptr(unsafe.Pointer(&flgs)))
	if int32(hr) < 0 {
		return "", syscall.Errno(hr)
	}
	hr, _, _ = dialog.Call(dialog.vtbl.SetOptions, uintptr(flgs|0x68)) // FOS_NOCHANGEDIR|FOS_PICKFOLDERS|FOS_FORCEFILESYSTEM
	if int32(hr) < 0 {
		return "", syscall.Errno(hr)
	}

	if opts.title != "" {
		ptr := syscall.StringToUTF16Ptr(opts.title)
		dialog.Call(dialog.vtbl.SetTitle, uintptr(unsafe.Pointer(ptr)))
	}

	if opts.filename != "" {
		var item *_IShellItem
		ptr := syscall.StringToUTF16Ptr(opts.filename)
		hr, _, _ = shCreateItemFromParsingName.Call(
			uintptr(unsafe.Pointer(ptr)), 0,
			_IID_IShellItem,
			uintptr(unsafe.Pointer(&item)))

		if hr >= 0 && item != nil {
			dialog.Call(dialog.vtbl.SetDefaultFolder, uintptr(unsafe.Pointer(item)))
			item.Call(item.vtbl.Release)
		}
	}

	hr, _, _ = dialog.Call(dialog.vtbl.Show, 0)
	if hr == 0x800704c7 { // ERROR_CANCELLED
		return "", nil
	}
	if int32(hr) < 0 {
		return "", syscall.Errno(hr)
	}

	var item *_IShellItem
	hr, _, _ = dialog.Call(dialog.vtbl.GetResult, uintptr(unsafe.Pointer(&item)))
	if int32(hr) < 0 {
		return "", syscall.Errno(hr)
	}
	defer item.Call(item.vtbl.Release)

	var ptr uintptr
	hr, _, _ = item.Call(item.vtbl.GetDisplayName,
		0x80058000, // SIGDN_FILESYSPATH
		uintptr(unsafe.Pointer(&ptr)))
	if int32(hr) < 0 {
		return "", syscall.Errno(hr)
	}
	defer coTaskMemFree.Call(ptr)

	res := reflect.SliceHeader{Data: ptr, Len: 32768, Cap: 32768}
	return syscall.UTF16ToString(*(*[]uint16)(unsafe.Pointer(&res))), nil
}

func browseForFolder(title string) (string, error) {
	var args _BROWSEINFO
	args.Flags = 0x1 // BIF_RETURNONLYFSDIRS

	if title != "" {
		args.Title = syscall.StringToUTF16Ptr(title)
	}

	ptr, _, _ := shBrowseForFolder.Call(uintptr(unsafe.Pointer(&args)))
	if ptr == 0 {
		return "", nil
	}
	defer coTaskMemFree.Call(ptr)

	res := [32768]uint16{}
	shGetPathFromIDListEx.Call(ptr, uintptr(unsafe.Pointer(&res[0])), uintptr(len(res)), 0)

	return syscall.UTF16ToString(res[:]), nil
}

func initDirAndName(filename string, name []uint16) (dir *uint16) {
	if filename != "" {
		d, n := splitDirAndName(filename)
		if n != "" {
			copy(name, syscall.StringToUTF16(n))
		}
		if d != "" {
			return syscall.StringToUTF16Ptr(d)
		}
	}
	return nil
}

func windowsFilters(filters []FileFilter) []uint16 {
	var res []uint16
	for _, f := range filters {
		res = append(res, utf16.Encode([]rune(f.Name))...)
		res = append(res, 0)
		for _, p := range f.Patterns {
			res = append(res, utf16.Encode([]rune(p))...)
			res = append(res, uint16(';'))
		}
		res = append(res, 0)
	}
	if res != nil {
		res = append(res, 0)
	}
	return res
}

func commDlgError() error {
	n, _, _ := commDlgExtendedError.Call()
	if n == 0 {
		return nil
	} else {
		return fmt.Errorf("Common Dialog error: %x", n)
	}
}

type _OPENFILENAME struct {
	StructSize      uint32
	Owner           uintptr
	Instance        uintptr
	Filter          *uint16
	CustomFilter    *uint16
	MaxCustomFilter uint32
	FilterIndex     uint32
	File            *uint16
	MaxFile         uint32
	FileTitle       *uint16
	MaxFileTitle    uint32
	InitialDir      *uint16
	Title           *uint16
	Flags           uint32
	FileOffset      uint16
	FileExtension   uint16
	DefExt          *uint16
	CustData        uintptr
	FnHook          uintptr
	TemplateName    *uint16
	PvReserved      uintptr
	DwReserved      uint32
	FlagsEx         uint32
}

type _BROWSEINFO struct {
	Owner        uintptr
	Root         uintptr
	DisplayName  *uint16
	Title        *uint16
	Flags        uint32
	CallbackFunc uintptr
	LParam       uintptr
	Image        int32
}

func uuid(s string) uintptr {
	return (*reflect.StringHeader)(unsafe.Pointer(&s)).Data
}

var (
	_IID_IShellItem       = uuid("\x1e\x6d\x82\x43\x18\xe7\xee\x42\xbc\x55\xa1\xe2\x61\xc3\x7b\xfe")
	_IID_IFileOpenDialog  = uuid("\x88\x72\x7c\xd5\xad\xd4\x68\x47\xbe\x02\x9d\x96\x95\x32\xd9\x60")
	_CLSID_FileOpenDialog = uuid("\x9c\x5a\x1c\xdc\x8a\xe8\xde\x4d\xa5\xa1\x60\xf8\x2a\x20\xae\xf7")
)

type _COMObject struct{}

func (o *_COMObject) Call(trap uintptr, a ...uintptr) (r1, r2 uintptr, lastErr error) {
	self := uintptr(unsafe.Pointer(o))
	nargs := uintptr(len(a))
	switch nargs {
	case 0:
		return syscall.Syscall(trap, nargs+1, self, 0, 0)
	case 1:
		return syscall.Syscall(trap, nargs+1, self, a[0], 0)
	case 2:
		return syscall.Syscall(trap, nargs+1, self, a[0], a[1])
	default:
		panic("COM call with too many arguments.")
	}
}

type _IFileOpenDialog struct {
	_COMObject
	vtbl *_IFileOpenDialogVtbl
}

type _IShellItem struct {
	_COMObject
	vtbl *_IShellItemVtbl
}

type _IFileOpenDialogVtbl struct {
	_IFileDialogVtbl
	GetResults       uintptr
	GetSelectedItems uintptr
}

type _IFileDialogVtbl struct {
	_IModalWindowVtbl
	SetFileTypes        uintptr
	SetFileTypeIndex    uintptr
	GetFileTypeIndex    uintptr
	Advise              uintptr
	Unadvise            uintptr
	SetOptions          uintptr
	GetOptions          uintptr
	SetDefaultFolder    uintptr
	SetFolder           uintptr
	GetFolder           uintptr
	GetCurrentSelection uintptr
	SetFileName         uintptr
	GetFileName         uintptr
	SetTitle            uintptr
	SetOkButtonLabel    uintptr
	SetFileNameLabel    uintptr
	GetResult           uintptr
	AddPlace            uintptr
	SetDefaultExtension uintptr
	Close               uintptr
	SetClientGuid       uintptr
	ClearClientData     uintptr
	SetFilter           uintptr
}

type _IModalWindowVtbl struct {
	_IUnknownVtbl
	Show uintptr
}

type _IShellItemVtbl struct {
	_IUnknownVtbl
	BindToHandler  uintptr
	GetParent      uintptr
	GetDisplayName uintptr
	GetAttributes  uintptr
	Compare        uintptr
}

type _IUnknownVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
}
