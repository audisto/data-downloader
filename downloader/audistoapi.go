package downloader

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
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
)

// AudistoAPIClient a struct holding all information requred to construct a URL with query params for Audisto API
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

	// meta
	requestMethod string
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
func (api *AudistoAPIClient) GetRelativePath() string {
	endpoint := strings.Trim(AudistoAPIEndpoint, "/")
	return fmt.Sprintf(
		"/%s/%s/%v/%s",
		AudistoAPIVersion, endpoint, api.CrawlID, api.Mode)
}

// GetQueryParams use net/url package to construct query params
func (api *AudistoAPIClient) GetQueryParams() url.Values {
	urlQueryParams := url.Values{}

	if api.Deep {
		urlQueryParams.Add("deep", "1")
	} else {
		urlQueryParams.Add("deep", "0")
	}

	if api.Filter != "" {
		urlQueryParams.Add("filter", api.Filter)
	}

	if api.Order != "" {
		urlQueryParams.Add("order", api.Order)
	}

	if api.Output == "" {
		urlQueryParams.Add("output", DefaultOutputFormat)
	} else {
		urlQueryParams.Add("output", api.Output)
	}

	urlQueryParams.Add("chunk", strconv.FormatUint(api.ChunkNumber, 10))
	urlQueryParams.Add("chunk_size", strconv.FormatUint(api.ChunkSize, 10))
	return urlQueryParams
}

// GetFullQueryURL returns the full url for interacting with Audisto API, INCLUDING query params
func (api *AudistoAPIClient) GetFullQueryURL() string {
	return api.GetQueryParams().Encode()
}

// SetChunkSize set AudistoAPI.ChunkSize to a new size
func (api *AudistoAPIClient) SetChunkSize(size uint64) {
	if size == 0 {
		api.ChunkSize = DefaultChunkSize
	} else {
		api.ChunkSize = size
	}
}

// SetNextChunk set AudistoAPI.ChunkNumber to the next chunk number
func (api *AudistoAPIClient) SetNextChunk(number uint64) {
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

// GetRequestURL returns a validated instance of url.URL, and an error if the validation fails
func (api *AudistoAPIClient) GetRequestURL() (*url.URL, error) {
	domain := api.GetBaseURL()
	requestURL, err := url.Parse(domain)
	if err != nil {
		return requestURL, err
	}
	requestURL.Path = api.GetRelativePath()
	requestURL.RawQuery = api.GetQueryParams().Encode()
	return requestURL, nil
}
