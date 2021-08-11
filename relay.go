package main

import (
	"fmt"
	"math"
	"math/rand"
	"os"

	"github.com/schollz/progressbar"
	"github.com/wiless/d3"
	"github.com/wiless/vlib"
)

func FindRelays(fname string, percentage float64) {

	fd, err := os.Create(fname)
	er(err)
	defer fd.Close()
	fmt.Printf("\nFindRelay : %s\n", fname)

	header, _ := vlib.Struct2HeaderLine(RelayNode{})
	fd.WriteString(header)

	NRelayPerCell := int(float64(itucfg.NumUEperCell) * percentage)
	NCells := simcfg.ActiveUECells
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
		if len(outdoorues) < NRelayPerCell {
			fmt.Println("Skipping Cell ", cell)
			break
		}
		outdooruesids := d3.FlatMap(outdoorues, "ID")

		sls := LoadSLSprofile(baseslsfile + fmt.Sprintf("-cell%02d.csv", cell))
		var potential vlib.VectorI

		d3.ForEach(sls, func(indx int, s SLSprofile) bool {
			found := false
			if s.BestSINR > 10 {
				found, _ = vlib.Contains(outdooruesids, s.RxNodeID)
				if found {
					// sls[indx].FreqInGHz = -1
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
			srcloc := vlib.Location3D{X: src.X, Y: src.Y, Z: src.Z}
			for iter, v := range relays {
				var d float64
				d = math.Inf(+1)
				if v.ID != src.ID && v.FrequencyGHz == -1 { /// If not src!=dest and already no frequency assigned
					destloc := vlib.Location3D{X: v.X, Y: v.Y, Z: v.Z}
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
func GenerateRelayLinkProps() {

}
func EvalauteMetricRelay(rx UElocation, tx RelayNode) LinkProfile {

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
	// newlink.Pathloss = // @Todo
	// newlink.CouplingLoss = // @Todo

	newlink.TxPower = 23.0
	newlink.BSAasgainDB = 0
	return newlink
}
