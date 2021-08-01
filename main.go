package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/jszwec/csvutil"
	"github.com/wiless/d3"
	"github.com/wiless/vlib"
)

// func main() {
// 	log.Println("Starting..")
// 	//	LoadUELocations("uelocation.csv")

// }
var v3 vlib.VectorIface

var myues []UElocation

func (u *UElocation) MatchFn() *func() {
	fn := func() {
		log.Printf("I am cool")
	}
	return &fn

}

func main() {
	rand.Seed(time.Now().Unix())
	fmt.Println("Starting..")
	// var ivec vlib.GIntVector
	// var ivec = []UElocation{{ID: 0, X: -100}, {ID: 1, X: 200}, {ID: 2, X: 200}, {ID: 3, X: 200}, {ID: 4, X: 200}}
	ues := LoadUELocations("uelocation.csv")
	slsprofile := LoadSLSprofile("slsprofile.csv")

	var filteredues []UElocation
	myfunc := func(ue UElocation) bool {
		return !ue.Indoor
	}
	filteredues = d3.Filter(ues, myfunc).([]UElocation)
	_ = filteredues
	// fmt.Println("Matching devices",
	// 	d3.FilterIndex(ues, myfunc))
	fmt.Printf("\n\n Outdoor Gcell %v", d3.FlatMap(filteredues, "ID"))

	fmt.Printf("\nSearch High SINR")

	type Relay struct {
		UElocation
		Relay bool
		SLSprofile
	}
	var relays []Relay

	newues := d3.Filter(filteredues, func(ue UElocation) bool {
		var findindex int
		findindex = d3.FindFirstIndex(slsprofile, func(indx int, sls SLSprofile) bool {
			valid := sls.BestSINR > 15 && sls.RxNodeID == ue.ID
			if valid {
				fmt.Printf("..")
				sls.FreqInGHz += float64(rand.Intn(4))
				relays = append(relays, Relay{ue, true, sls})
			}
			return valid
		})
		return (findindex >= 0)
	}).([]UElocation)

	// fmt.Printf("\n\n SINR %#v", relays)
	// str, _ := csvutil.Header(UElocation{}, "")
	fid, _ := os.Create("relays.csv")
	w := csv.NewWriter(fid)
	enc := csvutil.NewEncoder(w)
	enc.EncodeHeader(Relay{})
	for _, v := range relays {
		enc.Encode(v)
	}
	w.Flush()

	fmt.Printf("\n\n RelayID %v", d3.FlatMap(newues, "ID"))
	fmt.Println("\n\n done....")
}
