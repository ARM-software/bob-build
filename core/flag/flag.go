package flag

import (
	"path/filepath"

	"github.com/ARM-software/bob-build/core/backend"
	"github.com/google/blueprint"
)

type Type uint32

// Flags have an associated contextual information which is useful to filter based on use.
// TypeConly does not exist as it is equivalent to ((type & TypeCompilable) == TypeC)
const (
	// Flag types
	TypeUnset Type = 0
	TypeAsm   Type = 1 << iota
	TypeC
	TypeCpp
	TypeCC
	TypeLinker
	TypeLinkLibrary
	TypeInclude
	TypeIncludeLocal // Helper flag to mark local include dirs
	TypeIncludeGenerated
	TypeIncludeSystem

	TypeExported // Applied to direct downstream dep **only**
	TypeTransitive

	// Masks
	TypeCompilable = TypeAsm | TypeC | TypeCpp
)

type Flag struct {
	// raw stores the input as given from Bob. The final flag can then be constructed on demand using contextual
	// information from the owner and tag fields.
	raw string

	owner blueprint.Module // Pointer to the owning module, required for dynamic include paths which are scoped.
	tag   Type             // Type of flag. See the above enum for available types.
}

func (f Flag) Type() Type {
	return f.tag
}

func (f Flag) Raw() string {
	return f.raw
}

// Checks if flag matches the given mask exactly.
// Returns true if all the tags match.
func (f Flag) IsType(t Type) bool {
	return (f.tag & t) == t
}

// Check if a flag loosely matches the given type.
// This will return true if at least one of the tags matches.
func (f Flag) MatchesType(t Type) bool {
	return (f.tag & t) != 0
}

func (f Flag) IsNotType(t Type) bool {
	return ((f.tag & t) ^ t) != 0
}

// Helper string builder for include flags
func (f Flag) toStringInclude() string {
	prefix := "-I"
	path := f.raw

	if ((f.tag & TypeIncludeGenerated) == TypeIncludeGenerated) && (f.owner == nil) {
		panic("Owner must not be nil for generated include flags.")
	}

	if (f.tag & TypeIncludeSystem) != TypeUnset {
		prefix = "-isystem "
	}

	if (f.tag & TypeIncludeLocal) != TypeUnset {
		path = filepath.Join(backend.Get().SourceDir(), path)
	} else if (f.tag & TypeIncludeGenerated) != TypeUnset {
		path = filepath.Join(backend.Get().SourceOutputDir(f.owner), path)
	}

	return prefix + path
}

// Construct the final string flag at runtime.
func (f Flag) ToString() string {
	switch {
	case (f.tag & TypeInclude) != TypeUnset:
		return f.toStringInclude()
	default:
		return f.raw
	}
}

func FromIncludePath(path string, tag Type) Flag {
	return FromIncludePathOwned(path, nil, tag)
}

func FromIncludePathOwned(path string, owner blueprint.Module, tag Type) Flag {
	return Flag{
		owner: owner,
		raw:   path,
		tag:   tag | TypeInclude,
	}
}

func FromGeneratedIncludePath(path string, owner blueprint.Module, tag Type) Flag {
	return Flag{
		owner: owner,
		raw:   path,
		tag:   tag | TypeInclude | TypeIncludeGenerated,
	}
}

func FromDefineOwned(raw string, owner blueprint.Module, tag Type) Flag {
	return Flag{
		owner: owner,
		raw:   "-D" + raw,
		tag:   tag | TypeCC,
	}
}

func FromString(raw string, tag Type) Flag {
	return FromStringOwned(raw, nil, tag)
}

func FromStringOwned(raw string, owner blueprint.Module, tag Type) Flag {
	return Flag{
		owner: owner,
		raw:   raw,
		tag:   tag,
	}
}
