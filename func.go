package main

import (
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"

	"github.com/wiless/d3"
	"github.com/wiless/vlib"
)

// CreateSLS creates a new SLSprofile file (fname string, full bool)
// UEids are sorted, recalculates SINR based on linkproperties.csv
// full=false, can limit columns to just rxid, txid,sinr to generate a reduced filesize
// Suggestion use full=true, fname=slsprofile-new.csv
// Suggestion use full=false, fname=slsprofile-min.csv
func CreateSLS(slsfname, linkfname string, full bool) {
	var fullstr = "full"
	if !full {
		fullstr = "mini"
	}
	log.Printf("CreateSLS:(%s) : %s (Regenerate from : %s) ", fullstr, slsfname, linkfname)

	var data []LinkProfile

	fd, _ := os.Create(slsfname)

	defer fd.Close()
	cnt := 0
	var sls SLSprofile
	str, _ := vlib.Struct2HeaderLine(sls)
	if !full {
		newsls := SubStruct(sls, "RxNodeID", "BestRSRPNode", "BestSINR", "AssoTxAg", "AssoRxAg")
		str, _ = vlib.Struct2HeaderLine(newsls)
	}

	fmt.Fprintf(fd, "\n%s", str)

	TotalLines := itucfg.NumUEperCell * NBs * 19
	progress := 0
	step := int(float64(TotalLines*5) / 100.0) // 5%
	finfo, _ := os.Stat(linkfname)
	fmt.Printf("%d ROWS %vMB\n", TotalLines, finfo.Size()/(1024*1024))
	N0dB := vlib.Db(N0)
	ForEachParse(linkfname, func(l LinkProfile) {

		data = append(data, l)
		cnt++
		if cnt%NBs == 0 {
			// Process and reset counter
			var totalrssi = 0.0
			var sinrvalues []float64
			var rssi []float64
			for _, v := range data {
				tmp := vlib.InvDb(v.CouplingLoss + v.TxPower)
				totalrssi += tmp
				rssi = append(rssi, tmp)
				sinrvalues = append(sinrvalues, tmp)
			}

			for i, v := range sinrvalues {
				sinrvalues[i] = vlib.Db(v / (totalrssi - sinrvalues[i] + N0))
			}

			_, indx := vlib.Sorted(sinrvalues)
			bestid := indx[NBs-1]
			bestlink := data[bestid]
			maxsinr := sinrvalues[bestid]

			{

				var str string
				/// full  details
				if full {
					sls = SLSprofile{
						RxNodeID:  bestlink.Rxid,
						FreqInGHz: itucfg.CarriersGHz, BandwidthMHz: itucfg.BandwidthMHz, N0: N0dB, RSSI: totalrssi,
						BestRSRP:         vlib.Db(rssi[bestid]),
						BestRSRPNode:     bestlink.TxID,
						BestSINR:         maxsinr,
						RoIDbm:           vlib.Db(totalrssi - rssi[bestid]),
						BestCouplingLoss: bestlink.CouplingLoss,
						AssoTxAg:         bestlink.BSAasgainDB, AssoRxAg: bestlink.UEAasgainDB,
					}
					str, _ = vlib.Struct2String(sls)
				} else {
					// small file size.. if less columns are added
					sls = SLSprofile{
						RxNodeID:     bestlink.Rxid,
						BestRSRPNode: bestlink.TxID,
						BestSINR:     maxsinr,
						AssoTxAg:     bestlink.BSAasgainDB, AssoRxAg: bestlink.UEAasgainDB,
					}
					newsls := SubStruct(sls, "RxNodeID", "BestRSRPNode", "BestSINR", "AssoTxAg", "AssoRxAg")
					str, _ = vlib.Struct2String(newsls)
				}
				fmt.Fprintf(fd, "\n%s", str)
			}

			data = []LinkProfile{}

			cnt = 0
		}
		progress++
		if progress%step == 0 {
			fmt.Printf("==")

		}
	})
	fmt.Printf("\n")

}

