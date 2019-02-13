LOCAL_PATH := $(call my-dir)

include $(CLEAR_VARS)

LOCAL_MODULE := libbob_test_external_static
LOCAL_MODULE_CLASS := STATIC_LIBRARIES

LOCAL_SRC_FILES := external_static.c

include $(BUILD_STATIC_LIBRARY)
