#!/usr/bin/env bash

export CODE_TESTS_PATH="$PWD/out/test"
export CODE_TESTS_WORKSPACE="$PWD/testFixture"

node "$(pwd)/out/test/runTest"
