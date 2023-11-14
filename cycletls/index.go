package cycletls

import (
	"bytes"
	"encoding/json"
	"flag"
	http "github.com/Danny-Dasilva/fhttp"
	"github.com/gorilla/websocket"
	"io/ioutil"
	"log"
	nhttp "net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
)

// Options sets CycleTLS client options
type Options struct {
	URL             string            `json:"url"`
	Method          string            `json:"method"`
	Headers         map[string]string `json:"headers"`
	Body            string            `json:"body"`
	Ja3             string            `json:"ja3"`
	UserAgent       string            `json:"userAgent"`
	Proxy           string            `json:"proxy"`
	Cookies         []Cookie          `json:"cookies"`
	Timeout         int               `json:"timeout"`
	DisableRedirect bool              `json:"disableRedirect"`
	HeaderOrder     []string          `json:"headerOrder"`
	OrderAsProvided bool              `json:"orderAsProvided"` //TODO
}

type cycleTLSRequest struct {
	RequestID string  `json:"requestId"`
	Options   Options `json:"options"`
}

// rename to request+client+options
type fullRequest struct {
	req     *http.Request
	client  http.Client
	options cycleTLSRequest
}

// CycleTLS creates full request and response
type CycleTLS struct {
	ReqChan  chan fullRequest
	RespChan chan []byte
}

// ready Request
func processRequest(request cycleTLSRequest) (result fullRequest) {

	var browser = browser{
		JA3:       request.Options.Ja3,
		UserAgent: request.Options.UserAgent,
		Cookies:   request.Options.Cookies,
	}

	client, err := newClient(
		browser,
		request.Options.Timeout,
		request.Options.DisableRedirect,
		request.Options.UserAgent,
		request.Options.Proxy,
	)
	if err != nil {
		log.Fatal(err)
	}

	req, err := http.NewRequest(strings.ToUpper(request.Options.Method), request.Options.URL, strings.NewReader(request.Options.Body))
	if err != nil {
		log.Fatal(err)
	}
	headerorder := []string{}
	//master header order, all your headers will be ordered based on this list and anything extra will be appended to the end
	//if your site has any custom headers, see the header order chrome uses and then add those headers to this list
	if len(request.Options.HeaderOrder) > 0 {
		//lowercase headers
		for _, v := range request.Options.HeaderOrder {
			lowercasekey := strings.ToLower(v)
			headerorder = append(headerorder, lowercasekey)
		}
	} else {
		headerorder = append(headerorder,
			"host",
			"connection",
			"cache-control",
			"device-memory",
			"viewport-width",
			"rtt",
			"downlink",
			"ect",
			"sec-ch-ua",
			"sec-ch-ua-mobile",
			"sec-ch-ua-full-version",
			"sec-ch-ua-arch",
			"sec-ch-ua-platform",
			"sec-ch-ua-platform-version",
			"sec-ch-ua-model",
			"upgrade-insecure-requests",
			"user-agent",
			"accept",
			"sec-fetch-site",
			"sec-fetch-mode",
			"sec-fetch-user",
			"sec-fetch-dest",
			"referer",
			"accept-encoding",
			"accept-language",
			"cookie",
		)
	}

	headermap := make(map[string]string)
	//TODO: Shorten this
	headerorderkey := []string{}
	for _, key := range headerorder {
		for k, v := range request.Options.Headers {
			lowercasekey := strings.ToLower(k)
			if key == lowercasekey {
				headermap[k] = v
				headerorderkey = append(headerorderkey, lowercasekey)
			}
		}

	}
	headeOrder := parseUserAgent(request.Options.UserAgent).HeaderOrder

	//ordering the pseudo headers and our normal headers
	req.Header = http.Header{
		http.HeaderOrderKey:  headerorderkey,
		http.PHeaderOrderKey: headeOrder,
	}
	//set our Host header
	u, err := url.Parse(request.Options.URL)
	if err != nil {
		panic(err)
	}

	//append our normal headers
	for k, v := range request.Options.Headers {
		if k != "Content-Length" {
			req.Header.Set(k, v)
		}
	}
	req.Header.Set("Host", u.Host)
	req.Header.Set("user-agent", request.Options.UserAgent)
	return fullRequest{req: req, client: client, options: request}

}

// func dispatcher(res fullRequest) {
// 	defer res.client.CloseIdleConnections()

// 	resp, err := res.client.Do(res.req)
// 	if err != nil {

// 		parsedError := parseError(err)

// 		headers := make(map[string]string)
// 		return Response{res.options.RequestID, parsedError.StatusCode, parsedError.ErrorMsg + "-> \n" + string(err.Error()), headers}, nil //normally return error here

// 	}
// 	defer resp.Body.Close()

