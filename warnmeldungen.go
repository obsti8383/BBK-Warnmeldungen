// warnmeldungen.go
// https://warnung.bund.de/bbk.mowas/gefahrendurchsagen.json
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"time"

	scribble "github.com/nanobox-io/golang-scribble"
)

const (
	USER_AGENT  = "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/65.0.3325.181 Safari/537.36"
	ACCEPT      = "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8"
	ACCEPT_LANG = "en-US,en;q=0.8"
	DBNAME      = "Meldungen"
)

var IoTDB *scribble.Driver

type Warnmeldung struct {
	Identifier    string   `json:"identifier"`
	MsgType       string   `json:"msgType"`
	Sender        string   `json:"sender"`
	Scope         string   `json:"scope"`
	Sent          string   `json:"sent"`
	Status        string   `json:"status"`
	Code          []string `json:"code"`
	Informationen []Info   `json:"info"`
}

type Info struct {
	Severity    string      `json:"severity"`
	Urgency     string      `json:"urgency"`
	Description string      `json:"description"`
	Headline    string      `json:"headline"`
	Event       string      `json:"event"`
	Certainty   string      `json:"certainty"`
	Category    []string    `json:"category"`
	Parameter   []Parameter `json:"parameter"`
	Area        []Area      `json:"area"`
}

type Area struct {
	// ignore "polygon" for now
	AreaDesc string    `json:"areaDesc"`
	Geocode  []GeoCode `json:"geocode"`
}

type GeoCode struct {
	ValueName string `json:"valueName"`
	// ignore "Value" for now
}

type Parameter struct {
	ValueName string `json:"valueName"`
	Value     string `json:"value"`
}

func main() {
	DB, err := scribble.New("DB", nil)
	HandleError(err)

	warnmeldungen, err := GetMeldungen()
	if err != nil {
		fmt.Println("Could not get data due to error", err.Error())
	}

	for _, meldung := range warnmeldungen {

		var geleseneMeldung Warnmeldung
		err = DB.Read(DBNAME, meldung.Identifier, &geleseneMeldung)
		if err != nil {
			// Warnmeldung seems not to exist yet -> write it to DB first
			// write to file based DB
			err = DB.Write(DBNAME, meldung.Identifier, meldung)
			HandleError(err)
			fmt.Println("Neue Meldung:\n", prettyPrint(meldung))
		} else {
			fmt.Println("Alte Meldung gefunden:", meldung.Identifier)
		}
	}
}

func GetMeldungen() (meldungen []Warnmeldung, err error) {
	u, err := url.Parse("https://warnung.bund.de/bbk.mowas/gefahrendurchsagen.json")
	if err != nil {
		return nil, err
	}

	jsonByteArray, err := getJSON(u.String(), nil)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(jsonByteArray, &meldungen)
	if err != nil {
		return nil, err
	}

	return meldungen, err
}

func getJSON(url string, hvals map[string]string) ([]byte, error) {
	d := net.Dialer{}
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DialContext:           d.DialContext,
			MaxIdleConns:          200,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 5 * time.Second,
		},
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("User-Agent", USER_AGENT)
	req.Header.Add("Accept", ACCEPT)
	req.Header.Add("Accept-Language", ACCEPT_LANG)
	if hvals != nil {
		for k, v := range hvals {
			req.Header.Add(k, v)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	} else if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.New(resp.Status)
	}

	in, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	return in, nil
}

func prettyPrint(i interface{}) string {
	s, _ := json.MarshalIndent(i, "", "\t")
	return string(s)
}

func HandleError(err error) {
	if err != nil {
		panic(err)
	}
}
