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
import argparse

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

def process_message(msg, suppress_warnings=True, html=False):
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

    codestart, codeend = "`", "`"
    if html:
        codestart = "<pre>"
        codeeng = "</pre>"

    return "%s (%s): %s%s%s" % ("/".join(model), linenum, codestart,
        "".join(remaining_message).lstrip(" "), codeend)


def process_output(mode, fh):
    # Linter input is read directly from stdin, this returns when the
    # stdin input finishes, which is the complete JSON document.
    lint_input = sys.stdin.read()

    lint_json = json.loads(lint_input)

    if not "tests" in lint_json:
        print("invalid JSON received from linter", file=sys.stderr)
        sys.exit(1)

    output = []
    overall_failed = False
    for fn, result in lint_json["tests"].iteritems():

        testdir = "/".join(fn_to_model_dir(fn))
        if "status" in result:
            if result["message"] not in BUILD_ERRORS:
                if mode == "markdown":
                    output.append("* :no_entry: **%s**: %s" % (testdir,
                      result["message"]))
                else:
                    output.extend(["<details>",
                                   "  <summary>:no_entry: %s</summary>" % testdir,
                                   "  %s" % result["message"],
                                   "</details>"])
            overall_failed = True
            continue

        any_failed = False
        test_out = []
        for testname, test in result["tests"].iteritems():
            if not test["status"] == "pass":
                any_failed = True
                if mode == "markdown":
                    test_out.append("  * :no_entry: **%s**:" % testname)
                else:
                    test_out.extend(["<details>",
                                     "  <summary>:no_entry: %s</summary>" % testname,
                                     "  <ul>"])

                for message in test["messages"]:
                    h = False
                    if mode == "html":
                      h = True
                    m = process_message(message, html=h)
                    if m is not None:
                        if mode == "markdown":
                            test_out.append("       * %s" % m)
                        else:
                            test_out.append("   <li>%s</li>" % m)

                if mode == "html":
                    test_out.extend(["  </ul>",
                                     "</details>"])
            else:
                if mode == "markdown":
                    test_out.append("  * :white_check_mark: **%s**" % testname)
                else:
                    test_out.extend(["<details>",
                                     "  <summary>:white_check_mark: %s</summary>" % testname,
                                     "  Tests Passed.",
                                     "</details>"])
        if len(test_out):
            simg = ":white_check_mark:"
            if any_failed:
                simg = ":no_entry:"
                overall_failed = True

            if mode == "markdown":
                output.append("* %s **%s**" % (simg,testdir))
            else:
                output.extend(["<details>",
                               "  <summary>%s %s</summary>" % (simg, testdir)])

            for t in test_out:
                output.append(t)

            if mode == "html":
                output.append("</details>")
    
    print("\n".join(output), file=fh)
    return overall_failed


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument("-m", "--mode", action="store", default="html")
    parser.add_argument("-f", "--file", action="store", default="stdout")

    args = parser.parse_args()
    if args.mode.lower() not in ["html", "markdown"]:
        print("Invalid mode specified: %s" % mode, file=sys.stderr)
        sys.exit(1)

    if args.file != "stdout":
        try:
            fh = open(args.file, 'w')
        except IOError as e:
            print("Invalid file specified: %s, error: %s", args.file, e)
            sys.exit(1)
    else:
        fh = sys.stdout

    if process_output(args.mode.lower(), fh):
        sys.exit(1)
