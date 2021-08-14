package main

import (
	"fmt"
	"math/rand"
	"os"

	"github.com/schollz/progressbar"
	"github.com/wiless/d3"
	"github.com/wiless/vlib"
)

func FindRelays(fname string, percentage float64, cell int) []RelayNode {

	fd, err := os.Create(fname)
	er(err)
	defer fd.Close()
	fmt.Printf("\nFindRelay : %s\n", fname)

	header, _ := vlib.Struct2HeaderLine(RelayNode{})
	fd.WriteString(header)

	NRelayPerCell := int(float64(itucfg.NumUEperCell) * percentage)
	// NCells := simcfg.ActiveUECells
	// TotalRelays := NRelayPerCell * NCells
	NRelayChannels := 4 /// = Total channels -1  (Channel 0 reserved for BS)

	baseuefile := basedir + "uelocation"
	baseslsfile := basedir + "newsls-mini"

	// cellbar := progressbar.Default(int64(NCells), "Finding Relay in ")

	// for cell := 0; cell < NCells; cell++ {

	ues := LoadUELocations(baseuefile + fmt.Sprintf("-cell%02d.csv", cell))

	outdoorues := d3.Filter(ues, func(ue UElocation) bool {
		return !ue.Indoor
	}).([]UElocation)
	if len(outdoorues) < NRelayPerCell {
		fmt.Println("Skipping Cell ", cell)
		return []RelayNode{}
	}
	outdooruesids := d3.FlatMap(outdoorues, "ID")

	sls := LoadSLSprofile(baseslsfile + fmt.Sprintf("-cell%02d.csv", cell))
	var potential vlib.VectorI

	d3.ForEach(sls, func(indx int, s SLSprofile) bool {
		found := false
		if s.BestSINR > 10 {
			found, _ = vlib.Contains(outdooruesids, s.RxNodeID)
			if found {
				potential.AppendAtEnd(s.RxNodeID)
			}
		}
		return found
	})

	_, indx := vlib.RandUFVec(len(outdoorues)).Sorted2()
	rand.Shuffle(len(outdoorues), func(i, j int) {
		outdoorues[i], outdoorues[j] = outdoorues[j], outdoorues[i]
	})

	relays := make([]RelayNode, NRelayPerCell)
	for i := 0; i < NRelayPerCell; i++ {
		relays[i].UElocation = outdoorues[indx[i]]
		// Unassigned Beacon Frequency =-1
		relays[i].FrequencyGHz = -1

		// random between 1 to 4, ZERO is for basestation
		relays[i].FrequencyGHz = float64(rand.Intn(NRelayChannels)) + 1
	}

	/// UNCOMMENT -- START to DISABLE OPTIMAL assignment
	// APPROACH FOR OPTIMIZED Frequency Assignment
	/// Assign Frequency | Optimize Frequency Assignment based on distance..
	// NclustersPerFrequency := math.Floor(float64(NRelayPerCell) / float64(NRelayChannels))
	// extraCluster := NRelayPerCell % NRelayChannels

	// for f := 1; f <= NRelayChannels; f++ {

	// 	var NumRelaysInF = int(NclustersPerFrequency) + extraCluster
	// 	srcindx := d3.FindFirstIndex(relays, func(r RelayNode) bool {
	// 		return r.FrequencyGHz == -1 // Find the first UNASSIGNED relay
	// 	})

	// 	relays[srcindx].FrequencyGHz = float64(f)
	// 	src := relays[srcindx]
	// 	distance := make([]float64, (NRelayPerCell))
	// 	srcloc := vlib.Location3D{X: src.X, Y: src.Y, Z: src.Z}
	// 	for iter, v := range relays {
	// 		var d float64
	// 		d = math.Inf(+1)
	// 		if v.ID != src.ID && v.FrequencyGHz == -1 { /// If not src!=dest and already no frequency assigned
	// 			destloc := vlib.Location3D{X: v.X, Y: v.Y, Z: v.Z}
	// 			d = srcloc.DistanceFrom(destloc)
	// 		}
	// 		distance[iter] = d
	// 	}
	// 	sdistance, sindex := vlib.Sorted(distance)

	// 	cnt := 0
	// 	for d := sdistance.Len() - 1; d >= 0; d-- {
	// 		if !math.IsInf(sdistance[d], +1) && cnt < NumRelaysInF-1 {
	// 			relays[sindex[d]].FrequencyGHz = float64(f)
	// 			cnt++
	// 		}
	// 	}

	// 	if extraCluster > 0 {
	// 		extraCluster--
	// 	}
	// }

	// sort.Slice(relays, func(i, j int) bool {
	// 	return relays[i].ID < relays[j].ID
	// })
	/// UNCOMMENT -- END to DISABLE OPTIMAL assignment
	// if GENERATE {
	for _, v := range relays {
		str, _ := vlib.Struct2String(v)
		fd.WriteString("\n" + str)
	}
	// }

	return relays

	// }

}
func GroupByFreq(vr []RelayNode) map[float64][]RelayNode {
	result := make(map[float64][]RelayNode)
	for _, v := range vr {
		// fmt.Printf("\n Grouping Relay | ", v.FrequencyGHz)
		fq := v.FrequencyGHz
		tmp := result[fq]
		result[fq] = append(tmp, v)
	}
	return result
}

