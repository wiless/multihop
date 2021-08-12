package main

import (
	"flag"
	"fmt"

	"math"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/5gif/config"
	"github.com/schollz/progressbar"
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
var N0, UL_N0 float64 // N0 in linear scale
var UL_N0dB float64

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

	N0dB := -174 + vlib.Db(BW*1e6) + RxNoisedB // in linear scale
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
		FindRelays(basedir+"relaylocations.csv", 0.01) // 1% as relays

		//BS Information
		// bsalias := d3.FlatMap(bslocs, "Alias")
		// _ = bsalias

		GenerateRelayLinkProps()
	}

}

func CreateSINRTable() {

	// Find Interfering Cells
	result := CreateICellInfo(basedir + "linkproperties-mini-filtered.csv")

	MeanIPerSectordBm := GetMeanInterference(result)

	NActive := rand.Intn(ActiveBSCells*3 - 1)
	// NActive = ActiveBSCells*3 - 1 // ???
	seq := vlib.NewSegmentI(0, ActiveBSCells*3)
	rand.Shuffle(seq.Len(), func(i, j int) {
		seq[i], seq[j] = seq[j], seq[i]
	})

	SnapShotIPerSectordBm := GetSnapShotInterference(result, seq[0:NActive]...)
	fmt.Printf("\n MeanIPerSectordBm     %#v", MeanIPerSectordBm)
	fmt.Printf("\n SnapShotIPerSectordBm %#v", SnapShotIPerSectordBm)
	fmt.Printf("\n Active I Sectors %d :  %#v\n", NActive, seq[0:NActive])

	// Cell 0 users..

	var Cell0Sec0Users, Cell0Sec1Users, Cell0Sec2Users vLinkFiltered
	Cell0Sec0Users = make(vLinkFiltered, 0)
	Cell0Sec1Users = make(vLinkFiltered, 0)
	Cell0Sec2Users = make(vLinkFiltered, 0)
	pbar := progressbar.Default(3 * int64(simcfg.ActiveUECells) * int64(itucfg.NumUEperCell))
	CenterCellusers := make(vLinkFiltered, 0)
	fn := func(l LinkFiltered) {

		if math.Mod(float64(l.BestRSRPNode), float64(ActiveBSCells)) == 0 && l.BestRSRPNode == l.TxID {
			CenterCellusers = append(CenterCellusers, l)

		}
		if l.BestRSRPNode == 0 && l.TxID == 0 {
			// fmt.Printf("Processing %v ", l)
			Cell0Sec0Users = append(Cell0Sec0Users, l)
		}
		if l.BestRSRPNode == ActiveBSCells && l.TxID == ActiveBSCells {
			Cell0Sec1Users = append(Cell0Sec1Users, l)
		}
		if l.BestRSRPNode == 2*ActiveBSCells && l.TxID == 2*ActiveBSCells {
			Cell0Sec2Users = append(Cell0Sec2Users, l)
		}
		pbar.Add(1)
	}

	d3.ForEachParse(basedir+"linkproperties-mini-filtered.csv", fn)

	// fmt.Printf("Cell 0 - Sector 0 Users  :%#v ", Cell0Sec0Users)
	// calcSINR := func(lp LinkFiltered) float64 {
	// 	// fmt.Printf("\n %#v", lp)
	// 	signal := lp.CouplingLoss + ueTxPowerdBm
	// 	inter := MeanIPerSectordBm[lp.BestRSRPNode]
	// 	_ = inter
	// 	SIR := signal - inter
	// 	return SIR
	// }
	// sinr0 := d3.Map(Cell0Sec0Users, calcSINR).([]float64)
	// sinr1 := d3.Map(Cell0Sec1Users, calcSINR).([]float64)
	// sinr2 := d3.Map(Cell0Sec2Users, calcSINR).([]float64)

	type SINRInfo struct {
		RxNodeID     int
		BestRSRPNode int
		SINRmean     float64
		SINRsnap     float64
		SINRideal    float64
	}

	fd, er := os.Create(basedir + "ulsinr.csv")
	defer fd.Close()
	fmt.Print(er)
	header, _ := vlib.Struct2HeaderLine(SINRInfo{})
	fd.WriteString(header)

	d3.ForEach(CenterCellusers, func(lp LinkFiltered) {
		signal := lp.CouplingLoss + ueTxPowerdBm

		SINRmean := signal - MeanIPerSectordBm[lp.BestRSRPNode]         // UL_N0dB need to be added
		SINRsnapshot := signal - SnapShotIPerSectordBm[lp.BestRSRPNode] // UL_N0dB need to be added
		SINRideal := signal - UL_N0dB
		info := SINRInfo{RxNodeID: lp.RxNodeID, BestRSRPNode: lp.BestRSRPNode, SINRmean: SINRmean, SINRsnap: SINRsnapshot, SINRideal: SINRideal}
		infostr, _ := vlib.Struct2String(info)
		fd.WriteString("\n" + infostr)
	})
}

func GetSnapShotInterference(linkinfo map[int]CellMap, activeSectors ...int) map[int]float64 {
	SnapShotIPerSectordBm := make(map[int]float64)
	for sector, _ := range linkinfo {
		// fmt.Printf("\n Current Sector %d", sector)
		var snapShotI float64
		SnapShotIPerSectordBm[sector] = -1000
		for _, k := range activeSectors {
			v, ok := linkinfo[sector][k]
			if ok && k != sector {
				// fmt.Printf("\nActive Interfering Sector %d , with %d  NUEs", k, len(v))
				picked := v[rand.Intn(len(v))]
				// fmt.Printf("\n Picked User %d | %v ", picked.RxNodeID, picked)
				// closs := picked.CouplingLoss
				snapShotI += vlib.InvDb(picked.CouplingLoss + itucfg.UETxDbm)
			}

		}
		if snapShotI != 0 {
			SnapShotIPerSectordBm[sector] = vlib.Db(snapShotI)
		}

		fmt.Printf("\n Sector %d :  SnapShotInterference   dBm = %v @ %v UETxpower  ", sector, SnapShotIPerSectordBm[sector], itucfg.UETxDbm)
	}
	return SnapShotIPerSectordBm

}

