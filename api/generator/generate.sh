#! /bin/bash

set -e

if [ -d "api" ]; then
  cd api
fi

if [ -d "generator" ]; then
  cd generator
fi

GENERATOR_VERSION="7.6.0"
GENERATOR="openapi-generator-cli-$GENERATOR_VERSION.jar"

if [ ! -f "$GENERATOR" ]; then
  curl https://repo1.maven.org/maven2/org/openapitools/openapi-generator-cli/$GENERATOR_VERSION/$GENERATOR > $GENERATOR
fi

API_MODEL_PACKAGE_NAME="apimodel"

rm -rf tmp

mkdir -p tmp

java -jar "$GENERATOR" generate \
  -i "../openapi-spec.yaml" \
  -o "tmp/$API_MODEL_PACKAGE_NAME" \
  --package-name "$API_MODEL_PACKAGE_NAME" \
  --global-property models,modelTests=false,modelDocs=false \
  -g go

( cat tmp/"$API_MODEL_PACKAGE_NAME"/model_*.go | \
  go run postprocess.go "$API_MODEL_PACKAGE_NAME" > "../../internal/$API_MODEL_PACKAGE_NAME/apimodel.go" || \
  (rm -rf tmp && exit 1) \
); rm -rf tmp

gofmt -l -w "../../internal/$API_MODEL_PACKAGE_NAME/apimodel.go"
