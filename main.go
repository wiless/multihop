package main

import (
	"flag"
	"fmt"

	"github.com/5gif/config"
	"github.com/wiless/vlib"
)

// func main() {
// 	log.Println("Starting..")
// 	//	LoadUELocations("uelocation.csv")

// }
var v3 vlib.VectorIface
var basedir = "N500/"
var myues []UElocation
var BW float64 // Can be different than itucfg.BandwidthMHz, based on uplink/downlink
var RxNoisedB float64
var itucfg config.ITUconfig
var bslocs []BSlocation

var NBs int
var N0 float64 // N0 in linear scale

func init() {
	flag.StringVar(&basedir, "basedir", "N500/", "Prefix for result files, use as -basedir=results/")
	flag.Parse()
}

func main() {
	itucfg, _ = config.ReadITUConfig(basedir + "itu.cfg")
	// ----
	LoadCSV("bslocation.csv", &bslocs) // needed ?
	NBs = len(bslocs)

	BW = itucfg.BandwidthMHz
	RxNoisedB = itucfg.UENoiseFigureDb         // For Downlink
	N0dB := -174 + vlib.Db(BW*1e6) + RxNoisedB // in linear scale
	N0 = vlib.InvDb(N0dB)
	fmt.Println("N0 (dB)", N0dB)

	// SplitUELocationsByCell(basedir + "uelocation.csv")

	CreateSLS(basedir+"newsls.csv", basedir+"linkproperties.csv", true)            // Regenerate SLS full
	CreateSLS(basedir+"newsls-mini.csv", basedir+"linkproperties.csv", false)      // Regenerate SLS mini
	SplitSLSprofileByCell(basedir+"newsls-mini", basedir+"newsls-mini.csv", false) // Split SLS by Cell

	CreateMiniLinkProfiles(basedir+"linkproperties-mini.csv", basedir+"linkproperties.csv")
	SplitLinkProfilesByCell(basedir+"linkmini", basedir+"linkproperties.csv", false)

}

/*
func xmain() {
	rand.Seed(time.Now().Unix())
	fmt.Println("Starting..")
	// var ivec vlib.GIntVector
	// var ivec = []UElocation{{ID: 0, X: -100}, {ID: 1, X: 200}, {ID: 2, X: 200}, {ID: 3, X: 200}, {ID: 4, X: 200}}

	ues := LoadUELocations("uelocation.csv")

	// slsprofile := LoadSLSprofile("slsprofile.csv")

	var filteredues []UElocation
	myfunc := func(ue UElocation) bool {
		return ue.GCellID == 0
	}
	filteredues = d3.Filter(ues, myfunc).([]UElocation)
	_ = filteredues
	// fmt.Println("Matching devices",
	// 	d3.FilterIndex(ues, myfunc))
	fmt.Printf("\n\n Outdoor Gcell %v | %d", d3.FlatMap(filteredues, "ID"), len(filteredues))

	splitSLSprofile(ues)

	// fmt.Printf("\nSearch High SINR")

	// type Relay struct {
	// 	UElocation
	// 	Relay bool
	// 	SLSprofile
	// }
	// var relays []Relay
	// // type UEinfo struct {
	// // 	ID int
	// // 	GCellID int
	// // }
	// // uelookup := d3.Map(ues, func(ue UElocation)  {

	// // }

	// newues := d3.Filter(filteredues, func(ue UElocation) bool {
	// 	var findindex int
	// 	findindex = d3.FindFirstIndex(slsprofile, func(indx int, sls SLSprofile) bool {
	// 		valid := sls.BestSINR > 15 && sls.RxNodeID == ue.ID
	// 		if valid {
	// 			fmt.Printf("..")
	// 			sls.FreqInGHz += float64(rand.Intn(4))
	// 			relays = append(relays, Relay{ue, true, sls})
	// 		}
	// 		return valid
	// 	})
	// 	return (findindex >= 0)
	// }).([]UElocation)

	// // fmt.Printf("\n\n SINR %#v", relays)
	// // str, _ := csvutil.Header(UElocation{}, "")
	// fid, _ := os.Create("relays.csv")
	// w := csv.NewWriter(fid)
	// enc := csvutil.NewEncoder(w)
	// enc.EncodeHeader(Relay{})
	// for _, v := range relays {
	// 	enc.Encode(v)
	// }
	// w.Flush()

	// fmt.Printf("\n\n RelayID %v", d3.FlatMap(newues, "ID"))
	// fmt.Println("\n\n done....")
}
*/
