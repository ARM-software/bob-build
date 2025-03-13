#include <android/log.h>

void log_me(void) {
    __android_log_print(ANDROID_LOG_INFO, "LIBA", "libA secret: %d\n", 69);
}
