#!/bin/sh

git for-each-ref --format="%(refname)" refs/ubik | xargs -n 1 git update-ref -d
