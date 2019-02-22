LOCAL_PATH := $(call my-dir)

include $(CLEAR_VARS)

LOCAL_MODULE := libbob_test_external_static
LOCAL_MODULE_CLASS := STATIC_LIBRARIES

LOCAL_SRC_FILES := external_lib.c
LOCAL_CFLAGS := -DFUNC_NAME=external_static
LOCAL_EXPORT_C_INCLUDE_DIRS := $(LOCAL_PATH)/static

include $(BUILD_STATIC_LIBRARY)

include $(CLEAR_VARS)

LOCAL_MODULE := libbob_test_external_shared
LOCAL_MODULE_CLASS := SHARED_LIBRARIES

LOCAL_SRC_FILES := external_lib.c
LOCAL_CFLAGS := -DFUNC_NAME=external_shared
LOCAL_EXPORT_C_INCLUDE_DIRS := $(LOCAL_PATH)/shared

include $(BUILD_SHARED_LIBRARY)

include $(CLEAR_VARS)

LOCAL_MODULE := libbob_test_external_header
LOCAL_EXPORT_C_INCLUDE_DIRS := $(LOCAL_PATH)/header

include $(BUILD_HEADER_LIBRARY)
