package decoder

import (
	"github.com/ofauchon/rfskipper-adapter/core"
)

type Decoder interface {
	GetDecoderName() string
	Decode(pt core.PulseTrain) (error, string)
}
