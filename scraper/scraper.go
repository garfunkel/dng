package scraper

import (
	"errors"
	"encoding/json"
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

type PublicTransportInfo struct {
	BusStops distancematrix.Response
	TrainStations distancematrix.Response
}

func (transportInfo *PublicTransportInfo) getBusStops(latitude, longitude float64) (err error) {
	matrix, err := getDistanceMatrix(latitude, longitude, "bus_station")

	if err != nil {
		return
	}

	transportInfo.BusStops = *matrix

	return
} 

func (transportInfo *PublicTransportInfo) getTrainStations(latitude, longitude float64) (err error) {
	matrix, err := getDistanceMatrix(latitude, longitude, "train_station")

	if err != nil {
		return
	}

	transportInfo.TrainStations = *matrix

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

	matrix, err = distancematrix.DistanceMatrix(&matrixRequiredParams, &matrixModeParam)

	for index, result := range nearbyResponse.Results {
		matrix.DestinationAddresses[index] = result.Name

		if index == 4 {
			break
		}
	}

	return
}

func GetPublicTransportInfo(latitude, longitude float64) (info *PublicTransportInfo, err error) {
	info = new(PublicTransportInfo)

	err = info.getBusStops(latitude, longitude)

	if err != nil {
		return
	}

	err = info.getTrainStations(latitude, longitude)

	return
}

type Scraper struct {
	GeocodeInfo *geocoding.Info
	NBNInfo *nbn.Info
	RealEstateComAuInfo *realestatecomau.Info
	ADSLInfo *adsl.Info
	PublicTransportInfo *PublicTransportInfo
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
	defer scraper.Save()

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
	}

	scraper.ADSLInfo, err = adsl.Lookup(scraper.Address)

	if err != nil {
		log.Printf("could not get ADSL info for %v", scraper.Address)
	}

	scraper.MapsEmbed = fmt.Sprintf(MapsEmbedURL, GoogleAPIKey, url.QueryEscape(scraper.Address))

	scraper.PublicTransportInfo, err = GetPublicTransportInfo(lat, lng)

	return
}
