#if FOO != 1
#error "FOO is not 1. Top level property is broken."
#endif

#ifdef DEBUG
#define FEATURES_WORKING 1
#if DEBUG == 1
/* #warning "DEBUG is 1. Feature is on and working." */
#elif DEBUG == 0
/* #warning "DEBUG is 0. Feature is off and working." */
#endif
#else
#error "DEBUG is not set. Features broken"
#endif

/* Either HOST or TARGET should be set */
#if TARGET == 1
 /* #warning "TARGET is 1. Target build property works." */
#elif HOST == 1
 /* #warning "HOST is 1. Host build property works." */
#else
#error "Neither HOST or TARGET is 1. Target property propagation broken."
#endif

#ifdef FEATURES_WORKING
#if TARGET == 1
#if defined(TARGET_DEBUG) && TARGET_DEBUG == DEBUG
 /* #warning "TARGET_DEBUG matches DEBUG. Target specific features working." */
#else
#error "TARGET_DEBUG does not match DEBUG. Target specific features broken."
#endif /* TARGET_DEBUG */
#endif /* TARGET */

#if HOST == 1
#if defined(HOST_DEBUG) && HOST_DEBUG == DEBUG
 /* #warning "HOST_DEBUG matches DEBUG. Host specific features working." */
#else
#error "HOST_DEBUG does not match DEBUG. Host specific features broken."
#endif /* HOST_DEBUG */
#endif /* HOST */
#endif /* FEATURES_WORKING */

int func(void) {
	return 0;
}
