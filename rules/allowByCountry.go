package rules

import (
	"net/http"
	"sync"
	"time"
)

func init() {
	ipToCountryStorage = &IpToCountryStorage{
		storage: make(map[string]*IpToCountryRecord),
		mx:      sync.Mutex{},
	}

	ipToCountryService = &IpToCountryService{}
}

type IpToCountryService struct{}

func (ipcs *IpToCountryService) Lookup(ip string) {

}

var ipToCountryService *IpToCountryService

type IpToCountryRecord struct {
	Country   string
	CheckedAt time.Time
}

var ipToCountryExpirationDuration = float64(time.Hour * 48)

func (ipcr IpToCountryRecord) Outdated() bool {
	return time.Now().Sub(ipcr.CheckedAt).Hours() > ipToCountryExpirationDuration
}

func (ipcr IpToCountryRecord) IsAllowed() bool {
	allowed := ipcr.Country == "Azerbaijan" ||
		ipcr.Country == "Turkey" ||
		ipcr.Country == "Georgia" ||
		ipcr.Country == "Russia" ||
		ipcr.Country == "Russian Federation"

	return allowed
}

type IpToCountryStorage struct {
	storage map[string]*IpToCountryRecord
	mx      sync.Mutex
}

func (ipc *IpToCountryStorage) Get(ip string) *IpToCountryRecord {
	ipc.mx.Lock()
	ipcr := ipc.storage[ip]
	ipc.mx.Unlock()

	return ipcr
}

var ipToCountryStorage *IpToCountryStorage

func AllowByCountry(req *http.Request, res http.ResponseWriter, remoteAddr string) bool {
	return true

	if req.Header.Get("Is-Whitelisted") == "true" {
		return true
	}

	ipcRecord := ipToCountryStorage.Get(remoteAddr)
	if ipcRecord != nil {
		if ipcRecord.IsAllowed() {
			res.WriteHeader(403)
			_, _ = res.Write([]byte("Forbidden"))
			return false
		}

		if ipcRecord.Outdated() {
			go ipToCountryService.Lookup(remoteAddr)
		}
		return true
	}

	go ipToCountryService.Lookup(remoteAddr)

	return true
}