// CreateLinkProfiles creates a minimal version of linkproperties.csv by trucating fields to
// "Rxid", "TxID", "CouplingLoss"
func CreateMiniLinkProfiles(newfname string, linkfname string) {
	log.Printf("CreateMiniLinkProfiles: %s => %s", linkfname, newfname)

	fd, _ := os.Create(newfname)
	defer fd.Close()

	nl := SubStruct(LinkProfile{}, "Rxid", "TxID", "CouplingLoss")
	str, err := vlib.Struct2HeaderLine(nl)
	er(err)
	fd.WriteString("\n" + str)
	// progress bar
	TotalLines := itucfg.NumUEperCell * NBs * 19
	step := int(float64(TotalLines*5) / 100.0) // 5%
	cnt := 0

	finfo, _ := os.Stat(linkfname)
	fmt.Printf("%d ROWS %vMB\n", TotalLines, finfo.Size()/(1024*1024))
	ForEachParse(linkfname, func(l LinkProfile) {

		nl := SubStruct(l, "Rxid", "TxID", "CouplingLoss")
		str, err := vlib.Struct2String(nl)
		er(err)
		fd.WriteString("\n" + str)
		cnt++
		if cnt%step == 0 {
			fmt.Printf("==")

		}
	})
	fmt.Printf("\n")
}

// SplitSLSprofileByUEs
func SplitSLSprofileByUEs(ues []UElocation) {
	var fds [19]*os.File

	gcells := vlib.NewSegmentI(0, 19)
	for _, v := range gcells {

		fname := basedir + fmt.Sprintf("slsprofile-cell%02d.csv", v)
		fds[v], _ = os.Create(fname)

		headers, _ := vlib.Struct2HeaderLine(SLSprofile{})
		fds[v].WriteString(headers)

		defer fds[v].Close()

	}
	slsprofile := LoadSLSprofile("slsprofile.csv")
	// split SLSprofile entries into multiple GCellID
	d3.ForEach(slsprofile, func(sls SLSprofile) bool {
		indx := d3.FindFirstIndex(ues, func(ue UElocation) bool {
			return ue.ID == sls.RxNodeID
		})
		if indx != -1 {
			gcell := ues[indx].GCellID
			str, err := vlib.Struct2String(sls)
			er(err)
			fds[gcell].WriteString("\n" + str)

		}
		return indx != -1

	})

	// fmt.Fprintf(fds[0], "%v", selected0)
	for _, v := range gcells {
		// wrs[v].Flush()
		// fds[v].Sync()
		// fmt.Printf("Error ?? %v", wrs[v].Error())
		fds[v].Close()
	}

}

// SplitSLSprofileByCell assumes its ordered file with UE in order
// Use the file created by CreateSLS()
func SplitSLSprofileByCell(newfnamebase, slsfname string, full bool) {
	var fullstr = "full"
	if !full {
		fullstr = "mini"
	}
	log.Printf("SplitSLSprofileByCell(%s):: %s-cell[0-19]csv (Regenerate from : %s) ", fullstr, newfnamebase, slsfname)
	var err error
	var fds [19]*os.File
	for i := 0; i < 19; i++ {

		fname := fmt.Sprintf(newfnamebase+"-cell%02d.csv", i)
		fds[i], err = os.Create(fname)
		er(err)

		if !full {
			newsls := SubStruct(SLSprofile{}, "RxNodeID", "BestRSRPNode", "BestSINR", "AssoTxAg", "AssoRxAg")
			headers, _ := vlib.Struct2HeaderLine(newsls)
			fds[i].WriteString(headers)
		} else {
			headers, _ := vlib.Struct2HeaderLine(SLSprofile{})
			fds[i].WriteString(headers)
		}

		defer fds[i].Close()

	}

	TotalLines := itucfg.NumUEperCell
	progress := 0
	step := int(float64(TotalLines*5) / 100.0) // 5%
	finfo, _ := os.Stat(slsfname)
	fmt.Printf("%d ROWS %vMB\n", TotalLines, finfo.Size()/(1024*1024))

	// split SLSprofile entries into multiple GCellID
	var gcellid int = 0
	var cnt = 0
	NentriesPerCell := itucfg.NumUEperCell
	ForEachParse(slsfname, func(l SLSprofile) {
		var str string
		if !full {
			newsls := SubStruct(l, "RxNodeID", "BestRSRPNode", "BestSINR", "AssoTxAg", "AssoRxAg")
			str, _ = vlib.Struct2String(newsls)

		} else {
			str, _ = vlib.Struct2String(l)
		}

		fds[gcellid].WriteString("\n" + str)
		cnt++

		if cnt%(NentriesPerCell) == 0 {
			gcellid++
			cnt = 0
		}
		progress++
		if progress%step == 0 {
			fmt.Printf("==")
		}

	})
	fmt.Printf("\n")
}

