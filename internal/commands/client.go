package commands

import (
	"context"
	"net/http"
	"net/url"

	"github.com/eliben/gemini-cli/internal/apikey"
	"github.com/google/generative-ai-go/genai"
	"github.com/spf13/cobra"
	"google.golang.org/api/option"
)

// newGenaiClient creates a new genai.Client given the configuration of
// cmd flags (for API key, proxy selection, etc.)
func newGenaiClient(ctx context.Context, cmd *cobra.Command) (*genai.Client, error) {
	key := apikey.Get(cmd)

	var clientOpts []option.ClientOption
	if proxyURL, _ := cmd.Flags().GetString("proxy"); len(proxyURL) > 0 {
		c := &http.Client{Transport: &proxyRoundTripper{
			APIKey:   key,
			ProxyURL: proxyURL,
		}}

		clientOpts = append(clientOpts, option.WithHTTPClient(c))
	} else {
		clientOpts = append(clientOpts, option.WithAPIKey(key))
	}

	client, err := genai.NewClient(ctx, clientOpts...)
	return client, err
}

type proxyRoundTripper struct {
	// APIKey is the API Key to set on requests.
	APIKey string

	// ProxyURL is the URL of the proxy server. If empty, no proxy is used.
	ProxyURL string
}

func (t *proxyRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()

	if t.ProxyURL != "" {
		proxyURL, err := url.Parse(t.ProxyURL)
		if err != nil {
			return nil, err
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	newReq := req.Clone(req.Context())
	vals := newReq.URL.Query()
	vals.Set("key", t.APIKey)
	newReq.URL.RawQuery = vals.Encode()

	resp, err := transport.RoundTrip(newReq)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
