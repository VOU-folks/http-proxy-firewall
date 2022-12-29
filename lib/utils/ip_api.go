package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"time"

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

var maxmindUpdatePeriod = time.Hour * 24

func init() {
	go func() {
		for {
			initializeDB()
			time.Sleep(maxmindUpdatePeriod)
		}
	}()
}

func initializeDB() {
	var err error

	cwd, _ := os.Getwd()
	filesDir := cwd + "/files"
	maxmindFileName := "geo.mmdb"
	maxmindFile := filesDir + "/" + maxmindFileName

	downloadGeoDB(filesDir, maxmindFileName)

	maxMindDB, err = maxminddb.Open(filesDir + "/" + maxmindFileName)
	if err != nil {
		log.Println("Cannot read maxmind file", maxmindFile, err.Error())
		maxMindDB = nil
	}
}

func downloadGeoDB(destDir string, destFileName string) {
	licenseKey := GetEnv("MAXMIND_LICENSE_KEY")
	source := fmt.Sprintf(
		"https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-Country&license_key=%s&suffix=tar.gz",
		licenseKey,
	)

	client := http.Client{
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.URL.Opaque = r.URL.Path
			return nil
		},
	}
	resp, err := client.Get(source)
	if err != nil {
		log.Println("downloadGeoDB", err.Error())
	}
	defer resp.Body.Close()

	ExtractMMDBFromTarGz(resp.Body, destDir)
	FindMMDBAndMove(destDir, destDir, destFileName)
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

func ResolveUsingMaxMindAPI(ipAddress string) (IPAPIResponse, bool) {
	result := IPAPIResponse{
		Country:     "",
		CountryCode: "",
	}

	if maxMindDB == nil {
		return result, false
	}

	var lookupResult MaxMindResult
	ip := net.ParseIP(ipAddress)
	maxMindDB.Lookup(ip, &lookupResult)

	result.Country = lookupResult.Country.Names.EN
	result.CountryCode = lookupResult.Country.ISOCode

	return result, true
}
