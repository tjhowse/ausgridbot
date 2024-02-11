#!/bin/bash

# Read credentials.secret.json into a variable, stripping newlines and indentation
CREDENTIALS=$(cat credentials.secret.json | jq -c .)
flyctl secrets set GRID_BOT_CREDENTIALS=''$CREDENTIALS''
