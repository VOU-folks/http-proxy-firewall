package utils

import (
	"encoding/json"
	"fmt"
	"io"
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

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

func ResolveUsingIPAPI(ip string) *IPAPIResponse {
	url := fmt.Sprintf("http://ip-api.com/json/%s", ip)
	resp, err := httpClient.Get(url)
	if err != nil {
		log.Printf("ResolveUsingIPAPI: %v\n", err)
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("ResolveUsingIPAPI: %v\n", err)
		return nil
	}

	var ipApiResponse *IPAPIResponse
	if err = json.Unmarshal(body, &ipApiResponse); err != nil {
		log.Printf("ResolveUsingIPAPI: %v\n", err)
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

var geoDBClient = &http.Client{
	CheckRedirect: func(r *http.Request, via []*http.Request) error {
		r.URL.Opaque = r.URL.Path
		return nil
	},
	Timeout: 5 * time.Minute,
}

func initializeDB() {
	cwd, _ := os.Getwd()
	filesDir := cwd + "/files"
	maxmindFileName := "geo.mmdb"

	if err := downloadGeoDB(filesDir, maxmindFileName); err != nil {
		log.Printf("Cannot download maxmind file: %v\n", err)
	}

	var err error
	maxMindDB, err = maxminddb.Open(filesDir + "/" + maxmindFileName)
	if err != nil {
		log.Printf("Cannot read maxmind file: %v\n", err)
		maxMindDB = nil
	}
}

func downloadGeoDB(destDir string, destFileName string) error {
	licenseKey := GetEnv("MAXMIND_LICENSE_KEY")
	source := fmt.Sprintf(
		"https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-Country&license_key=%s&suffix=tar.gz",
		licenseKey,
	)

	resp, err := geoDBClient.Get(source)
	if err != nil {
		return fmt.Errorf("failed to download GeoIP database: %w", err)
	}
	defer resp.Body.Close()

	if err = ExtractMMDBFromTarGz(resp.Body, destDir); err != nil {
		return fmt.Errorf("failed to extract MMDB: %w", err)
	}

	if err = FindMMDBAndMove(destDir, destDir, destFileName); err != nil {
		return fmt.Errorf("failed to move MMDB: %w", err)
	}

	return nil
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

	ip := net.ParseIP(ipAddress)
	if ip == nil {
		return result, false
	}

	var lookupResult MaxMindResult
	if err := maxMindDB.Lookup(ip, &lookupResult); err != nil {
		return result, false
	}

	result.Country = lookupResult.Country.Names.EN
	result.CountryCode = lookupResult.Country.ISOCode

	return result, true
}