// GenerateRelayLinkProps generates the link properties of all devices towards these relays
func GenerateRelayLinkProps(relays []RelayNode, ues []UElocation) map[int]SLSprofile {
	result := make(map[int]SLSprofile)
	fd, er := os.Create(basedir + "relayLinkProperties-mini.csv")
	Er(er)
	defer fd.Close()

	header, _ := vlib.Struct2HeaderLine(LinkFiltered{})
	fd.WriteString(header)

	rfd, er := os.Create(basedir + "relaySLSprofile-mini.csv")
	Er(er)
	defer rfd.Close()
	// slsmini := d3.SubStruct(SLSprofile{}, "RxNodeID", "BestRSRPNode", "BestSINR", "BestULsinr", "AssoTxAg", "AssoRxAg")
	header, _ = vlib.Struct2HeaderLine(SLSprofile{})
	rfd.WriteString(header)

	allrelayIDs := vlib.VectorI(d3.FlatMap(relays, "ID").([]int))

	gr := GroupByFreq(relays)
	pbar := progressbar.Default(int64(len(ues)), "Evaluating UE")
	for _, ue := range ues {

		if !allrelayIDs.Contains(ue.ID) {
			// Filter relay by frequency ...
			bestSLS := SLSprofile{BestSINR: -1000}
			rlinks := make([]LinkFiltered, 0)
			for k, v := range gr {
				tmplinks := make([]LinkFiltered, 0)
				// fmt.Printf("\n %d FrequencyGHz : %v | %d Relays are ", ue.ID, k, len(v))
				for _, r := range v {
					// Only for UEs which are not RELAYs
					lp := EvaluateMetricRelay(ue, r)
					tmplinks = append(tmplinks, lp)
					rlinks = append(rlinks, lp)

				}
				// Find BestRSRP Node..among the relays

				sls := BestSINR(tmplinks, itucfg.UETxDbm, N0)
				sls.FreqInGHz = k
				// fmt.Printf("\n%d : %#v ", ue.ID, sls)
				if sls.BestSINR > bestSLS.BestSINR {
					bestSLS = sls
				}
			}
			// fmt.Printf("\n%d : BEST %#v ", ue.ID, bestSLS)
			for k := range rlinks {
				if rlinks[k].TxID == bestSLS.BestRSRPNode {
					rlinks[k].BestRSRPNode = bestSLS.BestRSRPNode
					// if save..
					str, _ := vlib.Struct2String(rlinks[k])
					fd.WriteString("\n" + str)
				}

			}

			result[bestSLS.RxNodeID] = bestSLS
			// fmt.Printf("\n BEST SLS : %#v ", bestSLS)
			// slsmini := d3.SubStruct(bestSLS, "RxNodeID", "BestRSRPNode", "BestSINR", "BestULsinr", "AssoTxAg", "AssoRxAg")
			str, _ := vlib.Struct2String(bestSLS)
			rfd.WriteString("\n" + str)
		}

		pbar.Add(1)
	}

	return result
}

