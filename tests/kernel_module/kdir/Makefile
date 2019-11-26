# Emulate the behavior of an out-of-tree kernel module build for testing
# purposes, i.e.:
#  - Accept a 'M=' parameter specifying the module directory
#  - Include the Kbuild directory in $(M) to get the value of $(obj-m)
#  - Try to create a shared library from a single source file where Bob expects
#    a .ko to be generated.

MODULES :=

include $(M)/Kbuild
MODULES += $(addprefix $(M)/,$(obj-m:.o=.ko))
DEPFILES += $(addprefix $(M)/,$(obj-m:.o=.d))
CPPFLAGS := -Iinclude -MMD -MP
CFLAGS := -fPIC $(EXTRA_CFLAGS)

# Test that KBUILD_EXTRA_SYMBOLS is correct by actually linking dependent
# modules together. Each Module.symvers file contains the basename of the
# module it was generated with, so that we can pass it to `-l`. Use the
# `:filename` syntax for `-l`, because this allows linking with libraries whose
# names are not of the form `lib*.so`.
DEPENDENT_MODULES := $(foreach symvers,$(KBUILD_EXTRA_SYMBOLS),-L $(dir $(symvers)) -l:$(shell cat $(symvers)))

all: $(MODULES)

$(M)/%.ko: $(M)/%.o
	@$(CROSS_COMPILE)gcc -Wl,--no-undefined -shared $< -o $@ $(DEPENDENT_MODULES)
	@echo $(@F) > $(M)/Module.symvers

%.o: %.c
	@$(CROSS_COMPILE)gcc -c -o $@ $(CPPFLAGS) $(CFLAGS) $<

-include $(DEPFILES)
