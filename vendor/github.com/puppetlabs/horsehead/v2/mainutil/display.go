package mainutil

import (
	"fmt"
	"io"
	"os"

	"github.com/puppetlabs/errawr-go/v2/pkg/errawr"
)

func ExitWithCLIError(w io.Writer, code int, err errawr.Error) {
	fmt.Fprintln(w, err.FormattedDescription())

	os.Exit(code)
}
