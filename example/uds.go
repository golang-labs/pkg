package main

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/golang-labs/pkg/uds"
	"k8s.io/klog"
)

const SOCK = "./etc/pkg/test.sock"

func main() {

	ch := make(chan interface{})
	go server(ch)

	select {
	case <-ch:
		go func() {
			defer klog.Info("press CTRL-C to exit.")
			client()
		}()
	}

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)
	<-ctx.Done() // block main process to wait goroutine print response message
}

func server(ch chan interface{}) {
	l, err := uds.NewListener(SOCK)
	if err != nil {
		klog.Error(err)
	}
	defer l.Close()

	ch <- struct{}{}

	mux := http.NewServeMux()
	mux.HandleFunc("/say-hello", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
	})
	err = http.Serve(l, mux)
	if err != nil {
		klog.Errorf("http.Serve error: %v", err)
	}
}

func client() {
	req, err := http.NewRequest(http.MethodGet, "http://unix/say-hello", nil)
	if err != nil {
		klog.Errorf("http.NewRequest error: %v", err)
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				dialer := &net.Dialer{}
				return dialer.DialContext(ctx, "unix", SOCK)
			},
			ForceAttemptHTTP2:     false,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		klog.Errorf("httpClient.Do error: %v", err)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		klog.Errorf("io.ReadAll error: %v", err)
	}
	klog.Info(string(b))
}
