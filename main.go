package main

import (
	"encoding/json"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

type OpenWeatherMapResponse struct {
	Main struct {
		Temp     float32 `json:"temp"`
		Pressure uint16  `json:"pressure"`
		Humidity uint8   `json:"humidity"`
	}
}

var q = url.Values{}
var qString string

func getMetrics(baseUrl string) *OpenWeatherMapResponse {
	log.Println("Getting weather using url " + baseUrl)

	base, err := url.Parse(baseUrl)
	base.RawQuery = qString

	res, err := http.Get(base.String())
	if err != nil {
		log.Println("Error getting weather")
		return nil
	}

	body, err := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode > 299 {
		log.Printf("Response failed with status code: %d and body: %s", res.StatusCode, body)
		return nil
	}
	var result OpenWeatherMapResponse
	if err := json.Unmarshal(body, &result); err != nil { // Parse []byte to go struct pointer
		log.Println("Can not unmarshal JSON")
		return nil
	}
	return &result
}

func recordMetrics(city string) {
	go func() {
		baseUrl := "https://api.openweathermap.org/data/2.5/weather"
		for {
			result := getMetrics(baseUrl)
			if result == nil {
				time.Sleep(10 * time.Minute)
				continue
			}
			temperatureOutside.WithLabelValues(city).Set(float64(result.Main.Temp))
			humidityOutside.WithLabelValues(city).Set(float64(result.Main.Humidity))
			pressureOutside.WithLabelValues(city).Set(float64(result.Main.Pressure))
			lastUpdatedOutside.WithLabelValues(city).Set(float64(time.Now().Unix()))
			time.Sleep(10 * time.Minute)
		}
	}()
}

var (
	temperatureOutside = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "weather_temperature",
		Help: "Outside temperature in °C",
	}, []string{"city"})

	humidityOutside = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "weather_humidity",
		Help: "Outside humidity in %",
	}, []string{"city"})

	pressureOutside = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "weather_pressure",
		Help: "Outside pressure in hPa",
	}, []string{"city"})

	lastUpdatedOutside = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "weather_last_updated",
		Help: "Last update of weather",
	}, []string{"city"})
)

func main() {
	log.Println("Starting server")

	// Get and handle errors for LAT
	lat := os.Getenv("LAT")
	if lat == "" {
		log.Fatal("LAT is empty")
	}
	q.Add("lat", lat)

	// Get and handle errors for LON
	lon := os.Getenv("LON")
	if lon == "" {
		log.Fatal("LON is empty")
	}
	q.Add("lon", lon)

	// Get and handle errors for UNITS
	units := os.Getenv("UNITS")
	if units == "" {
		log.Fatal("UNITS is empty")
	}
	q.Add("units", units)

	city := os.Getenv("CITY")
	if lon == "" {
        log.Fatal("CITY is empty")
    }

	// Get and handle errors for OPENWEWATHER_API_KEY
	apiKey := os.Getenv("OPENWEATHER_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENWEATHER_API_KEY is empty")
	}
	q.Add("appid", apiKey)
	qString = q.Encode()
	recordMetrics(city)

	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":"+os.Getenv("PORT"), nil))
}
