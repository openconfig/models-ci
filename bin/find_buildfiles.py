#!/usr/bin/env python
"""
  find_buildfiles extracts the YANG modules required to build an OC model

  Each OpenConfig model is distributed with a .spec.yml file that determines
  the set of files that should be used for building the YANG module in the
  directory in question. This script parses such files and returns:

    - A flag set to 0 (false) or (1) true, indicating whether the run-ci
      setting is set to true of false.
    - A comma separated set of files that should be provided to the model
      parser/compiler.

  The output value is provided in the form <flag>!<files>.
  A line of output of this form is output per build specification included
  in the YAML file.
"""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function
import yaml
import argparse
import sys

def main(argv=()):
  # Build an argument parser that lets the build file's path be
  # read from the command-line.
  parser = argparse.ArgumentParser()
  parser.add_argument('--specfile', '-s',
                      required=True,
                      type=str,
                      help='build specification file')

  args = parser.parse_args()
  if args.specfile is None:
    print('a specfile must be specified', file=sys.stderr)
    sys.exit(1)

  try:
    fh = open(args.specfile, 'r')
  except IOError as e:
    print('could not open the specification file:', e, file=sys.stderr)
    sys.exit(1)

  yaml_spec = yaml.safe_load(fh)

  for buildspec in yaml_spec:
    runci = True
    if "run-ci" in buildspec:
      runci = buildspec["run-ci"]

    name = "undefined"
    if "name" in buildspec:
      name = buildspec["name"]

    # If CI shouldn't be run, then don't parse the build rules.
    if not runci:
      print("0!")
      continue

    if "build" not in buildspec:
      print('build specification file did not include a build stanza', buildspec, file=sys.stderr)
      sys.exit(1)

    files = []
    for fn in buildspec["build"]:
      files.append(fn)

    print("%s!%s!%s" % (1 if runci else 0, ",".join(files), name))

if __name__ == '__main__':
  main()
