package main

import (
	"math"
	"math/rand"
)

var hUT = 1.5

func IsLOS(distance2d float64) bool {
	if distance2d < 18 {
		return true
	}

	prlos := 18/distance2d + math.Exp(-distance2d/36)*(1-18/distance2d)
	if rand.Float64() <= prlos {
		return true
	}
	return false

}

func PLNLOS(d2D, fcghz, hBS float64) float64 {
	var d3Distance = math.Sqrt(math.Pow((hBS-hUT), 2) + math.Pow(d2D, 2))
	// var ddBP = BPDist(fcghz, hBS)
	d3Distance = math.Sqrt(math.Pow((hBS-hUT), 2) + math.Pow(d2D, 2))

	var LOS = PL(d2D, fcghz, hBS)

	var PLN = 35.4*math.Log10(d3Distance) +
		22.4 +
		21.3*math.Log10(fcghz) -
		0.3*(hUT-1.5)

	return math.Max(LOS, PLN)
}
func PL(d2D, fcghz, hBS float64) float64 {
	var d3Distance = math.Sqrt(math.Pow((hBS-hUT), 2) + math.Pow(d2D, 2))
	var ddBP = BPDist(fcghz, hBS)
	d3Distance = math.Sqrt(math.Pow((hBS-hUT), 2) + math.Pow(d2D, 2))
	var PL1 = 32.4 + 21*math.Log10(d3Distance) + 20*math.Log10(fcghz)
	var PL2 = 32.4 +
		40*math.Log10(d3Distance) +
		20*math.Log10(fcghz) -
		9.5*math.Log10(math.Pow(ddBP, 2)+math.Pow((hBS-hUT), 2))

	if ddBP <= d2D {
		return PL2
	} else {
		return PL1
	}
}

func BPDist(fcghz, hBS float64) float64 {
	var fcHz = fcghz * 1e9
	var hE = 1.0
	var hdBS = hBS - hE
	var hdUT = hUT - hE
	var C = 3.0 * 1e8
	return (4 * hdBS * hdUT * fcHz) / C
}
