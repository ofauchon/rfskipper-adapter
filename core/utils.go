package core

// PulseTrain structure describe RF 0/1 signal over time
type PulseTrain struct {
	ID     int   `json:"id"`
	Count  int   `json:"count"`
	Pulses []int `json:"pulses"`
}
