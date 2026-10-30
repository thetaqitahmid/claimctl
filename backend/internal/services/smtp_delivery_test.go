package services

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSMTPDeliveryStalledServerDeadline(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()
	accepted := make(chan net.Conn, 1)
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			accepted <- conn
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	err = sendSMTP(ctx, listener.Addr().String(), "localhost", nil, "from@example.com", []string{"to@example.com"}, []byte("hello"))
	require.Error(t, err)
	require.Less(t, time.Since(start), time.Second)
	conn := <-accepted
	defer conn.Close()
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(time.Second)))
	_, err = conn.Read(make([]byte, 1))
	require.Error(t, err, "SMTP connection must close on deadline")
}

func TestSMTPDeliveryStalledDial(t *testing.T) {
	original := dialSMTP
	defer func() { dialSMTP = original }()
	dialSMTP = func(ctx context.Context, network, addr string) (net.Conn, error) { <-ctx.Done(); return nil, ctx.Err() }
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	start := time.Now()
	err := sendSMTP(ctx, "smtp.invalid:25", "smtp.invalid", nil, "from@example.com", nil, nil)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Less(t, time.Since(start), time.Second)
}

func TestSMTPDeliveryCancelledDial(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := sendSMTP(ctx, "127.0.0.1:1", "localhost", nil, "from@example.com", nil, nil)
	require.ErrorIs(t, err, context.Canceled)
}

func TestSMTPDeliveryCancelledGreeting(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	accepted := make(chan net.Conn, 1)
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			accepted <- conn
			cancel()
		}
	}()
	start := time.Now()
	require.Error(t, sendSMTP(ctx, listener.Addr().String(), "localhost", nil, "from@example.com", nil, nil))
	require.Less(t, time.Since(start), time.Second)
	conn := <-accepted
	defer conn.Close()
}
