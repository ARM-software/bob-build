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
CFLAGS := -fPIC $(EXTRA_CFLAGS) -nostdlib
LDFLAGS := -Wl,--no-undefined -shared -nostdlib

# Test that KBUILD_EXTRA_SYMBOLS is correct by actually linking dependent
# modules together. Each Module.symvers file contains the basename of the
# module it was generated with, so that we can pass it to `-l`. Use the
# `:filename` syntax for `-l`, because this allows linking with libraries whose
# names are not of the form `lib*.so`.
DEPENDENT_MODULES := $(foreach symvers,$(KBUILD_EXTRA_SYMBOLS),-L $(dir $(symvers)) -l:$(shell cat $(symvers)))

# Attempt to use the compiler requested by CC, CROSS_COMPILE and CLANG_TRIPLE variables.
#  If CLANG_TRIPLE is set assume we are using clang, otherwise gcc.
#  If CC is set, that is the gcc or clang binary to use.
ifndef CC
ifdef CLANG_TRIPLE
CC := clang
else
CC := $(CROSS_COMPILE)gcc
endif
endif

LOCAL_CC := $(CC)
ifdef CLANG_TRIPLE
LOCAL_CC += -target $(CLANG_TRIPLE)
LDFLAGS += -fuse-ld=lld
endif

all: $(MODULES)

$(M)/%.ko: $(M)/%.o
	@$(LOCAL_CC) $(LDFLAGS) $< -o $@ $(DEPENDENT_MODULES)
	@echo $(@F) > $(M)/Module.symvers

%.o: %.c
	$(LOCAL_CC) -c -o $@ $(CPPFLAGS) $(CFLAGS) $<

-include $(DEPFILES)
