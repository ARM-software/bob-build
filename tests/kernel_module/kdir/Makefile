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

all: $(MODULES)

$(M)/%.ko: $(M)/%.o
	@$(CROSS_COMPILE)gcc -shared $< -o $@
	@echo > $(M)/Module.symvers

%.o: %.c
	@$(CROSS_COMPILE)gcc -c -o $@ $(CPPFLAGS) $(CFLAGS) $<

-include $(DEPFILES)
