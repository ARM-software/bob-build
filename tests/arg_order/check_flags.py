#!/usr/bin/env python3


import logging
import os
import sys


logger = logging.getLogger(__name__)


def parse_args():
    # Dump all args into a dictionary returning the position of the
    # last copy of that arg.
    #
    # For include arguments, strip any intermediate directories (all
    # our local includes directories are leaves)
    #
    # Return information on whether this command is link, and also
    # what the output file is.

    args = dict()
    link = True
    output = None

    skiparg = False
    for i, arg in enumerate(sys.argv[1:]):
        if skiparg:
            skiparg = False
            continue

        if arg == "-c":
            link = False
        elif arg == "-o":
            output = sys.argv[i + 2]
        elif arg.startswith("-I"):
            if len(arg) > 2:
                includedir = arg[2:]
            else:
                # Handle "-I includedir", merging to a single arg
                includedir = sys.argv[i + 2]
                skiparg = True

            arg = "-I{}".format(os.path.basename(includedir))
        args[arg] = i

    return (link, output, args)


def find_range(args, subset):
    """
    Return inclusive range, (first, last), of the elements of `subset` in
    the dictionary `args`.
    """
    first = 65536
    last = -1
    for a in subset:
        if a in args:
            if args[a] < first:
                first = args[a]
            if args[a] > last:
                last = args[a]
        else:
            logger.error("%s not in args", a)
    return (first, last)


def check_set1_before_set2(args, set1, set2):
    """
    Check all set1 arguments occur before set2 in args
    """
    _, s1_last = find_range(args, set1)
    s2_first, _ = find_range(args, set2)
    return s1_last < s2_first


