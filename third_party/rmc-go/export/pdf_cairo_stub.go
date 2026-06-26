// +build !cairo

package export

import (
	"fmt"
	"io"

	"github.com/joagonca/rmc-go/parser"
)

// ExportToPDFCairo is a stub when Cairo is not available
func ExportToPDFCairo(tree *parser.SceneTree, w io.Writer) error {
	return fmt.Errorf("native PDF export not available: binary was not built with Cairo support\n" +
		"To use --native flag, rebuild with: make build-cairo\n" +
		"Or use the default Inkscape-based export without --native flag")
}

// ExportToMultipagePDFCairo is a stub when Cairo is not available
func ExportToMultipagePDFCairo(trees []*parser.SceneTree, w io.Writer) error {
	return fmt.Errorf("native multipage PDF export not available: binary was not built with Cairo support\n" +
		"To use --native flag, rebuild with: make build-cairo\n" +
		"Or use the default Inkscape-based export without --native flag")
}
