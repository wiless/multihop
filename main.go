package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/5gif/config"
	"github.com/schollz/progressbar"
	"github.com/wiless/d3"
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
	if !(strings.HasSuffix(basedir, "/") || strings.HasSuffix(basedir, "\\")) {
		basedir += "/"
	}
	// rand.Seed(time.Now().Unix())
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

	PrepareInputFiles()

	// Finding relays
	FindRelays(basedir+"relaylocations.csv", 0.01) // 1% as relays
	//BS Information
	bsalias := d3.FlatMap(bslocs, "Alias")
	_ = bsalias
	// fmt.Printf("\nBSLocation %#v ", bsalias)

}

func Shuffle(inparray interface{}) {

}

// Assuming

type RelayNode struct {
	UElocation
	FrequencyGHz float64
}

func FindRelays(fname string, percentage float64) {

	fd, err := os.Create(fname)
	er(err)
	defer fd.Close()
	fmt.Printf("\nFindRelay : %s\n", fname)

	header, _ := vlib.Struct2HeaderLine(RelayNode{})
	fd.WriteString(header)

	NRelayPerCell := int(float64(itucfg.NumUEperCell) * percentage)
	NCells := 19
	// TotalRelays := NRelayPerCell * NCells
	NRelayChannels := 4 /// = Total channels -1  (Channel 0 reserved for BS)

	baseuefile := basedir + "uelocation"
	baseslsfile := basedir + "newsls-mini"

	// cellbar := progressbar.New(NCells)
	cellbar := progressbar.Default(int64(NCells), "Working on Cell")

	for cell := 0; cell < NCells; cell++ {
		ues := LoadUELocations(baseuefile + fmt.Sprintf("-cell%02d.csv", cell))
		outdoorues := d3.Filter(ues, func(ue UElocation) bool {
			return !ue.Indoor
		}).([]UElocation)
		outdooruesids := d3.FlatMap(outdoorues, "ID")

		sls := LoadSLSprofile(baseslsfile + fmt.Sprintf("-cell%02d.csv", cell))
		var potential vlib.VectorI
		relaysls := d3.Filter(sls, func(indx int, s SLSprofile) bool {
			found := false
			if s.BestSINR > 10 {
				found, _ = vlib.Contains(outdooruesids, s.RxNodeID)
				if found {
					sls[indx].FreqInGHz = -1
					potential.AppendAtEnd(s.RxNodeID)
				}

			}
			return found

		})
		_ = relaysls

		_, indx := vlib.RandUFVec(len(outdoorues)).Sorted2()

		relays := make([]RelayNode, NRelayPerCell)
		for i := 0; i < NRelayPerCell; i++ {

			relays[i].UElocation = outdoorues[indx[i]]
			// Unassigned Beacon Frequency =-1
			relays[i].FrequencyGHz = -1

			// random between 1 to 4, ZERO is for basestation
			// relays[i].FrequencyGHz = float64(rand.Intn(NRelayChannels)) + 1
		}

		/// UNCOMMENT -- START to DISABLE OPTIMAL assignment
		// APPROACH FOR OPTIMIZED Frequency Assignment
		/// Assign Frequency | Optimize Frequency Assignment based on distance..
		NclustersPerFrequency := math.Floor(float64(NRelayPerCell) / float64(NRelayChannels))
		extraCluster := NRelayPerCell % NRelayChannels

		for f := 1; f <= NRelayChannels; f++ {

			var NumRelaysInF = int(NclustersPerFrequency) + extraCluster
			// for c := 0; c < NumRelaysInF; c++ {
			// fmt.Printf("\n\n Freq=%d | Relays in this Freq %d", f, NumRelaysInF)
			// if c == 0 {
			srcindx := d3.FindFirstIndex(relays, func(r RelayNode) bool {
				return r.FrequencyGHz == -1 // Find the first UNASSIGNED relay
			})

			relays[srcindx].FrequencyGHz = float64(f)
			src := relays[srcindx]
			// fmt.Printf("==> first relay %#v ", src)
			// Returns true if the current relay node is the Farthest from firstrelay
			distance := make([]float64, (NRelayPerCell))
			srcloc := vlib.Location3D{src.X, src.Y, src.Z}
			for iter, v := range relays {
				var d float64
				d = math.Inf(+1)
				if v.ID != src.ID && v.FrequencyGHz == -1 { /// If not src!=dest and already no frequency assigned
					destloc := vlib.Location3D{v.X, v.Y, v.Z}
					d = srcloc.DistanceFrom(destloc)
				}
				distance[iter] = d
			}
			sdistance, sindex := vlib.Sorted(distance)

			cnt := 0
			for d := sdistance.Len() - 1; d >= 0; d-- {
				if !math.IsInf(sdistance[d], +1) && cnt < NumRelaysInF-1 {
					relays[sindex[d]].FrequencyGHz = float64(f)
					cnt++
				}
			}

			if extraCluster > 0 {
				extraCluster--
			}
		}
		/// UNCOMMENT -- END to DISABLE OPTIMAL assignment
		ids := d3.FlatMap(relays, "ID").([]int)
		sids, idx := vlib.SortedI(ids)
		_ = sids
		for _, v := range idx {
			str, _ := vlib.Struct2String(relays[v])
			fd.WriteString("\n" + str)
			// fmt.Printf("\n Random %#v", v)
		}

		cellbar.Add(1)
	}

}

func PrepareInputFiles() {
	// for i := 0; i < 100; i++ {

	SplitUELocationsByCell(basedir + "uelocation.csv")
	CreateSLS(basedir+"newsls.csv", basedir+"linkproperties.csv", true)            // Regenerate SLS full
	CreateSLS(basedir+"newsls-mini.csv", basedir+"linkproperties.csv", false)      // Regenerate SLS mini
	SplitSLSprofileByCell(basedir+"newsls-mini", basedir+"newsls-mini.csv", false) // Split SLS by Cell
	CreateMiniLinkProfiles(basedir+"linkproperties-mini.csv", basedir+"linkproperties.csv")
	SplitLinkProfilesByCell(basedir+"linkmini", basedir+"linkproperties.csv", false)

	// }
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
