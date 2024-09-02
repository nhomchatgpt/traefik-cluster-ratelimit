package redis

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

const MAX_ACTIVE = 5
const DIAL_TIMEOUT = 3 * time.Second

type RedisClient struct {
	mu          sync.Mutex
	conns       chan net.Conn
	addr        string
	maxActive   int
	dialTimeout time.Duration
	auth        string
	db          int
}

// NewConnPool initializes a new connection pool
func NewRedisClient(addr string, db uint, authpassword string) (*RedisClient, error) {
	maxActive := MAX_ACTIVE
	dialTimeout := DIAL_TIMEOUT

	if maxActive <= 0 {
		return nil, errors.New("maxActive must be greater than 0")
	}

	r := &RedisClient{
		conns:       make(chan net.Conn, maxActive),
		addr:        addr,
		maxActive:   maxActive,
		dialTimeout: dialTimeout,
		auth:        authpassword,
	}

	// Prepopulate the pool with connections
	for i := 0; i < maxActive; i++ {
		conn, err := r.newConn()
		if err != nil {
			return nil, err
		}
		r.conns <- conn
	}

	return r, nil
}

func (r *RedisClient) newConn() (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", r.addr, r.dialTimeout)
	if err != nil {
		return nil, err
	}
	if r.auth != "" {
		resp, err := sendCommand(conn, "AUTH", r.auth)
		if err != nil {
			return nil, err
		}
		if resp.Success != RESP_SUCCESS || resp.Result != "OK" {
			return nil, fmt.Errorf("not able to authenticate (%s)", resp.Result)
		}
	}
	resp, err := sendCommand(conn, "SELECT", fmt.Sprintf("%d", r.db))
	if err != nil {
		return nil, err
	}
	if resp.Success != RESP_SUCCESS || resp.Result != "OK" {
		return nil, fmt.Errorf("not able to select db %d (%s)", r.db, resp.Result)
	}
	return conn, nil
}

// Get retrieves a connection from the pool
func (r *RedisClient) Get() (net.Conn, error) {
	select {
	case conn := <-r.conns:
		return conn, nil
	default:
		return r.newConn()
	}
}

// Put returns a connection back to the pool
func (r *RedisClient) Put(conn net.Conn) error {
	if conn == nil {
		return errors.New("nil connection cannot be added to the pool")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// If the pool is full, just close the connection
	if len(r.conns) >= r.maxActive {
		conn.Close()
		return nil
	}

	r.conns <- conn
	return nil
}

// Close closes all the connections in the pool
func (r *RedisClient) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()

	close(r.conns)
	for conn := range r.conns {
		conn.Close()
	}
}

const (
	RESP_SUCCESS = iota
	RESP_FAIL
	RESP_SUCCESS_WITH_RESULT
	RESP_UNKNOWN
)

type RedisResult struct {
	Success int
	Result  string
}

// sendCommand sends a command to Redis and returns the response.
func sendCommand(conn net.Conn, args ...string) (*RedisResult, error) {
	// Construct the RESP command
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("*%d\r\n", len(args))) // Array prefix
	for _, arg := range args {
		sb.WriteString(fmt.Sprintf("$%d\r\n%s\r\n", len(arg), arg)) // Bulk string for each argument
	}
	command := sb.String()

	// Send the command
	_, err := conn.Write([]byte(command))
	if err != nil {
		return nil, fmt.Errorf("error sending command: %w", err)
	}

	// Read the response
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	// strip \r\n
	if response[len(response)-1] == '\n' && response[len(response)-2] == '\r' {
		response = response[:len(response)-2]
	}

	if response[0] == '+' {
		return &RedisResult{
			Success: RESP_SUCCESS,
			Result:  response[1:],
		}, nil
	}
	if response[0] == '-' {
		return &RedisResult{
			Success: RESP_FAIL,
			Result:  response[1:],
		}, nil
	}

	// if the first line start with a '$' we are reading a RESP response
	// let's find the second line, with the actual content
	if response[0] == '$' {
		response, err = reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("error reading response: %w", err)
		}
		// strip \r\n
		if response[len(response)-1] == '\n' && response[len(response)-2] == '\r' {
			response = response[:len(response)-2]
		}
		return &RedisResult{
			Success: RESP_SUCCESS_WITH_RESULT,
			Result:  response,
		}, nil
	}

	return &RedisResult{
		Success: RESP_UNKNOWN,
		Result:  response,
	}, nil
}

func (r *RedisClient) Ping() error {
	conn, err := r.Get()
	if err != nil {
		return err
	}
	defer r.Put(conn)

	res, err := sendCommand(conn, "PING")
	if err != nil {
		return err
	}

	if res.Success != RESP_SUCCESS && res.Result != "PONG" {
		return fmt.Errorf("PING result error: %s", res.Result)
	}
	return nil
}
