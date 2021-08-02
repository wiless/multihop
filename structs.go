package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"reflect"

	"github.com/jszwec/csvutil"
)

// ID  ,X            ,Y            ,Z         ,TxPowerdBm ,Hdirection ,VTilt ,Active ,Alias
type BSlocation struct {
	ID                                     int
	X, Y, Z, TxPowerdBm, Hdirection, VTilt float64
	Active, Alias                          int
}

type UElocation struct {
	ID            int
	X, Y, Z       float64
	Indoor, InCar bool
	BSdist        float64
	GCellID       int

	// Name string `csv:"name"`
	// Address
}

// RxNodeID,FreqInGHz,BandwidthMHz,N0,RSSI,BestRSRP,BestRSRPNode,BestSINR,RoIDbm,BestCouplingLoss,MaxTxAg,MaxRxAg,AssoTxAg,AssoRxAg,MaxTransmitBeamID
type SLSprofile struct {
	RxNodeID                                                                 int
	FreqInGHz, BandwidthMHz, N0, RSSI, BestRSRP                              float64
	BestRSRPNode                                                             int
	BestSINR, RoIDbm, BestCouplingLoss, MaxTxAg, MaxRxAg, AssoTxAg, AssoRxAg float64
	MaxTransmitBeamID                                                        int
}

// Rxid,txID,distance,IndoorDistance,UEHeight,IsLOS,CouplingLoss,Pathloss,O2I,InCar,ShadowLoss,TxPower,BSAasgainDB,UEAasgainDB,TxGCSaz,TxGCSel,RxGCSaz,RxGCSel

type LinkProfile struct {
	Rxid                                                                                                                  int
	TxID                                                                                                                  int     `csv:"txID"`
	Distance                                                                                                              float64 `csv:"distance"`
	IndoorDistance, UEHeight                                                                                              float64
	IsLOS                                                                                                                 bool
	CouplingLoss, Pathloss, O2I, InCar, ShadowLoss, TxPower, BSAasgainDB, UEAasgainDB, TxGCSaz, TxGCSel, RxGCSaz, RxGCSel float64
}

func LoadCSV(fname string, v interface{}) interface{} {
	fname = basedir + fname

	fid, err := os.Open(fname)
	er(err)

	data, err := os.ReadFile(fname)
	er(err)

	// var sls []SLSprofile
	err = csvutil.Unmarshal(data, v)
	er(err)

	// fmt.Printf("LoadCSV %#v", v)
	defer fid.Close()

	fid.Close()
	return v
}

func LoadSLSprofile(fname string) []SLSprofile {
	// fname += ".csv"
	fname = basedir + fname

	fid, err := os.Open(fname)
	er(err)

	data, err := os.ReadFile(fname)
	er(err)

	var sls []SLSprofile
	err = csvutil.Unmarshal(data, &sls)
	er(err)

	defer fid.Close()

	fid.Close()
	return sls
}

var er = func(err error) {
	if err != nil {
		log.Println("Error ", err)
	}
}

//LoadUELocations saves the UE locations to fname in csv format
// uid,x,y,z,cellid,gdistance,in/out, car
func LoadUELocations(fname string) []UElocation {
	// fname += ".csv"
	fname = basedir + fname
	fid, err := os.Open(fname)
	er(err)

	data, err := os.ReadFile(fname)
	er(err)

	var ues []UElocation
	err = csvutil.Unmarshal(data, &ues)
	er(err)

	defer fid.Close()

	fid.Close()
	return ues
}

func ForEachParse(fname string, fn interface{}) {

	log.Printf("\n ForEach()..")
	tOffn := reflect.TypeOf(fn)

	fnVal := reflect.ValueOf(fn)

	// resultValue := reflect.MakeSlice(tOfArray, 0, avalue.Cap())

	// fmt.Printf("\n INPUT = arg1 = %v and arg2 =  %v  ", tOfArray, tOffn)
	// fmt.Printf("\n Kind arg1 %s ", tOfArray.Kind())
	// fmt.Printf("\n Kind arg2  %s ", tOffn.Kind())

	// fmt.Printf("\n ARRAY of type %s ", elemType)
	// fmt.Printf("\n Value / Handle of Function   %v ", fnVal)
	// fmt.Printf("\n Fn : Input Args:%d =", tOffn.NumIn())
	// for i := 0; i < tOffn.NumIn(); i++ {
	// 	fmt.Printf("\t %d=>%v,", i, tOffn.In(i))
	// }
	// fmt.Printf("\n Fn : Output Args:%d =", tOffn.NumOut())
	// for i := 0; i < tOffn.NumOut(); i++ {
	// 	fmt.Printf("\t %d=>%v,", i, tOffn.Out(i))
	// }

	if tOffn.NumIn() == 0 {
		fmt.Println("ForEach needs Fn with 1 or 2 input args")
		return
	}

	var fnType int = 1
	/// Function Argument must match Element type of the Array {
	if tOffn.NumIn() == 1 {
		fnType = 1
		// fmt.Printf("\nFunction TYPE 1 : (elem) : Element Type %v matches with Fn arg1 %v", elemType, tOffn.In(0), tOffn.In(1))
	} else if tOffn.NumIn() == 2 && tOffn.In(0).Kind() == reflect.Int {
		// Expect second argument of type "struct"
		fnType = 2
		// fmt.Printf("\nFunction TYPE 2 : (ind,elem)  : Element Type %v matches with Fn arg2 %v, arg 1=%v", elemType, tOffn.In(1), tOffn.In(0))
	} else {
		// fmt.Printf("\nError : Array Element %v DOES NOT MATCH with Fn arg %v", elemType, tOffn.In(0))
		return
	}

	var elemType reflect.Type
	elemType = tOffn.In(0)

	fd, err := os.Open(fname)
	er(err)
	csvReader := csv.NewReader(fd)

	dec, err := csvutil.NewDecoder(csvReader)
	if err != nil {
		log.Fatal(err)
	}

	header := dec.Header()
	_ = header
	var i int = 0
	// _ = fnVal
	elemValue := reflect.New(elemType)
	// fmt.Printf("\nType of Fn Arg is %v", elemType)
	// fmt.Printf("\nType of New Variable is %v | kind =%v", elemValue.Type(), elemType.Kind())

	for {

		// u := User{OtherData: make(map[string]string)}
		// element := reflect.New()

		u := elemValue.Interface()

		if err := dec.Decode(u); err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}
		// fmt.Printf("\nRead File : %#v", u)
		obj := elemValue.Elem()
		// fmt.Printf("\n OBJ %#v", obj)
		if fnType == 1 {
			fnVal.Call([]reflect.Value{obj})
		} else {
			var indx = reflect.ValueOf(i)
			fnVal.Call([]reflect.Value{indx, obj})
			i++
		}

	}

}
