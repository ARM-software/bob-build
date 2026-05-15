//go:build soong
// +build soong

package soong_compat

import (
	"android/soong/android"
	"bytes"
	"fmt"

	"github.com/google/blueprint/gobtools"
)

func init() {
	GenruleOutputInfoId = gobtools.RegisterType(func() gobtools.CustomDec { return new(GenruleOutputInfo) })
	GenruleExportInclInfoId = gobtools.RegisterType(func() gobtools.CustomDec { return new(GenruleExportInclInfo) })
}

var GenruleOutputInfoId int16
var GenruleExportInclInfoId int16

type GenruleOutputInfo struct {
	Outputs         android.Paths
	ImplicitOutputs android.Paths
}

type GenruleExportInclInfo struct {
	ExportIncludes android.Paths
}

// This definition is compatible with Soong SHAs after `aa2555387 Add ctx to
// AndroidMkExtraEntriesFunc`
func ConvertAndroidMkExtraEntriesFunc(f AndroidMkExtraEntriesFunc) []android.AndroidMkExtraEntriesFunc {
	return []android.AndroidMkExtraEntriesFunc{
		func(ctx android.AndroidMkExtraEntriesContext, entries *android.AndroidMkEntries) {
			f(entries)
		},
	}
}

func SoongSupportsMkInstallTargets() bool {
	return true
}

// This definition is compatible with Soong SHAs _after_
// `f22120fb1da7f75a571966124bb0da6a57fd4f07 Change CommonModuleInfoProvider to a pointer.`
func GetHostBinPath(ctx android.ModuleContext, m android.ModuleOrProxy, host_bin string) android.OptionalPath {
	if p, ok := android.OtherModuleProvider(ctx, m, android.CommonModuleInfoProvider); ok && p.HostToolInfo != nil {
		return p.HostToolInfo.HostToolPath
	} else {
		panic(fmt.Errorf("No CommonModuleInfoProvider for %s module!", host_bin))
	}

	return android.OptionalPath{}
}

func (s GenruleOutputInfo) Encode(ctx gobtools.EncContext, buf *bytes.Buffer) error {
	var err error

	// encode s.Outputs length
	if err = gobtools.EncodeInt(buf, len(s.Outputs)); err != nil {
		return err
	}

	// encode s.ImplicitOutputs length
	if err = gobtools.EncodeInt(buf, len(s.ImplicitOutputs)); err != nil {
		return err
	}

	for _, path := range s.Outputs {
		if err = gobtools.EncodeInterface(ctx, buf, path); err != nil {
			return err
		}
	}

	for _, path := range s.ImplicitOutputs {
		if err = gobtools.EncodeInterface(ctx, buf, path); err != nil {
			return err
		}
	}

	return err
}

func (s GenruleOutputInfo) GetTypeId() int16 {
	return GenruleOutputInfoId
}

func (s *GenruleOutputInfo) Decode(ctx gobtools.EncContext, buf *bytes.Reader) error {
	var err error
	var outputsLen int
	var implicitOutputsLen int

	if err = gobtools.DecodeInt(buf, &outputsLen); err != nil {
		return err
	}

	if err = gobtools.DecodeInt(buf, &implicitOutputsLen); err != nil {
		return err
	}

	s.Outputs = make(android.Paths, outputsLen)
	s.ImplicitOutputs = make(android.Paths, implicitOutputsLen)

	for i := 0; i < outputsLen; i++ {
		if val, err := gobtools.DecodeInterface(ctx, buf); err != nil {
			return err
		} else if val == nil {
			s.Outputs[i] = nil
		} else {
			s.Outputs[i] = val.(android.Path)
		}
	}

	for i := 0; i < implicitOutputsLen; i++ {
		if val, err := gobtools.DecodeInterface(ctx, buf); err != nil {
			return err
		} else if val == nil {
			s.ImplicitOutputs[i] = nil
		} else {
			s.ImplicitOutputs[i] = val.(android.Path)
		}
	}

	return err
}

func (s GenruleExportInclInfo) Encode(ctx gobtools.EncContext, buf *bytes.Buffer) error {
	var err error

	// encode s.ExportIncludes length
	if err = gobtools.EncodeInt(buf, len(s.ExportIncludes)); err != nil {
		return err
	}

	for _, path := range s.ExportIncludes {
		if err = gobtools.EncodeInterface(ctx, buf, path); err != nil {
			return err
		}
	}

	return err
}

func (s GenruleExportInclInfo) GetTypeId() int16 {
	return GenruleExportInclInfoId
}

func (s *GenruleExportInclInfo) Decode(ctx gobtools.EncContext, buf *bytes.Reader) error {
	var err error
	var exportIncludesLen int

	if err = gobtools.DecodeInt(buf, &exportIncludesLen); err != nil {
		return err
	}

	s.ExportIncludes = make(android.Paths, exportIncludesLen)

	for i := 0; i < exportIncludesLen; i++ {
		if val, err := gobtools.DecodeInterface(ctx, buf); err != nil {
			return err
		} else if val == nil {
			s.ExportIncludes[i] = nil
		} else {
			s.ExportIncludes[i] = val.(android.Path)
		}
	}

	return err
}