class mod:
    def __init__(self, name, cflags, includes, defaults, **kwargs):
        self.name = name
        self.cflags = cflags
        self.includes = ["-I" + inc for inc in includes]
        self.defaults = defaults
        self.feature_cflags = []
        self.feature_includes = []
        self.target_cflags = []
        self.target_includes = []
        self.target_feature_cflags = []
        self.target_feature_includes = []

        if "feature_cflags" in kwargs:
            self.feature_cflags = kwargs["feature_cflags"]
        if "target_cflags" in kwargs:
            self.target_cflags = kwargs["target_cflags"]
        if "target_feature_cflags" in kwargs:
            self.target_feature_cflags = kwargs["target_feature_cflags"]
        if "feature_includes" in kwargs:
            self.feature_includes = ["-I" + inc for inc in kwargs["feature_includes"]]
        if "target_includes" in kwargs:
            self.target_includes = ["-I" + inc for inc in kwargs["target_includes"]]
        if "target_feature_includes" in kwargs:
            self.target_feature_includes = [
                "-I" + inc for inc in kwargs["target_feature_includes"]
            ]

    def find_flags(self, args):
        """
        Return the range (inclusive) for this module's cflags
        """
        return find_range(
            args,
            self.cflags
            + self.feature_cflags
            + self.target_cflags
            + self.target_feature_cflags,
        )

    def find_includes(self, args):
        """
        Return the range (inclusive) for this module's includes
        """
        return find_range(
            args,
            self.includes
            + self.feature_includes
            + self.target_includes
            + self.target_feature_includes,
        )

    def find_flags_recursive(self, args):
        """
        Return the range (inclusive) for this module's cflags, and all its
        dependencies
        """
        first, last = self.find_flags(args)
        for sub in self.defaults:
            (sub_first, sub_last) = sub.find_flags_recursive(args)
            if sub_first < first:
                first = sub_first
            if sub_last > last:
                last = sub_last

        return (first, last)

    def find_includes_recursive(self, args):
        """
        Return the range (inclusive) for this module's includes and all
        its dependencies
        """
        (first, last) = self.find_includes(args)
        for sub in self.defaults:
            (sub_first, sub_last) = sub.find_includes_recursive(args)
            if sub_first < first:
                first = sub_first
            if sub_last > last:
                last = sub_last

        return (first, last)

    def check_order_maintained(self, all_args, args):
        """
        Check that args occur in the same order in all_args.
        """
        result = True
        for i, f1 in enumerate(args):
            if f1 not in all_args:
                logger.error("%s not in args", f1)
                result = False
                continue

            for f2 in args[i + 1 :]:
                if f2 not in all_args:
                    logger.error("%s not in args", f2)
                    result = False
                    continue

                if all_args[f1] >= all_args[f2]:
                    logger.error(
                        "Module %s arguments out of order %s vs %s", self.name, f1, f2
                    )
                    result = False

        return result

    def check_flags(self, args):
        """
        Check this module's flags are in the expected order, then check
        that the child modules' flags are in order. Finally check the
        child modules' flags relative to each other.
        """
        result = True

        # Flags for just this module are in the same order
        if not self.check_order_maintained(args, self.cflags):
            result = False

        # Feature-specific cflags
        if not self.check_order_maintained(args, self.feature_cflags):
            logger.error(
                "Module %s feature-specific flag order not maintained", self.name
            )
            result = False
        if not check_set1_before_set2(args, self.cflags, self.feature_cflags):
            logger.error(
                "Module %s feature-specific flags not overriding normal flags",
                self.name,
            )
            result = False

        # Target-specific cflags
        if not self.check_order_maintained(args, self.target_cflags):
            logger.error(
                "Module %s target-specific flag order not maintained", self.name
            )
            result = False
        if not check_set1_before_set2(args, self.cflags, self.target_cflags):
            logger.error(
                "Module %s target-specific flags not overriding normal flags", self.name
            )
            result = False

        # Target-and-feature-specific cflags
        if not self.check_order_maintained(args, self.target_feature_cflags):
            logger.error(
                "Module %s target-and-feature-specific flag order not maintained",
                self.name,
            )
            result = False
        if not check_set1_before_set2(args, self.cflags, self.target_feature_cflags):
            logger.error(
                "Module %s flags target-and-feature-specific not overriding normal",
                self.name,
            )
            result = False
        if not check_set1_before_set2(
            args, self.feature_cflags, self.target_feature_cflags
        ):
            logger.error(
                "Module %s flags target-and-feature-specific not overriding feature-specific",
                self.name,
            )
            result = False
        if not check_set1_before_set2(
            args, self.target_cflags, self.target_feature_cflags
        ):
            logger.error(
                "Module %s flags target-and-feature-specific not overriding target-specific",
                self.name,
            )
            result = False

        # Check each child module's flags
        for sub in self.defaults:
            if sub.check_flags(args) != 0:
                result = False

        # Check child module flag ordering
        last_sub_flag = -1
        for i, sub1 in enumerate(self.defaults):
            (sub1_first, sub1_last) = sub1.find_flags_recursive(args)
            if sub1_last > last_sub_flag:
                last_sub_flag = sub1_last
            for sub2 in self.defaults[i + 1 :]:
                (sub2_first, sub2_last) = sub2.find_flags_recursive(args)
                if sub1_last >= sub2_first:
                    logger.error(
                        "Module %s submodules %s and %s flags out of order",
                        self.name,
                        sub1.name,
                        sub2.name,
                    )
                    result = False

        # Check this module's flags are after all submodule flags
        first_flag, _ = self.find_flags(args)
        if first_flag <= last_sub_flag:
            logger.error(
                "Module %s module flags not overriding submodule flags", self.name
            )
            result = False

        return result

    def check_includes(self, args):
        """
        Check this module's includes are in the expected order, then check
        that the child modules' includes are in order. Finally check
        the child modules' includes relative to each other.
        """
        result = True

        # Includes for just this module are in the same order
        if not self.check_order_maintained(args, self.includes):
            result = False

        # Feature-specific includes
        if not self.check_order_maintained(args, self.feature_includes):
            logger.error(
                "Module %s feature-specific include order not maintained", self.name
            )
            result = False
        if not check_set1_before_set2(args, self.feature_includes, self.includes):
            logger.error(
                "Module %s feature-specific includes not overriding normal includes",
                self.name,
            )
            result = False

        # Target-specific includes
        if not self.check_order_maintained(args, self.target_includes):
            logger.error(
                "Module %s target-specific include order not maintained", self.name
            )
            result = False
        if not check_set1_before_set2(args, self.target_includes, self.includes):
            logger.error(
                "Module %s target-specific includes not overriding normal includes",
                self.name,
            )
            result = False

        # Target-and-feature-specific includes
        if not self.check_order_maintained(args, self.target_feature_includes):
            logger.error(
                "Module %s target-and-feature-specific include order not maintained",
                self.name,
            )
            result = False
        if not check_set1_before_set2(
            args, self.target_feature_includes, self.includes
        ):
            logger.error(
                "Module %s includes target-and-feature-specific not overriding normal",
                self.name,
            )
            result = False
        if not check_set1_before_set2(
            args, self.target_feature_includes, self.feature_includes
        ):
            logger.error(
                "Module %s includes target-and-feature-specific not overriding feature-specific",
                self.name,
            )
            result = False
        if not check_set1_before_set2(
            args, self.target_feature_includes, self.target_includes
        ):
            logger.error(
                "Module %s includes target-and-feature-specific not overriding target-specific",
                self.name,
            )
            result = False

        # Check each child module's includes
        for sub in self.defaults:
            if sub.check_includes(args) != 0:
                result = False

        # Check child module include ordering
        first_sub_include = 65536
        for i, sub1 in enumerate(self.defaults):
            (sub1_first, sub1_last) = sub1.find_includes_recursive(args)
            if sub1_first < first_sub_include:
                first_sub_include = sub1_first
            for sub2 in self.defaults[i + 1 :]:
                (sub2_first, sub2_last) = sub2.find_includes_recursive(args)
                # sub2 includes must be before sub1 includes
                if sub1_first <= sub2_last:
                    logger.error(
                        "Module %s submodules %s and %s includes out of order",
                        self.name,
                        sub1.name,
                        sub2.name,
                    )
                    result = False

        # Check this module's includes are before all submodule includes
        last_include, _ = self.find_includes(args)
        if last_include >= first_sub_include:
            logger.error(
                "Module %s module includes not overriding submodule includes", self.name
            )
            result = False

        return result


