#!/bin/bash

# Use $RANDOM to generate a random number and % 2 to get either 0 or 1
random_number=$((RANDOM % 2))

exit $random_number
