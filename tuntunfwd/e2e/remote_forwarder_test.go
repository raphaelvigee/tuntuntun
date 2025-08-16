package e2e

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"tuntuntun"
	"tuntuntun/tuntunfwd"
	"tuntuntun/tuntunmux"

	"github.com/stretchr/testify/suite"
)

type RemoteForwarderSuite struct {
	suite.Suite

	fwdTargetAddr string
}

func TestRemoteForwarder(t *testing.T) {
	suite.Run(t, new(RemoteForwarderSuite))
}

func (suite *RemoteForwarderSuite) setupTargetServer() {
	suite.T().Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello"))
	}))
	suite.T().Cleanup(srv.Close)

	suite.fwdTargetAddr = srv.Listener.Addr().String()
}

func (suite *RemoteForwarderSuite) setupServer(factory func(handler tuntuntun.Handler) http.Handler, mux bool) *httptest.Server {
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

func (suite *RemoteForwarderSuite) SetupTest() {
	suite.setupTargetServer()
}

func (suite *RemoteForwarderSuite) suiteRunSanity(tt Setup, mux bool) {
	srv := suite.setupServer(tt.Server, mux)

	opener := tt.Opener(srv.URL)
	if mux {
		ttm := tuntunmux.NewClient(opener)
		suite.T().Cleanup(func() {
			_ = ttm.Close()
		})
		opener = ttm
	}

	f := tuntunfwd.NewRemoteForwarder(opener)

	err := f.Start(suite.fwdTargetAddr, ":0")
	suite.Require().NoError(err)
	suite.T().Cleanup(func() {
		_ = f.Close()
	})

	res, err := http.Get("http://" + f.LocalAddr().String())
	suite.Require().NoError(err)
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	suite.Require().NoError(err)
	suite.Assert().Equal("hello", string(body))
}

func (suite *RemoteForwarderSuite) TestSanity() {
	for name, tt := range setup {
		suite.Run(name, func() {
			suite.suiteRunSanity(tt, false)
		})

		continue
		suite.Run(name+"+mux", func() {
			suite.suiteRunSanity(tt, true)
		})
	}
}