func GetMeanInterference(linkinfo map[int]CellMap) map[int]float64 {
	MeanIPerSectordBm := make(map[int]float64)
	for sector, _ := range linkinfo {
		fmt.Printf("\n Current Sector %d", sector)
		var meanI float64
		MeanIPerSectordBm[sector] = -1000
		for i := 0; i < NBs; i++ {
			k := i
			v, ok := linkinfo[sector][k]
			if ok && i%61 != 0 {
				// fmt.Printf("\nISector ID %d | NUEs = %v", k, len(v))
				closs := d3.Map(v, func(lf LinkFiltered) float64 {
					return lf.CouplingLoss + itucfg.UETxDbm

				}).([]float64)
				meanI += vlib.Mean(vlib.InvDbF(closs))
			}

		}

		if meanI != 0 {
			MeanIPerSectordBm[sector] = vlib.Db(meanI)
		}
		fmt.Printf("\n Sector %d : Mean Inteference dBm = %v  ", sector, MeanIPerSectordBm[sector])
	}
	return MeanIPerSectordBm

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

func CreateULInterferenceLinks(fname string) {

	slsprofile := LoadSLSprofile(basedir + "newsls-mini.csv")
	ActiveBSCells := simcfg.ActiveBSCells
	// selectedCell := 0
	fd, _ := os.Create(fname)
	// newsl := d3.SubStruct(LinkProfile{}, "RxNodeID", "TxID", "CouplingLoss")

	header, _ := vlib.Struct2HeaderLine(LinkFiltered{})
	fd.WriteString(header)
	pbar := progressbar.Default(int64(itucfg.NumUEperCell) * int64(NBs))

	fn := func(l LinkProfile) bool {
		gcell := (l.TxID%ActiveBSCells == 0) // 0,128,256 => GCELL 0 |  1,129,257=> GCELL=1 NBs/3-CELLS, NBs-Sectors

		if gcell {
			associatedBS := slsprofile[l.RxNodeID-NBs].BestRSRPNode
			//	if l.TxID != associatedBS {

			// fmt.Printf("\nData %v > SLS = %d?", l.RxNodeID, slsprofile[l.RxNodeID-383-1].RxNodeID)

			newsl := LinkFiltered{l.RxNodeID, l.TxID, l.CouplingLoss, associatedBS}
			// newsl := d3.SubStruct(l, "RxNodeID", "TxID", "CouplingLoss")
			str, _ := vlib.Struct2String(newsl)
			fd.WriteString("\n" + str)
			return true
			//}
		}
		pbar.Add(1)
		return false
	}
	d3.ForEachParse(basedir+"linkproperties-mini.csv", fn)

}

type vLinkFiltered []LinkFiltered
type CellMap map[int]vLinkFiltered

func CreateICellInfo(fname string) map[int]CellMap {
	result := make(map[int]CellMap)
	var Cell0Sec0, Cell0Sec1, Cell0Sec2 vLinkFiltered
	Cell0Sec0 = make(vLinkFiltered, 0)
	Cell0Sec1 = make(vLinkFiltered, 0)
	Cell0Sec2 = make(vLinkFiltered, 0)

	fmt.Println(3*int64(simcfg.ActiveUECells)*int64(itucfg.NumUEperCell), "to process")
	pbar := progressbar.Default(3 * int64(simcfg.ActiveUECells) * int64(itucfg.NumUEperCell))
	fn := func(l LinkFiltered) {
		if l.TxID == 0 && l.BestRSRPNode != 0 {
			// fmt.Printf("Processing %v ", l)

			Cell0Sec0 = append(Cell0Sec0, l)
		}
		if l.TxID == ActiveBSCells && l.BestRSRPNode != ActiveBSCells {
			Cell0Sec1 = append(Cell0Sec1, l)
		}
		if l.TxID == 2*ActiveBSCells && l.BestRSRPNode != 2*ActiveBSCells {
			Cell0Sec2 = append(Cell0Sec2, l)
		}
		pbar.Add(1)
	}

	d3.ForEachParse(fname, fn)

	result[0] = make(CellMap)
	d3.ForEach(Cell0Sec0, func(l LinkFiltered) {
		tmp := result[0][l.BestRSRPNode]
		tmp = append(tmp, l)
		result[0][l.BestRSRPNode] = tmp
	})

	result[ActiveBSCells] = make(CellMap)
	d3.ForEach(Cell0Sec1, func(l LinkFiltered) {
		tmp := result[ActiveBSCells][l.BestRSRPNode]
		tmp = append(tmp, l)
		result[ActiveBSCells][l.BestRSRPNode] = tmp
	})

	result[2*ActiveBSCells] = make(CellMap)
	d3.ForEach(Cell0Sec2, func(l LinkFiltered) {
		tmp := result[2*ActiveBSCells][l.BestRSRPNode]
		tmp = append(tmp, l)
		result[ActiveBSCells*2][l.BestRSRPNode] = tmp
	})

	fmt.Printf("\n \n Processing %#v \n %#v \n %#v", len(Cell0Sec0), len(Cell0Sec1), len(Cell0Sec2))
	return result
}

func CreateSectorInterfers(fname string) {

}
