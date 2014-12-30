package scraper

import (
	"errors"
	"time"
	"encoding/json"
	"github.com/garfunkel/dng/settings"
	"github.com/garfunkel/go-nbn"
	"github.com/garfunkel/go-realestatecomau"
	"github.com/garfunkel/go-adsl"
	"github.com/garfunkel/go-google/maps"
	"github.com/garfunkel/go-google/maps/geocoding"
	"github.com/garfunkel/go-google/maps/places/nearbysearch"
	"github.com/garfunkel/go-google/maps/distancematrix"
	"github.com/boltdb/bolt"
	"fmt"
	"net/url"
	"log"
)

const (
	DBPath = "dng.db"
	MapsEmbedURL = "https://www.google.com/maps/embed/v1/place?key=%v&q=%v&zoom=13"
	GoogleAPIKey = "AIzaSyC50lfM-BNpgJMXesZ9qV4Jx6ubTMmwwxA"
)

type NearbyAmenitiesInfo struct {
	Landmarks distancematrix.Response
	BusStops distancematrix.Response
	TrainStations distancematrix.Response
	Grocers distancematrix.Response
	Cafes distancematrix.Response
	Gyms distancematrix.Response
	Schools distancematrix.Response
	DepartmentStores distancematrix.Response
	Malls distancematrix.Response
	Bars distancematrix.Response
}

func (nearbyInfo *NearbyAmenitiesInfo) getLandmarks(latitude, longitude float64) (err error) {
	var landmarks maps.Locations

	for _, address := range settings.Settings.Landmarks {
		landmarks = append(landmarks, maps.AddressLocation{Address: address})
	}

	matrixRequiredParams := distancematrix.RequiredParams{
		Origins: maps.Locations{maps.LatLngLocation{
			Latitude: latitude,
			Longitude: longitude,
		}},
		Destinations: landmarks,
	}

	matrixModeParam := distancematrix.OptionalModeParam{
		Mode: "walking",
	}

	var matrix *distancematrix.Response

	for {
		matrix, err = distancematrix.DistanceMatrix(&matrixRequiredParams, &matrixModeParam)

		if err != nil {
			return
		}

		if matrix.Status != "OVER_QUERY_LIMIT" {
			break
		}

		time.Sleep(1 * time.Second)
	}

	nearbyInfo.Landmarks = *matrix

	return
} 

func (nearbyInfo *NearbyAmenitiesInfo) getBusStops(latitude, longitude float64) (err error) {
	matrix, err := getDistanceMatrix(latitude, longitude, "bus_station")

	if err != nil {
		return
	}

	nearbyInfo.BusStops = *matrix

	return
} 

func (nearbyInfo *NearbyAmenitiesInfo) getTrainStations(latitude, longitude float64) (err error) {
	matrix, err := getDistanceMatrix(latitude, longitude, "train_station")

	if err != nil {
		return
	}

	nearbyInfo.TrainStations = *matrix

	return
}

func (nearbyInfo *NearbyAmenitiesInfo) getGrocers(latitude, longitude float64) (err error) {
	matrix, err := getDistanceMatrix(latitude, longitude, "grocery_or_supermarket")

	if err != nil {
		return
	}

	nearbyInfo.Grocers = *matrix

	return
}

func (nearbyInfo *NearbyAmenitiesInfo) getCafes(latitude, longitude float64) (err error) {
	matrix, err := getDistanceMatrix(latitude, longitude, "cafe")

	if err != nil {
		return
	}

	nearbyInfo.Cafes = *matrix

	return
}

func (nearbyInfo *NearbyAmenitiesInfo) getGyms(latitude, longitude float64) (err error) {
	matrix, err := getDistanceMatrix(latitude, longitude, "gym")

	if err != nil {
		return
	}

	nearbyInfo.Gyms = *matrix

	return
}

func (nearbyInfo *NearbyAmenitiesInfo) getSchools(latitude, longitude float64) (err error) {
	matrix, err := getDistanceMatrix(latitude, longitude, "school")

	if err != nil {
		return
	}

	nearbyInfo.Schools = *matrix

	return
}

func (nearbyInfo *NearbyAmenitiesInfo) getDepartmentStores(latitude, longitude float64) (err error) {
	matrix, err := getDistanceMatrix(latitude, longitude, "department_store")

	if err != nil {
		return
	}

	nearbyInfo.DepartmentStores = *matrix

	return
}

func (nearbyInfo *NearbyAmenitiesInfo) getMalls(latitude, longitude float64) (err error) {
	matrix, err := getDistanceMatrix(latitude, longitude, "shopping_mall")

	if err != nil {
		return
	}

	nearbyInfo.Malls = *matrix

	return
}

func (nearbyInfo *NearbyAmenitiesInfo) getBars(latitude, longitude float64) (err error) {
	matrix, err := getDistanceMatrix(latitude, longitude, "bar")

	if err != nil {
		return
	}

	nearbyInfo.Bars = *matrix

	return
}

