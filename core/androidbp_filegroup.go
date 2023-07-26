package core

import (
	"github.com/ARM-software/bob-build/internal/utils"
	"github.com/google/blueprint"
)

func (g *androidBpGenerator) filegroupActions(m *ModuleFilegroup, ctx blueprint.ModuleContext) {
	mod, err := AndroidBpFile().NewModule("filegroup", m.shortName())
	if err != nil {
		utils.Die("%v", err.Error())
	}
	mod.AddStringList("srcs", m.Properties.Srcs)
}
