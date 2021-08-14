package main

import (
	"flag"
	"fmt"
	"os"

	"math/rand"
	"strings"
	"time"

	"github.com/5gif/config"
	log "github.com/sirupsen/logrus"
	"github.com/wiless/d3"
	"github.com/wiless/vlib"
)

// func main() {
// 	log.Println("Starting..")
// 	//	LoadUELocations("uelocation.csv")

// }

var basedir = "./"
var myues []UElocation
var BW float64 // Can be different than itucfg.BandwidthMHz, based on uplink/downlink
var RxNoisedB, TxNoisedB float64
var itucfg config.ITUconfig
var simcfg config.SIMconfig
var bslocs []BSlocation
var bsTxPowerdBm, ueTxPowerdBm float64
var NBs int
var ActiveBSCells int
var N0 float64      // N0 in linear at UE receiver
var N0dB float64    // N0 in dB scale at UE receiver
var UL_N0 float64   // Uplink N0 linear at the BS receiver
var UL_N0dB float64 // Uplink N0 dBm at the BS receiver

var Er = func(err error) {
	if err != nil {
		log.Println("Error ", err)
	}
}
var err error

var GENERATE bool
var FINDRELAYS bool

func init() {
	flag.StringVar(&basedir, "basedir", "N500/", "Prefix for result files, use as -basedir=results/")
	flag.BoolVar(&GENERATE, "generate", false, "Generate files ? usage as -GENERATE true")
	flag.BoolVar(&FINDRELAYS, "relays", false, "Identify relays ? usage as -relays true")
	flag.Parse()
	if !(strings.HasSuffix(basedir, "/") || strings.HasSuffix(basedir, "\\")) {
		basedir += "/"
	}
	rand.Seed(time.Now().Unix())
	log.Info("Setting BASEDIR ", basedir)
	log.Info("Generating Enabled ", GENERATE)
	log.Info("Identify Relays  ", FINDRELAYS)
}

func loadSysParams() {
	simcfg, err = config.ReadSIMConfig(basedir + "sim.cfg")
	er(err)
	ActiveBSCells = simcfg.ActiveBSCells
	fmt.Println("Active BSCells = ", simcfg.ActiveBSCells)
	fmt.Println("Active UECells = ", simcfg.ActiveUECells)
	itucfg, _ = config.ReadITUConfig(basedir + "itu.cfg")
	// ----
	d3.CSV(basedir+"bslocation.csv", &bslocs) // needed ?
	NBs = len(bslocs)

	BW = itucfg.BandwidthMHz
	RxNoisedB = itucfg.UENoiseFigureDb // For Downlink
	TxNoisedB = itucfg.BSNoiseFigureDb // For Uplink

	N0dB = -174 + vlib.Db(BW*1e6) + RxNoisedB // in linear scale
	N0 = vlib.InvDb(N0dB)

	UL_N0dB = -174 + vlib.Db(BW*1e6) + TxNoisedB // in linear scale
	UL_N0 = vlib.InvDb(UL_N0dB)

	bsTxPowerdBm = itucfg.TxPowerDbm
	ueTxPowerdBm = itucfg.UETxDbm

	fmt.Println("Total Active Sectors ", NBs)

	fmt.Println("DL : N0 (dB)", N0dB)
	fmt.Println("UL : N0 (dB)", UL_N0dB)
}

func main() {
	loadSysParams()
	if GENERATE {
		PrepareInputFiles()
	}

	CreateSINRTable()

	if FINDRELAYS {
		// Finding relays
		relays := FindRelays(basedir+"relaylocations.csv", 0.01, 0) // 1% as relays

		//BS Information
		// bsalias := d3.FlatMap(bslocs, "Alias")
		// _ = bsalias
		var ues []UElocation
		d3.CSV(basedir+"uelocation-cell00.csv", &ues)
		relaysls := GenerateRelayLinkProps(relays, ues)
		// minisls := make(map[int]SLSprofile)
		fd, _ := os.Create(basedir + "hybridnewsls-mini.csv")
		msls := d3.SubStruct(SLSprofile{}, "RxNodeID", "BestRSRPNode", "FreqInGHz", "BestCouplingLoss", "BestSINR", "BestULsinr", "AssoTxAg", "AssoRxAg")
		header, _ := vlib.Struct2HeaderLine(msls)
		fd.WriteString(header)
		d3.ForEachParse(basedir+"newsls-mini.csv", func(s SLSprofile) {
			rsls, ok := relaysls[s.RxNodeID]
			if ok {
				if rsls.BestSINR > s.BestSINR {
					msls := d3.SubStruct(rsls, "RxNodeID", "BestRSRPNode", "FreqInGHz", "BestCouplingLoss", "BestSINR", "BestULsinr", "AssoTxAg", "AssoRxAg")
					str, _ := vlib.Struct2String(msls)
					fd.WriteString("\n" + str)
				}else{
					s.FreqInGHz = 0
					msls := d3.SubStruct(s, "RxNodeID", "BestRSRPNode", "FreqInGHz", "BestCouplingLoss", "BestSINR", "BestULsinr", "AssoTxAg", "AssoRxAg")
					str, _ := vlib.Struct2String(msls)
					fd.WriteString("\n" + str)					
				}
				return
			}
			s.FreqInGHz = 0
			msls := d3.SubStruct(s, "RxNodeID", "BestRSRPNode", "FreqInGHz", "BestCouplingLoss", "BestSINR", "BestULsinr", "AssoTxAg", "AssoRxAg")
			str, _ := vlib.Struct2String(msls)
			fd.WriteString("\n" + str)
		})

		
		// compare

	}

}

func PrepareInputFiles() {

	SplitUELocationsByCell(basedir + "uelocation.csv")
	CreateSLS(basedir+"newsls.csv", basedir+"linkproperties.csv", true)       // Regenerate SLS full
	CreateSLS(basedir+"newsls-mini.csv", basedir+"linkproperties.csv", false) // Regenerate SLS mini

	SplitSLSprofileByCell(basedir+"newsls-mini", basedir+"newsls-mini.csv", false)        // Split SLS by Cell
	SplitSLSprofileByAssociation(basedir+"newsls-mini", basedir+"newsls-mini.csv", false) // Split SLS by Cell
	CreateMiniLinkProfiles(basedir+"linkproperties-mini.csv", basedir+"linkproperties.csv")

	SplitLinkProfilesByCell(basedir+"linkmini", basedir+"linkproperties.csv", false, nil)
	CreateULInterferenceLinks(basedir + "linkproperties-mini-filtered.csv")

}

type LinkFiltered struct {
	RxNodeID, TxID int
	CouplingLoss   float64
	BestRSRPNode   int
}
