package decoder

import (
	"errors"
	"fmt"

	"github.com/ofauchon/rfskipper-adapter/core"
)

const PulseTrainMinLen int = 295
const PulseTrainMaxLen int = 305
const BitMinDuration int = 190
const BitMaxDuration int = 220

// TicPulsesV2Decoder provides code to decide Prologue signals
type TicPulsesV2Decoder struct {
	decoderName string
}

// NewTicPulsesV2Decoder returns a pointer to a new  TicPulsesV2Decoder object
func NewTicPulsesV2Decoder() *TicPulsesV2Decoder {
	d := new(TicPulsesV2Decoder)
	d.decoderName = "ticpulsev2"
	return d
}

//GetDecoderName returns decoder name
func (d *TicPulsesV2Decoder) GetDecoderName() string {
	return d.decoderName
}

func (d *TicPulsesV2Decoder) ReadBytesFromBits(pBits []uint8, pStartBit int, pLength int) []uint8 {

	var ret []uint8
	const bitPerByte = 8

	if pStartBit+(pLength*bitPerByte) < len(pBits) {
		for i := 0; i < pLength; i++ {
			current := uint8(0)
			for j := 0; j < 8; j++ {
				if j > 0 {
					current = current << 1
				}
				if pBits[pStartBit+i*bitPerByte+j] == 1 {
					current |= 1
				}
			}
			ret = append(ret, current)
		}

	}
	return ret
}

func (d *TicPulsesV2Decoder) DecodeManchesterBits(pBits []uint8) []uint8 {
	var ret []uint8
	cnt := uint8(0)
	r := uint8(0)

	for i := 0; i < len(pBits)-1; i = i + 2 {
		b1 := pBits[i]
		b2 := pBits[i+1]
		if b1 == b2 {
			fmt.Printf("2 identical bits at position %d, forbidden in manchester\n", i)
			return ret
		} else if b1 == 1 {
			r |= 1
		}

		if cnt == 7 {
			ret = append(ret, r)
			r = 0
			cnt = 0
		} else {
			r = r << 1
			cnt++
		}

	}
	return ret
}

func (d *TicPulsesV2Decoder) PulsesToBits(pPulses []int, pBitDuration int) []uint8 {

	var ret []uint8

	state := uint8(1)
	for pos := 0; pos < len(pPulses); pos++ {
		nbits := pPulses[pos] / pBitDuration

		for k := 0; k < nbits; k++ {
			ret = append(ret, state)
		}
		state = state ^ 1
	}
	return ret
}

// Decode processes given PulseTrain
func (d *TicPulsesV2Decoder) Decode(pt core.PulseTrain) (error, string) {

	if len(pt.Pulses) < PulseTrainMinLen || len(pt.Pulses) > PulseTrainMaxLen {
		return errors.New("Invalid Pluses number or duration"), ""
	}

	// Check for sync bytes
	for i := 0; i < 10; i++ {
		if pt.Pulses[i] < BitMinDuration ||
			pt.Pulses[i] > BitMaxDuration {
			fmt.Println("ticpulsev2:Invalid preamble")
			return errors.New("Invalid sync bits length"), ""
		}
	}

	// Start decoding
	bits := d.PulsesToBits(pt.Pulses, BitMinDuration)
	sync := d.ReadBytesFromBits(bits, 0, 3)
	if sync[0] != 0xAA || sync[0] != 0xAA || sync[0] != 0xAA {
		return errors.New("Invalid Invalid sync signature (!=0xAAAA9C)"), ""
	}
	fmt.Println("ticpulsesv2: 3 first sync bytes OK")

	decoded := d.DecodeManchesterBits(bits[24:])
	//	decoded = decoded[1:] // Remove first bit (size)

	for _, v := range decoded[0:20] {
		fmt.Printf("%02X ", v)
	}

	// Ex: 12 01    07 00 9F 6D 39   11   D8 F3 A8 01   00 00 00 00   1C 02    02 BF
	// 12 => Taille du packet
	// 01 => Protocole (01 Teleinfo historique)
	// 07 00 9F 6D 39 => Compteur #1 Réel: 07 39 6D 9F 00 => Numérique: 31028256512
	// 11 => Type abo
	// 4 octets valeur compteur (D8 F3 A8 01)
	// 4 octets valeur second compteur (00 00 00 00)
	// 2 octets PAAP (1C 02) => 021C => 540
	// 1 octet status (02 => PAAP Valide)

	id := uint64(decoded[2])
	id = (id << 8) | uint64(decoded[6])
	id = (id << 8) | uint64(decoded[5])
	id = (id << 8) | uint64(decoded[4])
	id = (id << 8) | uint64(decoded[3])

	cntr := uint32(decoded[10])
	cntr = (cntr << 8) | uint32(decoded[9])
	cntr = (cntr << 8) | uint32(decoded[8])
	cntr = (cntr << 8) | uint32(decoded[7])

	paap := uint16(decoded[17]) << 8
	paap = paap | uint16(decoded[16])

	fmt.Printf("ticpulsev2: id: %d, cntr: %d, paap: %d\n", id, cntr, paap)

	return nil, "TicPulsesV2Decoder: End processing"
}
