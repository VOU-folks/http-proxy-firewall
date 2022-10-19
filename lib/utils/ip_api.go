package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/oschwald/maxminddb-golang"
)

type IPAPIResponse struct {
	Status      string  `json:"status"`
	CountryCode string  `json:"countryCode"`
	Country     string  `json:"country"`
	Region      string  `json:"region"`
	RegionName  string  `json:"regionName"`
	City        string  `json:"city"`
	ZIP         string  `json:"zip"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	Timezone    string  `json:"timezone"`
	ISP         string  `json:"isp"`
	Org         string  `json:"org"`
	As          string  `json:"as"`
	Query       string  `json:"query"`
}

func ResolveUsingIPAPI(ip string) *IPAPIResponse {
	var err error

	url := fmt.Sprintf("http://ip-api.com/json/%s", ip)
	resp, err := http.Get(url)
	if err != nil {
		log.Println("ResolveUsingIPAPI", err.Error())
		return nil
	}

	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		log.Println("ResolveUsingIPAPI", err.Error())
		return nil
	}

	var ipApiResponse *IPAPIResponse
	err = json.Unmarshal(body, &ipApiResponse)
	if err != nil {
		log.Println("ResolveUsingIPAPI", err.Error())
		return nil
	}

	return ipApiResponse
}

func init() {
	var err error

	cwd, _ := os.Getwd()
	maxmindFile := cwd + "/files/geo.mmdb"
	maxMindDB, err = maxminddb.Open(maxmindFile)
	if err != nil {
		log.Println("Cannot read maxmind file", maxmindFile, err.Error())
		maxMindDB = nil
	}
}

var maxMindDB *maxminddb.Reader

type MaxMindResult struct {
	Country struct {
		ISOCode string `maxminddb:"iso_code"`
		Names   struct {
			EN string `maxminddb:"en"`
		} `maxminddb:"names"`
	} `maxminddb:"country"`
}

func ResolveUsingMaxMindAPI(ipAddress string) *IPAPIResponse {
	if maxMindDB == nil {
		return nil
	}

	var lookupResult *MaxMindResult
	ip := net.ParseIP(ipAddress)
	maxMindDB.Lookup(ip, &lookupResult)

	return &IPAPIResponse{
		Country:     lookupResult.Country.Names.EN,
		CountryCode: lookupResult.Country.ISOCode,
	}
}
