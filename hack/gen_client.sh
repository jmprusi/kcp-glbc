#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

export GOPATH=$(go env GOPATH)

SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
pushd "${SCRIPT_ROOT}"
BOILERPLATE_HEADER="$( pwd )/hack/boilerplate/boilerplate.go.txt"
popd
CODEGEN_PKG=${CODEGEN_PKG:-$(cd "${SCRIPT_ROOT}"; go list -f '{{.Dir}}' -m k8s.io/code-generator)}

echo "Generating Go client code..."

bash "${CODEGEN_PKG}"/generate-groups.sh "deepcopy,client,informers,listers" \
  github.com/kuadrant/kcp-glbc/pkg/client/kuadrant github.com/kuadrant/kcp-glbc/pkg/apis \
  "kuadrant:v1" \
  --go-header-file="${BOILERPLATE_HEADER}" \
  --output-base=${SCRIPT_ROOT} \
  --trim-path-prefix=github.com/kuadrant/kcp-glbc
pushd ./pkg/apis

${CODE_GENERATOR} \
  "client:outputPackagePath=github.com/kuadrant/kcp-glbc/pkg/client/kuadrant,apiPackagePath=github.com/kuadrant/kcp-glbc/pkg/apis,singleClusterClientPackagePath=github.com/kuadrant/kcp-glbc/pkg/client/kuadrant/clientset/versioned,headerFile=${BOILERPLATE_HEADER}" \
  "lister:apiPackagePath=github.com/kuadrant/kcp-glbc/pkg/apis,headerFile=${BOILERPLATE_HEADER}" \
  "informer:outputPackagePath=github.com/kuadrant/kcp-glbc/pkg/client/kuadrant,singleClusterClientPackagePath=github.com/kuadrant/kcp-glbc/pkg/client/kuadrant/clientset/versioned,apiPackagePath=github.com/kuadrant/kcp-glbc/pkg/apis,headerFile=${BOILERPLATE_HEADER}" \
  "paths=./..." \
  "output:dir=./../client/kuadrant"
popd

go install "${CODEGEN_PKG}"/cmd/openapi-gen

"${GOPATH}"/bin/openapi-gen --input-dirs github.com/kuadrant/kcp-glbc/pkg/apis/kuadrant/v1 \
  --output-package  github.com/kuadrant/kcp-glbc/pkg/openapi -O zz_generated.openapi \
  --go-header-file ${BOILERPLATE_HEADER} \
  --output-base "../" \
  --trim-path-prefix github.com/kuadrant/kcp-glbc