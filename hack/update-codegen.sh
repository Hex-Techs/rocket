#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
sed -i 's#// ##' ${SCRIPT_ROOT}/hack/tools.go
go mod tidy
go mod vendor
CODEGEN_PKG=${CODEGEN_PKG:-$(cd "${SCRIPT_ROOT}"; ls -d -1 ./vendor/k8s.io/code-generator 2>/dev/null || echo ../code-generator)}

Alpha_API_BASE="${SCRIPT_ROOT}"/api/rocket/v1alpha1

mkdir -p ${Alpha_API_BASE}
cp ${SCRIPT_ROOT}/api/v1alpha1/* ${Alpha_API_BASE}

# generate the code with:
# --output-base    because this script should also be able to run inside the vendor dir of
#                  k8s.io/kubernetes. The output-base is needed for the generators to output into the vendor dir
#                  instead of the $GOPATH directly. For normal projects this can be dropped.
bash "${CODEGEN_PKG}"/generate-groups.sh "client,lister,informer" \
   github.com/hex-techs/rocket/client github.com/hex-techs/rocket/api \
   "rocket:v1alpha1" \
   --go-header-file "${SCRIPT_ROOT}"/hack/boilerplate.go.txt \
   --output-base "$(dirname "${BASH_SOURCE[0]}")/../../../.."

if [ -d "${Alpha_API_BASE}" ]; then
   rm -rf ${SCRIPT_ROOT}/api/rocket
fi

find client -type f -name "*.go" | xargs sed -i".out" -e "s#github.com/hex-techs/rocket/api/rocket/v1alpha1#github.com/hex-techs/rocket/api/v1alpha1#g"
find client -type f -name "*go.out" | xargs rm -rf

sed -i  's#import#// &#' ${SCRIPT_ROOT}/hack/tools.go
rm -rf ${SCRIPT_ROOT}/vendor
