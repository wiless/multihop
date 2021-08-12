package main

import (
	"log"
	"os"

	"github.com/jszwec/csvutil"
)

// ID  ,X            ,Y            ,Z         ,TxPowerdBm ,Hdirection ,VTilt ,Active ,Alias
type BSlocation struct {
	ID                                     int
	X, Y, Z, TxPowerdBm, Hdirection, VTilt float64
	Active, Alias                          int
}

type UElocation struct {
	ID            int
	X, Y, Z       float64
	Indoor, InCar bool
	BSdist        float64
	GCellID       int

	// Name string `csv:"name"`
	// Address
}

// RxNodeID,FreqInGHz,BandwidthMHz,N0,RSSI,BestRSRP,BestRSRPNode,BestSINR,RoIDbm,BestCouplingLoss,MaxTxAg,MaxRxAg,AssoTxAg,AssoRxAg,MaxTransmitBeamID
type SLSprofile struct {
	RxNodeID                                                                 int
	FreqInGHz, BandwidthMHz, N0, RSSI, BestRSRP                              float64
	BestRSRPNode                                                             int
	BestSINR, RoIDbm, BestCouplingLoss, MaxTxAg, MaxRxAg, AssoTxAg, AssoRxAg float64
	MaxTransmitBeamID                                                        int
	BestULsinr                                                               float64
}

type LinkProfile struct {
	RxNodeID                                                                                                              int
	TxID                                                                                                                  int
	Distance                                                                                                              float64
	IndoorDistance, UEHeight                                                                                              float64
	IsLOS                                                                                                                 bool
	CouplingLoss, Pathloss, O2I, InCar, ShadowLoss, TxPower, BSAasgainDB, UEAasgainDB, TxGCSaz, TxGCSel, RxGCSaz, RxGCSel float64
}

type RelayNode struct {
	UElocation
	FrequencyGHz float64
}

func LoadSLSprofile(fname string) []SLSprofile {
	// fname += ".csv"

	fid, err := os.Open(fname)
	er(err)

	data, err := os.ReadFile(fname)
	er(err)

	var sls []SLSprofile
	err = csvutil.Unmarshal(data, &sls)
	er(err)

	defer fid.Close()

	fid.Close()
	return sls
}

var er = func(err error) {
	if err != nil {
		log.Println("Error ", err)
	}
}

//LoadUELocations saves the UE locations to fname in csv format
// uid,x,y,z,cellid,gdistance,in/out, car
func LoadUELocations(fname string) []UElocation {
	// fname += ".csv"
	fid, err := os.Open(fname)
	er(err)

	data, err := os.ReadFile(fname)
	er(err)

	var ues []UElocation
	err = csvutil.Unmarshal(data, &ues)
	er(err)

	defer fid.Close()

	fid.Close()
	return ues
}