// 	encoding := resp.Header["Content-Encoding"]
// 	content := resp.Header["Content-Type"]

// 	bodyBytes, err := ioutil.ReadAll(resp.Body)
// 	if err != nil {
// 		log.Print("Parse Bytes" + err.Error())
// 		return response, err
// 	}

// 	Body := DecompressBody(bodyBytes, encoding, content)
// 	headers := make(map[string]string)

// 	for name, values := range resp.Header {
// 		if name == "Set-Cookie" {
// 			headers[name] = strings.Join(values, "/,/")

// 		} else {
// 			for _, value := range values {
// 				headers[name] = value
// 			}
// 		}
// 	}

// 	Response{res.options.RequestID, resp.StatusCode, Body, headers}
// }


func dispatcherAsync(res fullRequest, chanWrite chan []byte) {
	defer res.client.CloseIdleConnections()

	var b bytes.Buffer

	b.Write([]byte(res.options.RequestID))

	chanWrite <- b.Bytes();

	// @TODO: When does this trigger an error ?
	// Are we sure that the parsedError will include headers and a satus code ?
	// resp, err := res.client.Do(res.req)

	// if err != nil {

	// 	parsedError := parseError(err)

	// 	headers := make(map[string]string)

	// 	// return Response{res.options.RequestID, parsedError.StatusCode, parsedError.ErrorMsg + "-> \n" + string(err.Error()), headers}, nil //normally return error here
	// 	return;
	// }

	// defer resp.Body.Close()

	// encoding := resp.Header["Content-Encoding"]
	// content := resp.Header["Content-Type"]

	// bodyBytes, err := ioutil.ReadAll(resp.Body)
	// if err != nil {
	// 	log.Print("Parse Bytes" + err.Error())
	// 	return response, err
	// }

	// Body := DecompressBody(bodyBytes, encoding, content)
	// headers := make(map[string]string)

	// for name, values := range resp.Header {
	// 	if name == "Set-Cookie" {
	// 		headers[name] = strings.Join(values, "/,/")

	// 	} else {
	// 		for _, value := range values {
	// 			headers[name] = value
	// 		}
	// 	}
	// }
}

func readSocket(chanRead chan fullRequest, wsSocket *websocket.Conn) {
	for {
		_, message, err := wsSocket.ReadMessage()

		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				return
			}
			log.Print("Socket Error", err)
			return
		}

		request := new(cycleTLSRequest)
		err = json.Unmarshal(message, &request)

		if err != nil {
			log.Print("Unmarshal Error", err)
			return
		}

		chanRead <- processRequest(*request)
	}
}

// Worker
func readProcess(chanRead chan fullRequest, chanWrite chan []byte) {
	for request := range chanRead {
		go dispatcherAsync(request, chanWrite)
	}
}

func writeSocket(chanWrite chan []byte, wsSocket *websocket.Conn) {
	for buf := range chanWrite {
		err := wsSocket.WriteMessage(websocket.BinaryMessage, buf)

		if err != nil {
			log.Print("Socket WriteMessage Failed" + err.Error())
			continue
		}
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// WSEndpoint exports the main cycletls function as we websocket connection that clients can connect to
func WSEndpoint(w nhttp.ResponseWriter, r *nhttp.Request) {
	upgrader.CheckOrigin = func(r *nhttp.Request) bool { return true }

	// upgrade this connection to a WebSocket
	// connection
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		//Golang Received a non-standard request to this port, printing request
		var data map[string]interface{}
		bodyBytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Print("Invalid Request: Body Read Error" + err.Error())
		}
		err = json.Unmarshal(bodyBytes, &data)
		if err != nil {
			log.Print("Invalid Request: Json Conversion failed ")
		}
		body, err := PrettyStruct(data)
		if err != nil {
			log.Print("Invalid Request:", err)
		}
		headers, err := PrettyStruct(r.Header)
		if err != nil {
			log.Fatal(err)
		}
		log.Println(headers)
		log.Println(body)

	} else {
		chanRead := make(chan fullRequest)
		chanWrite := make(chan []byte)

		go readSocket(chanRead, ws)
		go readProcess(chanRead, chanWrite)

		// Run as main thread
		writeSocket(chanWrite, ws)
	}
}

func setupRoutes() {
	nhttp.HandleFunc("/", WSEndpoint)
}

func main() {
	port, exists := os.LookupEnv("WS_PORT")
	var addr *string
	if exists {
		addr = flag.String("addr", ":"+port, "http service address")
	} else {
		addr = flag.String("addr", ":9112", "http service address")
	}

	runtime.GOMAXPROCS(runtime.NumCPU())

	setupRoutes()
	log.Fatal(nhttp.ListenAndServe(*addr, nil))
}
