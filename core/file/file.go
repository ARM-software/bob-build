package file

import (
	"path"
	"path/filepath"

	"github.com/ARM-software/bob-build/core/backend"
	"github.com/ARM-software/bob-build/core/toolchain"
)

type Type uint32

const (
	TypeUnset      = 0
	TypeSrc   Type = 1 << iota
	TypeGenerated
	TypeTool
	TypeBinary
	TypeExecutable
	TypeImplicit
	TypeC
	TypeCpp
	TypeAsm
	TypeHeader

	TypeArchive
	TypeShared
	TypeKernelModule
	TypeInstallable
	TypeDep
	TypeRsp
	TypeToc

	TypeLink // Special tag to indicate this file is a symlink

	// Masks:
	TypeCompilable = TypeC | TypeCpp | TypeAsm
)

type Path struct {
	backendPath string // either absolute location of the source tree, or generated file build root for AOSP/Linux respectively

	namespacePath string
	relativePath  string
	tag           Type // tag to indicate type

	symlink *Path
}

func (file Path) RelBuildPath() string {
	if file.IsType(TypeGenerated) {
		// We want to preserve /gen/ in the path when using relative build path
		return filepath.Join("gen", file.namespacePath, file.relativePath)
	} else {
		return filepath.Join(file.namespacePath, file.relativePath)
	}
}

func (file Path) BuildPath() string {
	return filepath.Join(file.backendPath, file.namespacePath, file.relativePath)
}

func (file Path) UnScopedPath() string {
	return file.relativePath
}

func (file Path) ScopedPath() string {
	return filepath.Join(file.namespacePath, file.relativePath)
}

func (file Path) Scope() string {
	return file.namespacePath
}

func (file Path) Ext() string {
	return path.Ext(file.relativePath)
}

func (file Path) IsType(ft Type) bool {
	return (file.tag & ft) != 0
}

func (file Path) IsNotType(ft Type) bool {
	return ((file.tag & ft) ^ ft) != 0
}

func (file Path) IsSymLink() bool {
	return file.symlink != nil
}

func (file Path) ExpandLink() *Path {
	if file.symlink != nil {
		return file.symlink
	} else {
		return &file
	}
}

func (file Path) FollowLink() *Path {
	if file.symlink != nil {
		return file.symlink.FollowLink()
	} else {
		return &file
	}
}

var FileNoNameSpace string = ""

func NewPath(relativePath string, namespace string, tag Type) Path {
	return New(relativePath, namespace, tag)
}

func NewLink(relativePath string, namespace string, from *Path, tag Type) Path {
	link := New(relativePath, namespace, from.tag|tag|TypeLink)
	link.symlink = from
	return link
}

func FromWithTag(from *Path, tag Type) Path {
	new := *from
	new.tag |= tag
	return new
}

func New(relativePath string, namespace string, tag Type) Path {

	switch path.Ext(relativePath) {
	case ".s", ".S":
		tag |= TypeAsm
	case ".c":
		tag |= TypeC
	case ".cc", ".cpp", ".cxx":
		tag |= TypeCpp
	case ".h", ".hpp":
		tag |= TypeHeader
	case ".a":
		tag |= TypeArchive
	case ".so", ".dll", ".dylib":
		tag |= TypeShared
	case ".ko":
		tag |= TypeKernelModule
	case ".toc":
		tag |= TypeToc
	}

	var backendPath string
	scopedPath := ""

	if (tag & TypeGenerated) != 0 {
		backendPath = filepath.Join(backend.Get().BuildDir(), "gen")
		scopedPath = namespace
	} else if (tag & (TypeBinary | TypeExecutable)) != 0 {
		backendPath = backend.Get().BinaryOutputDir(toolchain.TgtType(namespace))
	} else if (tag&TypeArchive) != 0 && ((tag&TypeSrc)^TypeSrc) != 0 {
		backendPath = backend.Get().StaticLibOutputDir(toolchain.TgtType(namespace))
	} else if (tag&(TypeShared|TypeToc)) != 0 && ((tag&TypeSrc)^TypeSrc) != 0 {
		backendPath = backend.Get().SharedLibsDir(toolchain.TgtType(namespace))
	} else if (tag & TypeKernelModule) != 0 {
		backendPath = filepath.Join(backend.Get().KernelModOutputDir(), namespace)
	} else {
		backendPath = backend.Get().SourceDir()
		scopedPath = FileNoNameSpace
	}

	return Path{
		backendPath:   backendPath,
		namespacePath: scopedPath,
		relativePath:  relativePath,
		tag:           tag,
		symlink:       nil,
	}
}
