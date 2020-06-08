package decoder

import (
	"errors"
	"fmt"

	"github.com/ofauchon/rfskipper-adapter/core"
)

const prologuePulseLengthMin int = 70
const prologuePulseLengthMax int = 75
const prologueBitstartPulseMin int = 450
const prologueBitstartPulseMax int = 550
const prologueBit0PulseMin int = 1500
const prologueBit0PulseMax int = 2500
const prologueBit1PulseMin int = 3500
const prologueBit1PulseMax int = 4500

// PrologueDecoder provides code to decide Prologue signals
type PrologueDecoder struct {
	decoderName string
}

// NewPrologueDecoder returns a pointer to a new  PrologueDecoder object
func NewPrologueDecoder() *PrologueDecoder {
	d := new(PrologueDecoder)
	d.decoderName = "prologue"
	return d
}

//GetDecoderName returns decoder name
func (d *PrologueDecoder) GetDecoderName() string {
	return d.decoderName
}

func (d *PrologueDecoder) getBit(pt core.PulseTrain, pos int) (bool, error) {

	fmt.Println("prologue: Start")

	if pt.Pulses[pos] < prologueBitstartPulseMin ||
		pt.Pulses[pos] > prologueBitstartPulseMax {
		return false, fmt.Errorf("Bits must start with high pulse (%dms-%dms)", prologueBitstartPulseMin, prologueBitstartPulseMax)
	}
	fmt.Println("prologue: Good Length")

	if pt.Pulses[pos+1] > prologueBit0PulseMin &&
		pt.Pulses[pos+1] < prologueBit0PulseMax {
		return false, nil
	} else if pt.Pulses[pos+1] > prologueBit1PulseMin ||
		pt.Pulses[pos+1] < prologueBit1PulseMax {
		return true, nil
	} else {
		return false, fmt.Errorf("Wrong low pulse duration:")
	}
}

// Decode tries to decode PulseTrain
func (d *PrologueDecoder) Decode(pt core.PulseTrain) (error, string) {

	//fmt.Println("PulseLen: ", len(pt.Pulses))
	//fmt.Println("Pulse0: ", pt.Pulses[0])

	// Checks if received signal looks like a Prologue signal
	if len(pt.Pulses) < prologuePulseLengthMin ||
		len(pt.Pulses) > prologuePulseLengthMax ||
		pt.Pulses[0] < prologueBitstartPulseMin ||
		pt.Pulses[0] > prologueBitstartPulseMax {
		return errors.New("Invalid Pluses number or duration"), ""
	}

	// Read nibbles
	var nibbles [7]uint8

	for i := 0; i < len(pt.Pulses)-1 && (i/8) < len(nibbles); i = i + 2 {
		bit, err := d.getBit(pt, i)
		if err != nil {
			return err, ""
		}

		if i > 0 {
			nibbles[i/8] = nibbles[i/8] << 1
		}

		if bit {
			nibbles[i/8] |= 1
		}

	}
	/*
		for i, ni := range nibbles {
			fmt.Printf("Nibble %d, %b (%x)\n", i, ni, ni)
		}
	*/
	dType := nibbles[0]
	dID := ((uint8(nibbles[1]) & 0x0F) << 4) | nibbles[2]
	dBat := nibbles[3] & 0x08
	dTemp := uint16(nibbles[4])
	dTemp = dTemp<<4 + uint16(nibbles[5])
	dTemp = dTemp<<4 + uint16(nibbles[6])

	fmt.Printf("Prologue => Type:%d Id:%d Bat:%d Temp:%d\n", dType, dID, dBat, dTemp)

	/*
		uint8_t u8_Type = pu8_nibble[0];
		uint8_t u8_Id = ((pu8_nibble[1] & 0x0F) << 4) | pu8_nibble[2];
		uint8_t u8_Bat = pu8_nibble[3] & 0x08;
		uint16_t u16_Temp = pu8_nibble[4];
		u16_Temp = (u16_Temp << 4) + pu8_nibble[5];
		u16_Temp = (u16_Temp << 4) + pu8_nibble[6];
	*/
	return nil, "PrologueDecoder: End processing"
}
