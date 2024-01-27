#!/usr/bin/env bash

# convert markdown adrs to asciidoc
./dtcw exportMarkdown

# create a special include file which references all adrs
./dtcw specialCollectIncludes

# build the microsite
./dtcw generateSite
