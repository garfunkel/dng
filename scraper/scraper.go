package scraper

import (
	"errors"
	"encoding/json"
	"github.com/garfunkel/go-nbn"
	"github.com/garfunkel/go-geocode"
	"github.com/garfunkel/go-realestatecomau"
	"github.com/garfunkel/go-adsl"
	"github.com/boltdb/bolt"
	"fmt"
	"net/url"
	"log"
)

const (
	DBPath = "dng.db"
	MapsEmbedURL = "https://www.google.com/maps/embed/v1/place?key=%v&q=%v&zoom=12"
	GoogleAPIKey = "AIzaSyC50lfM-BNpgJMXesZ9qV4Jx6ubTMmwwxA"
)

type Scraper struct {
	GeocodeInfo *geocode.Info
	NBNInfo *nbn.Info
	RealEstateComAuInfo *realestatecomau.Info
	ADSLInfo *adsl.Info
	Address string
	MapsEmbed string
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

func (scraper *Scraper) Scrape() (err error) {
	defer db.Update(func (tx *bolt.Tx) error {
		value, err := json.Marshal(scraper)

		if err != nil {
			return err
		}

		return tx.Bucket([]byte("addresses")).Put([]byte(scraper.Address), value)
	})

	scraper.GeocodeInfo, err = geocode.Geocode(scraper.Address)

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
	}

	scraper.ADSLInfo, err = adsl.Lookup(scraper.Address)

	if err != nil {
		log.Printf("could not get ADSL info for %v", scraper.Address)
	}

	scraper.MapsEmbed = fmt.Sprintf(MapsEmbedURL, GoogleAPIKey, url.QueryEscape(scraper.Address))

	return
}
