package e2e

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"tuntuntun"
	"tuntuntun/tuntunfwd"
	"tuntuntun/tuntunmux"

	"github.com/stretchr/testify/suite"
)

type DialerSuite struct {
	suite.Suite

	fwdTargetUrl string
}

func TestDialer(t *testing.T) {
	suite.Run(t, new(DialerSuite))
}

func (suite *DialerSuite) setupTargetServer() {
	suite.T().Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello"))
	}))
	suite.T().Cleanup(srv.Close)

	suite.T().Log("target server: ", srv.URL)

	suite.fwdTargetUrl = srv.URL
}

func (suite *DialerSuite) setupServer(factory func(handler tuntuntun.Handler) http.Handler, mux bool) *httptest.Server {
	suite.T().Helper()

	var tth tuntuntun.Handler = tuntunfwd.NewServer(tuntunfwd.ServerConfig{
		AllowServerForward: func(ctx context.Context, addr string) error {
			return nil
		},
	})
	if mux {
		tth = tuntunmux.NewServer(tth)
	}

	h := factory(tth)

	srv := httptest.NewServer(h)
	suite.T().Cleanup(srv.Close)

	suite.T().Log("tun server: ", srv.URL)

	return srv
}

func (suite *DialerSuite) SetupTest() {
	suite.setupTargetServer()
}

func (suite *DialerSuite) suiteRunSanity(tt Setup, mux bool) {
	srv := suite.setupServer(tt.Server, mux)

	opener := tt.Opener(srv.URL)
	if mux {
		ttm := tuntunmux.NewClient(opener)
		defer ttm.Close()
		opener = ttm
	}

	ctx := suite.T().Context()
	//ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	//defer cancel()

	client := &http.Client{Transport: &http.Transport{
		DialContext: func(ctx context.Context, network string, addr string) (net.Conn, error) {
			return tuntunfwd.RemoteDialContext(ctx, opener, addr)
		},
	}}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, suite.fwdTargetUrl, nil)
	suite.Require().NoError(err)

	res, err := client.Do(req)
	suite.Require().NoError(err)
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	suite.Require().NoError(err)
	suite.Assert().Equal("hello", string(body))
}

func (suite *DialerSuite) TestSanity() {
	for name, tt := range setup {
		suite.Run(name, func() {
			suite.suiteRunSanity(tt, false)
		})

		suite.Run(name+"+mux", func() {
			suite.suiteRunSanity(tt, true)
		})
	}
}
