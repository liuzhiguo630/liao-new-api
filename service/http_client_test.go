package service

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ForceHTTP1 渠道依赖 ALPN 真正协商到 http/1.1：只把 ForceAttemptHTTP2 设为 false
// 是无效的，net/http 会在 TLSNextProto 为 nil 时自动装配 HTTP/2，渠道又会退回到
// 单连接多流，GOAWAY 仍然会成批截断在途请求。
func TestNewRelayTransportForceHTTP1DisablesHTTP2(t *testing.T) {
	h2 := newRelayTransport(false)
	assert.True(t, h2.ForceAttemptHTTP2)
	assert.Nil(t, h2.TLSNextProto, "default transport must keep HTTP/2 auto-configuration")

	h1 := newRelayTransport(true)
	assert.False(t, h1.ForceAttemptHTTP2)
	require.NotNil(t, h1.TLSNextProto, "TLSNextProto must be non-nil to disable HTTP/2")
	assert.Empty(t, h1.TLSNextProto)
	require.NotNil(t, h1.TLSClientConfig)
	assert.Equal(t, []string{"http/1.1"}, h1.TLSClientConfig.NextProtos)
}

// common.InsecureTLSConfig 是进程级共享的，forceHTTP1 写 NextProtos 时必须先 Clone，
// 否则会把 http/1.1 强加到所有使用该配置的出站连接上。
func TestNewRelayTransportForceHTTP1DoesNotMutateSharedTLSConfig(t *testing.T) {
	original := common.TLSInsecureSkipVerify
	common.TLSInsecureSkipVerify = true
	t.Cleanup(func() { common.TLSInsecureSkipVerify = original })

	h1 := newRelayTransport(true)

	assert.Equal(t, []string{"http/1.1"}, h1.TLSClientConfig.NextProtos)
	assert.True(t, h1.TLSClientConfig.InsecureSkipVerify, "insecure skip verify must survive the clone")
	assert.NotSame(t, common.InsecureTLSConfig, h1.TLSClientConfig)
	assert.Nil(t, common.InsecureTLSConfig.NextProtos, "shared TLS config must stay untouched")
}

// 端到端确认 ALPN 结果：默认 client 与上游谈 HTTP/2，ForceHTTP1 client 谈 HTTP/1.1。
// 这是渠道级 force_http1 的核心契约——上游发 GOAWAY 时，h1.1 让影响面收敛到单个请求。
func TestRelayClientsNegotiatedProtocol(t *testing.T) {
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, r.Proto)
	}))
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.Close)

	originalInsecure := common.TLSInsecureSkipVerify
	common.TLSInsecureSkipVerify = true
	t.Cleanup(func() {
		common.TLSInsecureSkipVerify = originalInsecure
		InitHttpClient()
	})
	InitHttpClient()

	get := func(client *http.Client) string {
		resp, err := client.Get(server.URL)
		require.NoError(t, err)
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		return string(body)
	}

	assert.Equal(t, "HTTP/2.0", get(GetHttpClient()))
	assert.Equal(t, "HTTP/1.1", get(GetHTTP1Client()))
}
