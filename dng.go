package main

import (
	"net/http"
	"log"
	"encoding/json"
	"github.com/garfunkel/dng/scraper"
	"flag"
	"fmt"
)

const (
	host = "localhost"
	port = "8080"
)

var addresses = []string{
	"39 Porter Road Engadine, NSW 2233",
	"15/28A Henry Street Ashfield NSW 2131",
	"59/47 Hampstead Road, Homebush West, NSW 2140",
	"18/52 Parramatta Road, Homebush, NSW 2140",
	"27/8-12 Marlborough Road Homebush West NSW 2140",
	"4/63 Gipps St, Drummoyne, NSW 2047",
	"18/19 Johnston Street Annandale",
	"43/29-45 Parramatta Road, Concord, NSW 2137",
	"4/421 Liverpool Road, Ashfield, NSW 2131",
}

func index(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "templates/index.html")
}

func getAddresses(w http.ResponseWriter, r *http.Request) {
	data, err := json.Marshal(addresses)

	if err != nil {
		log.Fatal(err)
	}

	w.Write(data)
}

func getAddressInfo(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()

	if err != nil {
		log.Fatal(err)
	}

	addressValue, ok := r.Form["address"]

	if !ok {
		log.Fatal("no address in querystring")
	}

	address := addressValue[0]
	scrape, _, err := scraper.New(address)

	if err != nil {
		log.Println(err)
	}

	data, err := json.Marshal(scrape)

	w.Write(data)
}

func getStatic(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "." + r.URL.Path)
}

func refreshInfo(c chan string, address string) {
	scrape, scraped, err := scraper.New(address)

	if err != nil {
		log.Println(err)
	}

	if !scraped {
		scrape.Scrape()
	}

	c <- address
}

func main() {
	refresh := flag.Bool("r", false, "refresh real estate information")

	flag.Parse()

	if *refresh {
		log.Printf("refreshing real estate information")

		c := make(chan string)

		for _, address := range addresses {
			log.Printf("refreshing %v", address)

			go refreshInfo(c, address)
		}

		num := 0

		for address := range c {
			num += 1

			log.Printf("refreshed %v", address)

			if num == len(addresses) {
				break
			}
		}

		close(c)

		log.Printf("refreshed real estate information")
	}

	http.HandleFunc("/", index)
	http.HandleFunc("/addresses", getAddresses)
	http.HandleFunc("/addressinfo", getAddressInfo)
	http.HandleFunc("/static/", getStatic)

	log.Printf("listening on http://%v:%v", host, port)

	log.Fatal(http.ListenAndServe(fmt.Sprintf("%v:%v", host, port), nil))
}
