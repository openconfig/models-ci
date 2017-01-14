#!/usr/bin/env python
# -*- coding: UTF-8 -*-
"""
    lint_to_md takes an input OpenConfig linter document and outputs GFM.

    lint_to_md takes the JSON output of the OpenConfig linter CI script and
    outputs GitHub flavoured MarkDown. The intention is that the output that
    is provided by this script can be put into a GitHub comment (e.g., in a
    pull request) showing whether the CI tests have passed or failed.
"""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function
from __future__ import unicode_literals

import sys
import json

# Build errors to be ignored based on the fact that they indicate a repo
# setup issue, not an error to be reported to a user.
BUILD_ERRORS = [
    "did not have a valid build file",
]

def fn_to_model_dir(fn):
    """
    fn_to_model_dir takes an input file path, and outputs the sections of the
    path that comes after "models/yang" in the path. This gives the elements of
    the path that allow localisation of the OpenConfig model.

    Input:
        - fn (string) - the file path to be considered
    Output:
        - model_parts (list) - the parts of the file path that follow
          "models/yang"
    """
    parts = fn.split("/")
    model_parts = []
    in_models, in_yang = False, False
    for p in parts:
        if p == "models":
            in_models = True
            continue
        elif p == "yang":
            in_yang = True
            continue

        if in_yang and in_models:
            model_parts.append(p)
    return model_parts

def process_message(msg, suppress_warnings=True):
    """
    process_message
    """
    message_parts = msg.split(":")
    if len(message_parts) < 3:
        # This is an unknown message that does not have the expected structure.
        return msg

    model = fn_to_model_dir(message_parts[0])
    remaining_start = 3

    linenum = message_parts[1]
    # Get rid of subpath information with the line number as this is not
    # useful to users.
    if "(" in linenum:
        linenum = linenum.split("(")[0].replace(" ", "")
        remaining_start = 4

    remaining_message = []
    for i in range(remaining_start,len(message_parts)):
        if message_parts[i] == "":
            remaining_message.append(":")
        else:
            remaining_message.append(message_parts[i])

    errorlevel = message_parts[2]

    if suppress_warnings and errorlevel.replace(" ", "") == "warning":
        return None

    return "%s (%s): `%s`" % ("/".join(model), linenum,
        "".join(remaining_message).lstrip(" "))


def main():
    # Linter input is read directly from stdin, this returns when the
    # stdin input finishes, which is the complete JSON document.
    lint_input = sys.stdin.read()

    lint_json = json.loads(lint_input)

    if not "tests" in lint_json:
        print("invalid JSON received from linter", file=sys.stderr)
        sys.exit(1)

    output = []
    for fn, result in lint_json["tests"].iteritems():

        testdir = "/".join(fn_to_model_dir(fn))
        if "status" in result:
            if result["message"] not in BUILD_ERRORS:
                output.append("* :no_entry: **%s**: %s" % (testdir,
                    result["message"]))
            continue

        any_failed = False
        test_out = []
        for testname, test in result["tests"].iteritems():
            if not test["status"] == "pass":
                any_failed = True
                test_out.append("  * :no_entry: **%s**:" % testname)
                for message in test["messages"]:
                    m = process_message(message)
                    if m is not None:
                        test_out.append("       * %s" % m)
            else:
                test_out.append("  * :white_check_mark: **%s**" % testname)

        if len(test_out):
            if any_failed:
                output.append("* :no_entry: **%s**:" % testdir)
            else:
                output.append("* :white_check_mark: **%s**" % testdir)

            for t in test_out:
                output.append(t)
    print("\n".join(output))


if __name__ == '__main__':
    main()
