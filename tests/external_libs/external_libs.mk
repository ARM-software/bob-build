LOCAL_PATH := $(call my-dir)

include $(CLEAR_VARS)

LOCAL_MODULE := libbob_test_external_static
LOCAL_MODULE_CLASS := STATIC_LIBRARIES

LOCAL_SRC_FILES := external_lib.c
LOCAL_CFLAGS := -DTYPE=static
LOCAL_EXPORT_C_INCLUDE_DIRS := static

include $(BUILD_STATIC_LIBRARY)

include $(CLEAR_VARS)

LOCAL_MODULE := libbob_test_external_shared
LOCAL_MODULE_CLASS := STATIC_LIBRARIES

LOCAL_SRC_FILES := external_lib.c
LOCAL_CFLAGS := -DTYPE=shared
LOCAL_EXPORT_C_INCLUDE_DIRS := shared

include $(BUILD_SHARED_LIBRARY)
