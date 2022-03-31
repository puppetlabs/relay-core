// +build tools

package tools

import (
	_ "github.com/google/ko"
	_ "github.com/google/wire/cmd/wire"
	_ "gotest.tools/gotestsum"
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen"
)
