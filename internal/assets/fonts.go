package assets

import (
	"bytes"
	"log"

	_ "embed"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

//go:embed fonts/AgaveNerdFont-Regular.ttf
var agaveFontData []byte

var (
	AgaveFaceSource *text.GoTextFaceSource
)

func init() {
	source, err := text.NewGoTextFaceSource(bytes.NewReader(agaveFontData))
	if err != nil {
		log.Fatalf("failed to parse font: %v", err)
	}
	AgaveFaceSource = source
}

// GetFont returns a configured face for the embedded Agave font.
func GetFont(size float64) *text.GoTextFace {
	return &text.GoTextFace{
		Source: AgaveFaceSource,
		Size:   size,
	}
}
