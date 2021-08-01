package main

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/5gif/config"
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

func splitUELocations(fname string) {
	fname = basedir + fname
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

func splitLinkProfile(fname string) {
	fname = basedir + fname
	var fds [19]*os.File

	gcells := vlib.NewSegmentI(0, 19)
	for _, v := range gcells {

		fname := basedir + fmt.Sprintf("uelocations-cell%02d.csv", v)
		fds[v], _ = os.Create(fname)

		headers, _ := vlib.Struct2Header(LinkProfile{})
		fds[v].Write([]byte(strings.Join(headers, ",")))

		defer fds[v].Close()

	}

	// var cnt int = 0

	ForEachParse(fname, func(u LinkProfile) {

		// gcell := u.GCellID
		// fmt.Printf("\n %d [%d]| %v ", cnt, gcell, u)
		// str, err := vlib.Struct2String(u)
		// er(err)
		// fds[gcell].WriteString("\n" + str)

		// cnt++
	})

}

func splitSLSprofile(ues []UElocation) {
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

func splitLinkLevelInfo(fname string) {
	fname = basedir + fname
	var fds [19]*os.File

	gcells := vlib.NewSegmentI(0, 19)
	for _, v := range gcells {

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
			fmt.Println("Changin Cell ", gcellid, cnt)
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

var itucfg config.ITUconfig
var bslocs []BSlocation

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
func main() {
	itucfg, _ = config.ReadITUConfig(basedir + "itu.cfg")
	LoadCSV("bslocation.csv", &bslocs)
	// splitLinkLevelInfo("linkproperties.csv")

	fname := basedir + "linkproperties.csv"
	fd, _ := os.Create(basedir + "linkproperties-min.csv")
	defer fd.Close()

	nl := SubStruct(LinkProfile{}, "Rxid", "TxID", "CouplingLoss")
	str, err := vlib.Struct2HeaderLine(nl)
	er(err)
	fd.WriteString("\n" + str)

	ForEachParse(fname, func(l LinkProfile) {

		// fmt.Println("Read : %#v", l)
		// gcellid := ((l.Rxid - len(bslocs)) / itucfg.NumUEperCell)
		// fmt.Printf("\n%v | CNT = %d, ue=%d ", gcellid, cnt, l.Rxid-len(bslocs))
		// cnt++
		nl := SubStruct(l, "Rxid", "TxID", "CouplingLoss")
		str, err := vlib.Struct2String(nl)
		er(err)
		fd.WriteString("\n" + str)
		// fmt.Printf("\nWriting ...%v", str)

	})

	// data := LinkProfile{TxID: 100, Rxid: 430, CouplingLoss: 80, IsLOS: true}
	// result, err := SubStruct(data, "Rxid", "TxID", "CouplingLoss").(struct {
	// 	Rxid         int
	// 	TxID         int
	// 	CouplingLoss float64
	// })
	// fmt.Println("Error ", err)
	//"Pathloss", "O2I", "InCar", "ShadowLoss", "BSAasgainDB"
	// splitUELocations("uelocation.csv")
	// fmt.Printf("%#v | %#v", result, MiniLink{})
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