// SplitLinkProfilesByCells linkproperties.csv into linkproperties-cellXX.csv XX=00,01,02,...18
// with minimal fields "Rxid", "TxID", "CouplingLoss"
func SplitLinkProfilesByCell(newfnamebase, linkfname string, full bool) {

	var fullstr = "full"
	if !full {
		fullstr = "mini"
	}
	log.Printf("SplitLinkProfilesByCell:(%s) : %s-[0-19]csv (Regenerate from : %s) ", fullstr, newfnamebase, linkfname)

	var fds [19]*os.File
	gcells := vlib.NewSegmentI(0, 19)
	for _, v := range gcells {
		// Rxid,txID,distance,IndoorDistance,UEHeight,IsLOS,CouplingLoss,Pathloss,O2I,InCar,ShadowLoss,TxPower,BSAasgainDB,UEAasgainDB,TxGCSaz,TxGCSel,RxGCSaz,RxGCSel
		fname := fmt.Sprintf(newfnamebase+"-cell%02d.csv", v)
		fds[v], _ = os.Create(fname)
		var headers string
		if !full {
			nlobj := SubStruct(LinkProfile{}, "Rxid", "TxID", "CouplingLoss")
			headers, _ = vlib.Struct2HeaderLine(nlobj)
		} else {

			headers, _ = vlib.Struct2HeaderLine(LinkProfile{})
		}
		fds[v].WriteString(headers)

		defer fds[v].Close()

	}

	// split LinkProperties entries into multiple GCellID
	TotalLines := itucfg.NumUEperCell * NBs * 19
	progress := 0
	step := int(float64(TotalLines*5) / 100.0) // 5%
	finfo, _ := os.Stat(linkfname)
	fmt.Printf("%d ROWS %vMB\n", TotalLines, finfo.Size()/(1024*1024))

	var cnt int = 0
	NLinksPerCell := itucfg.NumUEperCell * NBs
	var gcellid int = 0
	ForEachParse(linkfname, func(l LinkProfile) {

		if !full {
			nl := SubStruct(l, "Rxid", "TxID", "CouplingLoss")
			str, err := vlib.Struct2String(nl)
			er(err)
			fds[gcellid].WriteString("\n" + str)

		} else {
			str, err := vlib.Struct2String(l)
			er(err)
			fds[gcellid].WriteString("\n" + str)
		}

		cnt++
		if cnt%(NLinksPerCell) == 0 {
			cnt = 0
			gcellid++
		}
		progress++
		if progress%step == 0 {
			fmt.Printf("==")
		}

	})
	fmt.Printf("\n")
	// fmt.Fprintf(fds[0], "%v", selected0)
	for _, v := range gcells {
		// wrs[v].Flush()
		// fds[v].Sync()
		// fmt.Printf("Error ?? %v", wrs[v].Error())
		fds[v].Close()
	}

}

// SubStruct creates array of objs with selected properties "fields" from the input array of objects
func SubStruct(v interface{}, fields ...string) interface{} {
	// fmt.Printf("\n Input : %#v", v)
	tOfv := reflect.TypeOf(v)
	var subfields []reflect.StructField
	var fnames []string
	for _, f := range fields {
		ftype, ok := tOfv.FieldByName(f)
		if ok {
			subfields = append(subfields, ftype)
			fnames = append(fnames, f)
		}
	}
	resultType := reflect.StructOf(subfields)
	elemVal := reflect.ValueOf(v)
	result := reflect.New(resultType)

	for _, f := range fnames {
		inpval := elemVal.FieldByName(f)
		// fmt.Printf("\n\nField  %v is %v ", f, inpval)
		newfield := result.Elem().FieldByName(f)
		// fmt.Printf("\nBefore Setting  %v is %#v ", f, newfield)
		if newfield.CanSet() {
			newfield.Set(inpval)
			// fmt.Printf("\nSetting  %v is %#v ", f, newfield)
		}

	}

	retobj := result.Elem()

	// fmt.Printf("\n Created : %#v", retobj)

	return retobj.Interface()
	// for i := 0; i < N; i++ {
	// 	tOfv.FieldByName()
	// }
}

// SplitUELocations split uelocations.csv based on GCell
func SplitUELocationsByCell(fname string) {
	log.Println("SplitUELocationsByCell:", fname)
	var fds [19]*os.File

	gcells := vlib.NewSegmentI(0, 19)
	for _, v := range gcells {

		fname := basedir + fmt.Sprintf("uelocation-cell%02d.csv", v)
		fds[v], _ = os.Create(fname)

		headers, _ := vlib.Struct2Header(UElocation{})
		fds[v].Write([]byte(strings.Join(headers, ",")))

		defer fds[v].Close()

	}

	var cnt int = 0

	ForEachParse(fname, func(u UElocation) {

		gcell := u.GCellID
		// fmt.Printf("\n %d [%d]| %v ", cnt, gcell, u)
		str, err := vlib.Struct2String(u)
		er(err)
		fds[gcell].WriteString("\n" + str)

		cnt++
	})

}