func EvaluateMetricRelay(rx UElocation, tx RelayNode) LinkFiltered {
	src := vlib.Location3D{tx.X, tx.Y, tx.Z}
	dest := vlib.Location3D{rx.X, rx.Y, rx.Z}
	newlink := LinkProfile{
		RxNodeID: rx.ID,
		TxID:     tx.ID,
		Distance: dest.DistanceFrom(src),
		UEHeight: rx.Z,
	}
	// IsLOS:
	// CouplingLoss, Pathloss, O2I, InCar, ShadowLoss, TxPower, BSAasgainDB, UEAasgainDB, TxGCSaz, TxGCSel, RxGCSaz, RxGCSel
	var indoordist = 0.0
	if rx.Indoor {
		indoordist = 25.0 * rand.Float64() // Assign random indoor distance  See Table 7.4.3-2
	}

	newlink.IndoorDistance = indoordist
	newlink.IsLOS = IsLOS(newlink.Distance) // @Todo

	newlink.InCar = 0 // NO CARS in mMTC
	if rx.InCar {
		newlink.InCar = O2ICarLossDb() // Calculate InCar Loss

	}

	if newlink.IsLOS {
		newlink.Pathloss = PL(newlink.Distance, itucfg.CarriersGHz, 1.5) // @Todo
	} else {
		newlink.Pathloss = PLNLOS(newlink.Distance, itucfg.CarriersGHz, 1.5) // @Todo
	}

	if rx.Indoor {
		newlink.O2I = O2ILossDb(itucfg.CarriersGHz, newlink.IndoorDistance)
	}
	newlink.CouplingLoss = newlink.BSAasgainDB - (newlink.Pathloss + newlink.O2I + newlink.InCar) // CouplingGain

	newlink.TxPower = 23.0
	newlink.BSAasgainDB = 0
	var lp LinkFiltered
	lp = LinkFiltered{RxNodeID: newlink.RxNodeID, TxID: newlink.TxID, CouplingLoss: newlink.CouplingLoss, BestRSRPNode: -1}

	return lp
}

func BestSINR(vlf vLinkFiltered, txpower float64, N0dB float64) SLSprofile {
	if len(vlf) == 0 {
		return SLSprofile{}
	}
	// Process and reset counter
	var totalrssi = 0.0
	var sinrvalues []float64
	var rssi []float64
	for _, v := range vlf {
		tmp := vlib.InvDb(v.CouplingLoss + txpower)
		totalrssi += tmp
		rssi = append(rssi, tmp)
		sinrvalues = append(sinrvalues, tmp)
	}

	for i, v := range sinrvalues {
		sinrvalues[i] = vlib.Db(v / (totalrssi - sinrvalues[i] + N0))
	}
	// fmt.Printf("\n Before sorting SINR %v ", sinrvalues)
	_, indx := vlib.Sorted(sinrvalues)
	// fmt.Printf("\n Size of SINR %d \n %v ", len(indx), sinrvalues)
	bestid := indx[len(indx)-1] //  indx[NBs-1]
	if bestid == -1 {
		fmt.Println("Index of SORT ", indx, len(indx)-1, sinrvalues, vlf)
	}

	bestlink := vlf[bestid]
	maxsinr := sinrvalues[bestid]

	/// full  details
	// if full {
	sls := SLSprofile{
		RxNodeID:  bestlink.RxNodeID,
		FreqInGHz: itucfg.CarriersGHz, BandwidthMHz: itucfg.BandwidthMHz, N0: N0dB, RSSI: totalrssi,
		BestRSRP:         vlib.Db(rssi[bestid]),
		BestRSRPNode:     bestlink.TxID,
		BestSINR:         maxsinr,
		RoIDbm:           vlib.Db(totalrssi - rssi[bestid]),
		BestCouplingLoss: bestlink.CouplingLoss,
		BestULsinr:       bestlink.CouplingLoss + ueTxPowerdBm - UL_N0dB,
	}
	// BestULsinr  assumes single transmission ideal UPlink
	// str, _ = vlib.Struct2String(sls)
	// }
	// else {
	// 	// small file size.. if less columns are added
	// 	sls = SLSprofile{
	// 		RxNodeID:     bestlink.RxNodeID,
	// 		BestRSRPNode: bestlink.TxID,
	// 		BestSINR:     maxsinr,
	// 		AssoTxAg:     bestlink.BSAasgainDB, AssoRxAg: bestlink.UEAasgainDB,
	// 		BestULsinr: bestlink.CouplingLoss + ueTxPowerdBm - UL_N0dB,
	// 	}
	// 	newsls := d3.SubStruct(sls, "RxNodeID", "BestRSRPNode", "BestSINR", "BestULsinr", "AssoTxAg", "AssoRxAg")
	// 	str, _ = vlib.Struct2String(newsls)
	// }

	return sls
}
