package downloader

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	// AudistoAPIDomain the domain name endpoint for Audisto API
	AudistoAPIDomain = "api.audisto.com"

	// AudistoAPIEndpoint URL enpoint for Audisto API, put "" or "/" string if the endpoint is at the root domain
	AudistoAPIEndpoint = "/crawls/"

	// AudistoAPIVersion the version of Audisto API version this downloader will talk to
	AudistoAPIVersion = "2.0"

	// EndpointSchema http or https, this probably wont change, hence it is set here
	EndpointSchema = "https"

	// DefaultRequestMethod used when http request method is not explicitly set
	DefaultRequestMethod = "GET"

	// DefaultOutputFormat the default formatting or file extension for the response we get from Audisto API if not expilictly set
	DefaultOutputFormat = "tsv"

	// DefaultChunkSize the default chunk size for interacting with Audisto API if NOT expilicty set
	// This should not affect the way throttling works
	DefaultChunkSize = 10000

	// ContentType type of the http request to send using the http client
	ContentType = "application/x-www-form-urlencoded"

	// AcceptEncoding Content encoding for the http request
	AcceptEncoding = "gzip, deflate"

	// ConnectionType is the value of "Connection" http header to be send using the http client
	ConnectionType = "Keep-Alive"
)

// AudistoAPIClient a struct holding all information required to construct a URL with query params for Audisto API
type AudistoAPIClient struct {

	// request path / DSN
	BasePath string
	Username string
	Password string
	Mode     string
	CrawlID  uint64

	// request query params
	Deep        bool
	Filter      string
	Order       string
	Output      string
	ChunkNumber uint64
	ChunkSize   uint64

	// HTTP Client
	httpClient http.Client

	// meta
	requestMethod string
}

// chunk is used to get unmarshal the json containing the total number of chunks
type chunk struct {
	Chunk struct {
		Total uint64 `json:"total"`
		Page  int    `json:"page"`
		Size  int    `json:"size"`
	} `json:"chunk"`
}

// IsValid check if the struct info look good. This does not do any remote request.
func (api *AudistoAPIClient) IsValid() error {

	if api.Username == "" || api.Password == "" || api.CrawlID == 0 {
		return fmt.Errorf("username, password or crawl should NOT be empty")
	}

	if api.Mode != "" && api.Mode != "pages" && api.Mode != "links" {
		return fmt.Errorf("mode has to be 'links' or 'pages'")
	}

	return nil
}

// GetAPIEndpoint constructs the Audisto API endpoint without the query params nor the dsn part.
func (api *AudistoAPIClient) GetAPIEndpoint() string {
	endpoint := strings.Trim(AudistoAPIEndpoint, "/")
	urlParts := []string{AudistoAPIDomain, AudistoAPIVersion, endpoint}
	return strings.Join(urlParts, "/")
}

// GetBaseURL construct the base url for quering Audisto API in the form of:
// username:password@api.audisto.com
func (api *AudistoAPIClient) GetBaseURL() string {
	return fmt.Sprintf(
		"%s://%s:%s@%s",
		EndpointSchema, api.Username, api.Password, api.GetAPIEndpoint())
}

// GetURLPath returns the full url for interacting with Audisto API, WITHOUT query params
// e.g. username:password@api.audisto.com/crawls/pages|links
func (api *AudistoAPIClient) GetURLPath() string {
	return fmt.Sprintf("%s/%v/%s", api.GetBaseURL(), api.CrawlID, api.Mode)
}

// GetRelativePath return the relative path to the api domain name
// e.g. /2.0/crawls/123456/links
func (api *AudistoAPIClient) GetRelativePath() string {
	endpoint := strings.Trim(AudistoAPIEndpoint, "/")
	return fmt.Sprintf(
		"/%s/%s/%v/%s",
		AudistoAPIVersion, endpoint, api.CrawlID, api.Mode)
}

// GetQueryParams use net/url package to construct query params
// If forTheFirstRequest is set to true: chunk_size, deep are set to 0 and the output is forced to be json
// This is used to request the first chunk in json and get total number of elements
func (api *AudistoAPIClient) GetQueryParams(forTheFirstRequest bool) url.Values {
	urlQueryParams := url.Values{}

	if api.Filter != "" {
		urlQueryParams.Add("filter", api.Filter)
	}

	if api.Order != "" {
		urlQueryParams.Add("order", api.Order)
	}

	urlQueryParams.Add("deep", "0")

	if forTheFirstRequest {
		urlQueryParams.Add("chunk", "0")
		urlQueryParams.Add("chunk_size", "1")
		urlQueryParams.Set("output", "json")
	} else {

		if api.Deep {
			urlQueryParams.Set("deep", "1")
		}

		if api.Output == "" {
			urlQueryParams.Add("output", DefaultOutputFormat)
		} else {
			urlQueryParams.Add("output", api.Output)
		}

		urlQueryParams.Add("chunk", strconv.FormatUint(api.ChunkNumber, 10))
		urlQueryParams.Add("chunk_size", strconv.FormatUint(api.ChunkSize, 10))
	}
	return urlQueryParams
}

// GetFullQueryURL returns the full url for interacting with Audisto API, INCLUDING query params
func (api *AudistoAPIClient) GetFullQueryURL(forTheFirstRequest bool) string {
	return fmt.Sprintf("%s?%s", api.GetURLPath(), api.GetQueryParams(forTheFirstRequest).Encode())
}

