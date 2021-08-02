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
	fmt.Fprintf(fd, "\n%s", str)
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

			sortedsinr, indx := vlib.Sorted(sinrvalues)
			bestid := indx[NBs-1]
			bestlink := data[bestid]
			maxsinr := sortedsinr[bestid]

			{

				/// full  details
				if full {
					sls = SLSprofile{
						RxNodeID:  bestlink.Rxid,
						FreqInGHz: itucfg.CarriersGHz, BandwidthMHz: itucfg.BandwidthMHz, N0: N0, RSSI: totalrssi, BestRSRP: rssi[bestid],
						BestRSRPNode: bestlink.TxID,
						BestSINR:     maxsinr, RoIDbm: vlib.Db(totalrssi - rssi[bestid]), BestCouplingLoss: bestlink.CouplingLoss,
						AssoTxAg: bestlink.BSAasgainDB, AssoRxAg: bestlink.UEAasgainDB,
					}
				} else {
					// small file size.. if less columns are added
					sls = SLSprofile{
						RxNodeID: bestlink.Rxid,

						BestRSRPNode: bestlink.TxID,
						BestSINR:     maxsinr,
						AssoTxAg:     bestlink.BSAasgainDB, AssoRxAg: bestlink.UEAasgainDB,
					}
				}

				str, _ := vlib.Struct2String(sls)
				fmt.Fprintf(fd, "\n%s", str)
			}

			data = []LinkProfile{}

			cnt = 0
		}

	})
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
	ForEachParse(linkfname, func(l LinkProfile) {

		nl := SubStruct(l, "Rxid", "TxID", "CouplingLoss")
		str, err := vlib.Struct2String(nl)
		er(err)
		fd.WriteString("\n" + str)

	})
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
func SplitSLSprofileByCell(fname string) {
	log.Println("SplitSLSprofileByCell: ", fname)

	var fds [19]*os.File
	for i := 0; i < 19; i++ {

		fname := basedir + fmt.Sprintf("slsprofile-cell%02d.csv", i)
		fds[i], _ = os.Create(fname)

		headers, _ := vlib.Struct2HeaderLine(SLSprofile{})
		fds[i].WriteString(headers)

		defer fds[i].Close()

	}

	// split SLSprofile entries into multiple GCellID
	var gcellid int = 0
	var cnt = 0
	NentriesPerCell := itucfg.NumUEperCell
	ForEachParse(fname, func(l SLSprofile) {

		str, err := vlib.Struct2String(l)
		er(err)
		fds[gcellid].WriteString("\n" + str)
		cnt++

		if cnt%(NentriesPerCell) == 0 {
			gcellid++
			cnt = 0
		}

	})

}

// SplitLinkProfilesByCells linkproperties.csv into linkproperties-cellXX.csv XX=00,01,02,...18
// with minimal fields "Rxid", "TxID", "CouplingLoss"
func SplitLinkProfilesByCell(fname string) {
	log.Println("SplitLinkProfilesByCell: ", fname)

	var fds [19]*os.File
	gcells := vlib.NewSegmentI(0, 19)
	for _, v := range gcells {
		// Rxid,txID,distance,IndoorDistance,UEHeight,IsLOS,CouplingLoss,Pathloss,O2I,InCar,ShadowLoss,TxPower,BSAasgainDB,UEAasgainDB,TxGCSaz,TxGCSel,RxGCSaz,RxGCSel
		fname := basedir + fmt.Sprintf("linkproperties-cell%02d.csv", v)
		fds[v], _ = os.Create(fname)

		nlobj := SubStruct(LinkProfile{}, "Rxid", "TxID", "CouplingLoss")
		headers, _ := vlib.Struct2HeaderLine(nlobj)
		fds[v].Write([]byte(headers))

		defer fds[v].Close()

	}

	// split LinkProperties entries into multiple GCellID
	var cnt int = 0
	NentriesPerCell := itucfg.NumUEperCell * len(bslocs)
	ForEachParse(fname, func(l LinkProfile) {

		// fmt.Println("Read : %#v", l)
		gcellid := ((l.Rxid - len(bslocs)) / itucfg.NumUEperCell)
		// fmt.Printf("\n%v | CNT = %d, ue=%d ", gcellid, cnt, l.Rxid-len(bslocs))
		// cnt++
		nl := SubStruct(l, "Rxid", "TxID", "CouplingLoss")
		str, err := vlib.Struct2String(nl)
		er(err)
		fds[gcellid].WriteString("\n" + str)
		// fmt.Printf("\nWriting ...%v", str)
		cnt++
		if cnt%(NentriesPerCell) == 0 {
			// fmt.Println("Changin Cell ", gcellid, cnt)
			cnt = 0
		}

	})

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
		fmt.Printf("\n %d [%d]| %v ", cnt, gcell, u)
		str, err := vlib.Struct2String(u)
		er(err)
		fds[gcell].WriteString("\n" + str)

		cnt++
	})

}