func getDistanceMatrix(latitude, longitude float64, nearbyType string) (matrix *distancematrix.Response, err error) {
	nearbyRequiredParams := nearbysearch.RequiredParams{
		APIKey: GoogleAPIKey,
		Location: maps.LatLngLocation{
			Latitude: latitude,
			Longitude: longitude,
		},
	}

	nearbyRankByParam := nearbysearch.OptionalRankByParam{
		RankBy: "distance",
	}

	nearbyTypesParam := nearbysearch.OptionalTypesParam{
		Types: []string{nearbyType},
	}

	nearbyResponse, err := nearbysearch.NearbySearch(&nearbyRequiredParams, &nearbyRankByParam, &nearbyTypesParam)

	if err != nil {
		return
	}

	var locations maps.Locations

	for index, result := range nearbyResponse.Results {
		locations = append(locations, result.Geometry.Location)

		if index == 4 {
			break
		}
	}

	matrixRequiredParams := distancematrix.RequiredParams{
		Origins: maps.Locations{maps.LatLngLocation{
			Latitude: latitude,
			Longitude: longitude,
		}},
		Destinations: locations,
	}

	matrixModeParam := distancematrix.OptionalModeParam{
		Mode: "walking",
	}

	for {
		matrix, err = distancematrix.DistanceMatrix(&matrixRequiredParams, &matrixModeParam)

		if err != nil {
			return
		}

		if matrix.Status != "OVER_QUERY_LIMIT" {
			break
		}

		time.Sleep(1 * time.Second)
	}

	for index, result := range nearbyResponse.Results {
		matrix.DestinationAddresses[index] = result.Name

		if index == 4 {
			break
		}
	}

	return
}

func GetNearbyAmenitiesInfo(latitude, longitude float64) (info *NearbyAmenitiesInfo, err error) {
	info = new(NearbyAmenitiesInfo)

	if err = info.getLandmarks(latitude, longitude); err != nil {
		return
	}

	if err = info.getBusStops(latitude, longitude); err != nil {
		return
	}

	if err = info.getTrainStations(latitude, longitude); err != nil {
		return
	}

	if err = info.getGrocers(latitude, longitude); err != nil {
		return
	}

	if err = info.getCafes(latitude, longitude); err != nil {
		return
	}

	if err = info.getGyms(latitude, longitude); err != nil {
		return
	}

	if err = info.getSchools(latitude, longitude); err != nil {
		return
	}

	if err = info.getDepartmentStores(latitude, longitude); err != nil {
		return
	}

	if err = info.getMalls(latitude, longitude); err != nil {
		return
	}

	err = info.getBars(latitude, longitude)

	return
}

type Scraper struct {
	GeocodeInfo *geocoding.Info
	NBNInfo *nbn.Info
	RealEstateComAuInfo *realestatecomau.Info
	ADSLInfo *adsl.Info
	NearbyAmenitiesInfo *NearbyAmenitiesInfo
	Address string
	MapsEmbed string
	Notes string
}

var db *bolt.DB

func New(address string) (scraper *Scraper, scraped bool, err error) {
	scraper = &Scraper{
		Address: address,
	}

	if db == nil {
		db, err = bolt.Open("dng.db", 0666, nil)

		err = db.Update(func (tx *bolt.Tx) error {
			tx.CreateBucket([]byte("addresses"))

			return nil
		})
	}

	var value []byte

	err = db.View(func (tx *bolt.Tx) error {
		value = tx.Bucket([]byte("addresses")).Get([]byte(address))

		return nil
	})

	if value == nil {
		err = scraper.Scrape()
		scraped = true
	} else {
		err = json.Unmarshal(value, scraper)
	}

	return
}

func (scraper *Scraper) Save() error {
	return db.Update(func (tx *bolt.Tx) error {
		value, err := json.Marshal(scraper)

		if err != nil {
			return err
		}

		return tx.Bucket([]byte("addresses")).Put([]byte(scraper.Address), value)
	})
}

func (scraper *Scraper) Scrape() (err error) {
	defer func() {
		if err == nil {
			err = scraper.Save()
		}
	}()

	scraper.GeocodeInfo, err = geocoding.Geocode(scraper.Address)

	if err != nil {
		return
	}

	if scraper.GeocodeInfo.Status != "OK" {
		err = errors.New("geocode failed")

		return
	}

	lat := scraper.GeocodeInfo.Results[0].Geometry.Location.Latitude
	lng := scraper.GeocodeInfo.Results[0].Geometry.Location.Longitude

	scraper.NBNInfo, err = nbn.RolloutInfo(lat, lng)

	if err != nil {
		log.Printf("could not get NBN rollout info for %v", scraper.Address)
	}

	if scraper.NBNInfo.ServingArea.ServiceStatus == "" {
		scraper.NBNInfo.ServingArea.ServiceStatus = "unavailable"
	}

	scraper.RealEstateComAuInfo, err = realestatecomau.GetInfo(scraper.Address)

	if err != nil {
		log.Printf("could not get real estate info for %v", scraper.Address)
	} else {
		err = scraper.RealEstateComAuInfo.GetImages()

		if err != nil {
			log.Printf("could not get real estate images for %v", scraper.Address)
		}

		err = scraper.RealEstateComAuInfo.GetInspections()

		if err != nil {
			log.Printf("could not get real estate inspections for %v", scraper.Address)
		}
	}

	scraper.ADSLInfo, err = adsl.Lookup(scraper.Address)

	if err != nil {
		log.Printf("could not get ADSL info for %v", scraper.Address)
	}

	scraper.MapsEmbed = fmt.Sprintf(MapsEmbedURL, GoogleAPIKey, url.QueryEscape(scraper.Address))

	scraper.NearbyAmenitiesInfo, err = GetNearbyAmenitiesInfo(lat, lng)

	if err != nil {
		log.Println(err)
	}

	// no critical errors at this point.
	err = nil

	return
}