// SetChunkSize set AudistoAPI.ChunkSize to a new size
func (api *AudistoAPIClient) SetChunkSize(size uint64) {
	if size == 0 {
		api.ChunkSize = DefaultChunkSize
	} else {
		api.ChunkSize = size
	}
}

func (api *AudistoAPIClient) ResetChunkSize() {
	api.ChunkSize = DefaultChunkSize
}

// SetNextChunkNumber set AudistoAPI.ChunkNumber to the next chunk number
func (api *AudistoAPIClient) SetNextChunkNumber(number uint64) {
	api.ChunkNumber = number
}

// GetRequestMethod returns the HTTP request method, GET (by default)
func (api *AudistoAPIClient) GetRequestMethod() string {
	if api.requestMethod == "" {
		return DefaultRequestMethod
	}
	return api.requestMethod
}

// SetRequestMethod sets the HTTP request method for interacting with Audisto API
// Allowed method: GET, POST, PATCH, DELETE
func (api *AudistoAPIClient) SetRequestMethod(method string) error {
	method = strings.ToUpper(method)

	if method != "GET" && method != "POST" && method != "PATCH" && method != "DELETE" {
		return fmt.Errorf("Method not supported: %s", method)
	}
	api.requestMethod = method
	return nil
}

func (api *AudistoAPIClient) SetTargetPageFilter(pageID uint64) {
	api.Filter = fmt.Sprintf("target_page:%d", pageID)
}

// GetRequestURL returns a validated instance of url.URL, and an error if the validation fails
func (api *AudistoAPIClient) GetRequestURL() (*url.URL, error) {

	requestURL, err := url.Parse(api.GetURLPath())
	if err != nil {
		return requestURL, err
	}
	return requestURL, nil
}

// Do execute an http request adding Audisto API header values
// This also do variable validation before executing the request for less http roundtrips
func (api *AudistoAPIClient) Do(request *http.Request) (*http.Response, error) {
	err := api.IsValid()
	if err != nil {
		return nil, err
	}

	bodyParams := url.Values{}
	request.Header = http.Header{}
	request.Header.Add("Content-Length", strconv.Itoa(len(bodyParams.Encode())))
	request.Header.Add("Connection", ConnectionType)
	request.Header.Add("Accept-Encoding", AcceptEncoding)
	request.Header.Add("Content-Type", ContentType)
	return api.httpClient.Do(request)
}

// FetchRawChunk makes an http request to the server for a given chunk
func (api *AudistoAPIClient) FetchRawChunk(forTheFirstRequest bool) ([]byte, int, error) {

	requestURL, err := client.GetRequestURL()
	if err != nil {
		return []byte(""), 0, err
	}
	bodyParameters := url.Values{}
	requestURL.RawQuery = client.GetQueryParams(forTheFirstRequest).Encode()

	debugf("request url: %s", requestURL.String())
	request, err := http.NewRequest(
		api.GetRequestMethod(), requestURL.String(),
		bytes.NewBufferString(bodyParameters.Encode()))
	if err != nil {
		return []byte(""), 0, fmt.Errorf("Failed to get the URL %s: %s", requestURL, err)
	}

	response, err := api.Do(request)
	if err != nil {
		return []byte(""), 0, fmt.Errorf("Failed to get the URL %s: %s", requestURL, err)
	}

	defer response.Body.Close()

	var responseReader io.ReadCloser
	switch response.Header.Get("Content-Encoding") {
	case "gzip":
		decompressedBodyReader, err := gzip.NewReader(response.Body)
		if err != nil {
			return []byte(""), response.StatusCode, err
		}
		responseReader = decompressedBodyReader
		defer responseReader.Close()
	default:
		responseReader = response.Body
	}

	responseBody, err := ioutil.ReadAll(responseReader)
	if err != nil {
		return []byte(""), response.StatusCode, err
	}

	return responseBody, response.StatusCode, nil
}

// FetchTotalElements sets up the request for the first chunk in json,
// containing the total number of elements.
func (api *AudistoAPIClient) FetchTotalElements() ([]byte, int, error) {

	body, statusCode, err := api.FetchRawChunk(true)

	if err != nil {
		return []byte(""), 0, err
	}

	return body, statusCode, nil
}

// GetTotalElements asks the server the total number of elements
func (api *AudistoAPIClient) GetTotalElements() (uint64, error) {
	var body []byte
	var statusCode int

	err := retry(5, 3, func() error {
		var err error
		body, statusCode, err = api.FetchTotalElements()
		if err != nil {
			return err
		}

		errorString, ok := StatusCodesErrors[statusCode]

		if !ok && statusCode >= 400 { // we've got a status code that reflects an error

			switch statusCode {
			case 429, 504: // errors that require sleep
				{
					// 429: multiple requests
					// 504: timeout
					time.Sleep(time.Second * 5)
					return fmt.Errorf(errorString)
				}

			case 401, 403, 404: // no-sleep errors
				{
					return fmt.Errorf(errorString)
				}
			default: // unknown errors
				{
					if statusCode < 500 {
						return fmt.Errorf("Unknown error occurred (code %v)", statusCode)
					}
					// server errors, seelp
					time.Sleep(time.Second * 5)
					return fmt.Errorf("Error while getting total number of elements: %v, server error", statusCode)
				}
			}
		}

		return nil
	})

	if err != nil {
		return 0, err
	}

	var firstChunk chunk
	err = json.Unmarshal(body, &firstChunk)
	if err != nil {
		return 0, err
	}

	return firstChunk.Chunk.Total, nil

}
