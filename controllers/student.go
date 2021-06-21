package controllers

import (
	"encoding/csv"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"time"
	"github.com/paulmach/go.geojson"
	"github.com/pkg/errors"
)

func StudentAqisHandler(w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()
	to, from, err := parseTimeInput(values)
	if err != nil {
		http.Error(w, "Could not parse time: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var within string
	var area string
	var plotMap string
	var plotChart string

	if len(values["within"]) > 0 {
		within = values["within"][0]
	}
	if len(values["area"]) > 0 {
		area = values["area"][0]
	}
	if len(values["plotmap"]) > 0 {
		plotMap = values["plotmap"][0]
	}
	if len(values["plotchart"]) > 0 {
		plotChart = values["plotchart"][0]
	}

	filter := StudentFilter{
		ToTime:    to,
		FromTime:  from,
		Within:    within,
		Area:      area,
		PlotMap:   plotMap,
		PlotChart: plotChart,
	}

	data, err := getStudentData(filter)
	if err != nil {
		http.Error(w, "Could not parse student data: "+err.Error(), http.StatusInternalServerError)
		return
	}

	fc := geojson.NewFeatureCollection()

	for _, measurement := range data.Data {
		geom := geojson.NewPointGeometry([]float64{measurement.Attributes.Longitude, measurement.Attributes.Latitude})
		f := geojson.NewFeature(geom)
		f.SetProperty("date", measurement.Attributes.Date)
		f.SetProperty("pmTen", measurement.Attributes.PmTen)
		f.SetProperty("pmTwoFive", measurement.Attributes.PmTwoFive)
		f.SetProperty("humidity", measurement.Attributes.Humidity)
		f.SetProperty("temperature", measurement.Attributes.Temperature)
		f.SetProperty("color", "6ee86e")
		f.SetProperty("weight", 10)
		fc = fc.AddFeature(f)
	}

	b, err := fc.MarshalJSON()
	if err != nil {
		http.Error(w, "Could not marshal geojson"+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(b)
	return
}

type StudentFilter struct {
	Area      string
	Within    string
	FromTime  time.Time
	ToTime    time.Time
	PlotMap   string
	PlotChart string
}

func StudentHandler(w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()
	to, from, err := parseTimeInput(values)

	if err != nil {
		http.Error(w, "Could not parse time: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var within string
	var area string
	var plotMap string
	var plotChart string

	if len(values["within"]) > 0 {
		within = values["within"][0]
	}
	if len(values["area"]) > 0 {
		area = values["area"][0]
	}
	if len(values["plotmap"]) > 0 {
		plotMap = values["plotmap"][0]
	}
	if len(values["plotchart"]) > 0 {
		plotChart = values["plotchart"][0]
	}

	filter := StudentFilter{
		ToTime:    to,
		FromTime:  from,
		Within:    within,
		Area:      area,
		PlotMap:   plotMap,
		PlotChart: plotChart,
	}

	data, err := getStudentData(filter)
	if err != nil {
		http.Error(w, "Could not parse student data: "+err.Error(), http.StatusInternalServerError)
		return
	}

	records := [][]string{}
	header := []string{"timestamp", "latitude", "longitude", "pmTen", "pmTwoFive", "humidity", "temperature"}
	records = append(records, header)

	for _, measurement := range data.Data {
		var latitude float64
		var longitude float64
		var valuePmTen float64
		var valuePmTwoFive float64
		var valueHumidity float64
		var valueTemperature float64

		latitude = measurement.Attributes.Latitude
		longitude = measurement.Attributes.Longitude
		valuePmTen = measurement.Attributes.PmTen
		valuePmTwoFive = measurement.Attributes.PmTwoFive
		valueHumidity = measurement.Attributes.Humidity
		valueTemperature = measurement.Attributes.Temperature

		formattedLatitude := strconv.FormatFloat(latitude, 'f', -1, 64)
		formattedLongitude := strconv.FormatFloat(longitude, 'f', -1, 64)
		formattedPmTenValue := strconv.FormatFloat(valuePmTen, 'f', -1, 64)
		formattedPmTwoFiveValue := strconv.FormatFloat(valuePmTwoFive, 'f', -1, 64)
		formattedHumidityValue := strconv.FormatFloat(valueHumidity, 'f', -1, 64)
		formattedTemperatureValue := strconv.FormatFloat(valueTemperature, 'f', -1, 64)

		timestamp := measurement.Attributes.Date

		// fmt.Println(time.Parse(studentResponseTimeLayouttimestamp.Format("2006-01-02 15:04:05"))

		record := []string{
			timestamp,
			formattedLatitude,
			formattedLongitude,
			formattedPmTenValue,
			formattedPmTwoFiveValue,
			formattedHumidityValue,
			formattedTemperatureValue,
		}
		records = append(records, record)
	}

	writer := csv.NewWriter(w)

	filename := "studentdata.csv"
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)

	err = writer.WriteAll(records)
	if err != nil {
		http.Error(w, "Could not write csv", http.StatusInternalServerError)
		return
	}
}

var studentTimeLayout = "2006-01-02T15:04:05"
var studentResponseTimeLayout = "2006-01-02 15:04:05 -0700"

type Result struct {
	Data []Record `json:"data"`
}

type Record struct {
	Id string `json:"id"`
	Type string `json:"type"`
	Attributes Measurement `json:"attributes"`
}

type Measurement struct {
	Latitude    float64 
	Longitude   float64
	PmTen       float64
	PmTwoFive   float64
	Humidity    float64
	Temperature float64
	Date        string `json:"Timestamp"`
}

// Fetches and parses the student collected data
func getStudentData(filter StudentFilter) (Result, error) {

	fromDate := filter.FromTime.Format(studentTimeLayout)
	toDate := filter.ToTime.Format(studentTimeLayout)
	within := filter.Within
	area := filter.Area
	plotMap := filter.PlotMap
	plotChart := filter.PlotChart

	var u string
	if len(within) > 0 {
		u = "http://localhost:8000/api/data?totime=" + toDate + "&fromtime=" + fromDate + "&within=" + within
	}	else if len(area) > 0 {
		u = "http://localhost:8000/api/data?totime=" + toDate + "&fromtime=" + fromDate + "&area=" + url.QueryEscape(area)
	}	else {
		u = "http://localhost:8000/api/data?totime=" + toDate + "&fromtime=" + fromDate
	}
	// if len(within) > 0 {
	// 	u = "https://luft-184208.appspot.com/api/data?totime=" + toDate + "&fromtime=" + fromDate + "&within=" + within
	// } else if len(area) > 0 {
	// 	u = "https://luft-184208.appspot.com/api/data?totime=" + toDate + "&fromtime=" + fromDate + "&area=" + url.QueryEscape(area)
	// } else {
	// 	u = "https://luft-184208.appspot.com/api/data?totime=" + toDate + "&fromtime=" + fromDate
	// }

	if len(plotMap) > 0 {
		u += "&plotmap=" + plotMap
	} else if len(plotChart) > 0 {
		u += "&plotchart=" + plotChart
	}

	resp, err := http.Get(u)
	if err != nil {
		return Result{}, errors.Wrap(err, "Could not download data from luftprosjekttromso")
	}


	var data Result
	err = json.NewDecoder(resp.Body).Decode(&data)
	return data, nil
}