def check_args(args):
    # The default structure looks like:
    #
    #         bin
    #       /     \
    #     a         b
    #   /   \     /   \
    # aa     ab ba     bb
    #
    # arguments specified in a single cflags [] should not be reordered.
    #
    # cflags in defaults at the same level should be in the order of
    # the defaults. i.e. a's cflags first, then b's cflags.
    #
    # defaults higher in the hierarchy have precedence, so bin's
    # cflags must come last, and a's cflags before aa and ab's.
    #
    aa = mod("aa", ["-aa_flag1", "-aa_flag2"], ["aa_include1", "aa_include2"], [])
    ab = mod("ab", ["-ab_flag1", "-ab_flag2"], ["ab_include1", "ab_include2"], [])
    ba = mod(
        "ba",
        ["-ba_flag1", "-ba_flag2"],
        ["ba_include1", "ba_include2"],
        [],
        target_flags=["-batarg1_flag", "-batarg2_flag"],
        target_feature_flags=["-batf1_flag", "batf2_flag"],
        target_includes=["batarg1_include", "batarg2_include"],
        target_feature_includes=["batf1_include", "batf2_include"],
    )
    bb = mod(
        "bb",
        ["-bb_flag1", "-bb_flag2"],
        ["bb_include1", "bb_include2"],
        [],
        target_flags=["-bbtarg1_flag", "-bbtarg2_flag"],
        target_feature_flags=["-bbtf1_flag", "bbtf2_flag"],
        target_includes=["bbtarg1_include", "bbtarg2_include"],
        target_feature_includes=["bbtf1_include", "bbtf2_include"],
    )
    a = mod("a", ["-a_flag1", "-a_flag2"], ["a_include1", "a_include2"], [aa, ab])
    b = mod("b", ["-b_flag1", "-b_flag2"], ["b_include1", "b_include2"], [ba, bb])
    bin = mod(
        "bin",
        ["-bin_flag1", "-bin_flag2"],
        ["bin_include1", "bin_include2"],
        [a, b],
        feature_flags=["-binfeat1_flag", "-binfeat2_flag"],
        target_flags=["-bintarg1_flag", "-bintarg2_flag"],
        target_feature_flags=["-bintf1_flag", "bintf2_flag"],
        feature_includes=["binfeat1_include", "binfeat2_include"],
        target_includes=["bintarg1_include", "bintarg2_include"],
        target_feature_includes=["bintf1_include", "bintf2_include"],
    )
    r1 = bin.check_flags(args)
    r2 = bin.check_includes(args)
    return r1 and r2


def main():
    link, output, args = parse_args()
    result = 0
    if not link:
        if check_args(args):
            result = 0
        else:
            result = 1

    if not output:
        logging.error("No output specified")
        result = 1

    if result == 0:
        # Ensure we update the write time on the output file
        # No need to actually call the compiler
        try:
            os.utime(output, None)
        except OSError:
            open(output, "a").close()
    return result


if __name__ == "__main__":
    logging.basicConfig(format="%(levelname)s: %(message)s", level=logging.WARNING)

    sys.exit(main())
