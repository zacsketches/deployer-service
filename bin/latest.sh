#!/bin/bash
latest_tag=$(git tag --sort=-v:refname | head -n 1)
echo "Latest tag: $latest_tag"