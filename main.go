package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
)

var errHTTPForbidden error = errors.New("Forbidden")
var errHTTPRateLimited error = errors.New("Rate Limited")
var errHTTPMisc error = errors.New("HTTP request not OK")
var cfg Config
var url string
var csrf string
var body []byte

type purchasesRequest struct {
	ExpectedCurrency uint `json:"expectedCurrency,omitempty"`
	ExpectedPrice    uint `json:"expectedPrice,omitempty"`
	ExpectedSellerID uint `json:"expectedSellerId,omitempty"`
	UAID             uint `json:"userAssetId,omitempty"`
}

type purchasesResponse struct {
	Purchased bool   `json:"purchased"`
	Reason    string `json:"reason"`
}

func makeRequest(c *fasthttp.Client, url string, csrf string, cookie string, body []byte) ([]byte, error) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetBody(body)
	req.Header.SetMethod("POST")
	req.Header.SetContentType("application/json")
	req.Header.SetCookie(".ROBLOSECURITY", cookie)
	req.Header.Set("x-csrf-token", csrf)
	req.SetRequestURI(url)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	c.Do(req, resp)
	log.Println(resp.StatusCode())
	statusCode := resp.StatusCode()
	if statusCode != http.StatusOK {
		switch statusCode {
		case http.StatusForbidden:
			return resp.Header.Peek("x-csrf-token"), errHTTPForbidden
		case http.StatusTooManyRequests:
			return []byte{}, errHTTPRateLimited
		default:
			return []byte{}, errHTTPMisc
		}
	}

	return resp.Body(), nil
}

func createProxyClients() []*fasthttp.Client {
	file, err := os.Open("proxies.txt")
	if err != nil {
		log.Fatalln(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var clients []*fasthttp.Client
	for scanner.Scan() {
		clients = append(clients, &fasthttp.Client{
			TLSConfig: &tls.Config{InsecureSkipVerify: true},
			Dial:      fasthttpproxy.FasthttpHTTPDialer(scanner.Text()),
		})
	}

	return clients
}

func setVars() {
	url = fmt.Sprintf("https://economy.roblox.com/v1/purchases/products/%d", cfg.ItemID)
	body, _ = json.Marshal(purchasesRequest{
		ExpectedCurrency: 1,
		ExpectedPrice:    cfg.Price,
		ExpectedSellerID: cfg.SellerID,
		UAID:             cfg.UAID,
	})

	client := &fasthttp.Client{
		TLSConfig: &tls.Config{InsecureSkipVerify: true},
		Dial:      fasthttpproxy.FasthttpProxyHTTPDialer(),
	}
	csrfBytes, _ := makeRequest(client, url, "", cfg.Cookie, []byte{})
	csrf = string(csrfBytes)

}

func doRequests(c *fasthttp.Client, ch chan<- bool) {
	for {
		respBody, err := makeRequest(c, url, csrf, cfg.Cookie, body)
		if err != nil {
			switch err {
			case errHTTPRateLimited:
				time.Sleep(time.Second * 5)
				continue
			default:
				log.Println(err)
				return
			}
		}
		var resp purchasesResponse
		json.Unmarshal(respBody, &resp)
		log.Println(resp.Reason)
		if resp.Purchased {
			ch <- true
			return
		}
	}

}

func main() {
	var err error
	cfg, err = ParseConfig("config.yml")
	if err != nil {
		log.Fatal(err)
	}

	setVars()

	clients := createProxyClients()

	done := make(chan bool)

	for _, c := range clients {
		go doRequests(c, done)
	}

	<-done

}
