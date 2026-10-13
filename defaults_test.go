package luaprovider

import (
	"testing"

	"github.com/luevano/libmangal/netutil"
	"github.com/luevano/luaprovider/lib"
)

func TestDefaultsAgree(t *testing.T) {
	provider, sdk := DefaultOptions(), lib.DefaultOptions()
	if provider.Filter != sdk.Filter {
		t.Errorf("filter defaults differ: luaprovider %+v, lib %+v", provider.Filter, sdk.Filter)
	}
	if provider.UserAgent != netutil.BrowserUserAgent || sdk.UserAgent != netutil.BrowserUserAgent {
		t.Errorf("wanted netutil.BrowserUserAgent, got luaprovider %q, lib %q", provider.UserAgent, sdk.UserAgent)
	}
	if provider.HTTPClient.Timeout != netutil.DefaultTimeout || sdk.HTTPClient.Timeout != netutil.DefaultTimeout {
		t.Errorf("wanted netutil.DefaultTimeout, got luaprovider %s, lib %s", provider.HTTPClient.Timeout, sdk.HTTPClient.Timeout)
	}
}
