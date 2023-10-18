package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "nats-health-check",
		Usage: "A command-line script for performing remote node health checks using NATS.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "host",
				Usage:   "Specify the NATS server host.",
				EnvVars: []string{"NATS_HOST"},
				Value:   "localhost",
			},
			&cli.IntFlag{
				Name:    "port",
				Aliases: []string{"p"},
				Usage:   "Specify the NATS server port.",
				Value:   4222,
				EnvVars: []string{"NATS_PORT"},
			},
			&cli.StringFlag{
				Name:    "subject",
				Aliases: []string{"s", "sub", "t", "topic"},
				Usage:   "Specify the NATS subject to send health check messages.",
				EnvVars: []string{"NATS_REQUEST_SUBJECT"},
			},
			&cli.DurationFlag{
				Name:    "timeout",
				Usage:   "Specify the timeout for the health check in seconds.",
				EnvVars: []string{"NATS_REQUEST_TIMEOUT"},
				Value:   5 * time.Second,
			},
			&cli.StringFlag{
				Name:    "user-agent",
				Usage:   "Specify a custom user agent string for identifying the client.",
				EnvVars: []string{"NATS_USER_AGENT"},
				Value:   "NATS Health Check",
			},
		},
		Action: run,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func run(cli *cli.Context) error {
	subject := cli.String("subject")
	if subject == "" {
		return errors.New("invalid subject")
	}

	dialer := NewDialer(cli.String("user-agent"))
	opt := nats.SetCustomDialer(dialer)

	url := "nats://" + cli.String("host") + ":" + strconv.Itoa(cli.Int("port"))
	nc, err := nats.Connect(url, opt)
	if err != nil {
		return err
	}
	defer nc.Drain()

	req, err := json.Marshal(&dialer)
	if err != nil {
		return err
	}

	resp, err := nc.Request(subject, req, cli.Duration("timeout"))
	if err != nil {
		return err
	}

	msg := string(resp.Data)
	if msg != "ok" {
		return errors.New(msg)
	}

	fmt.Print(msg)
	return nil
}

var RequestInfo struct {
	ClientIP  string `json:"client_ip"`
	UserAgent string `json:"user_agent"`
}

func NewDialer(agent string) nats.CustomDialer {
	d := new(dialer)
	d.UserAgent = agent
	return d
}

type dialer struct {
	ClientIP  string `json:"client_ip"`
	UserAgent string `json:"user_agent"`
}

func (d *dialer) Dial(network, address string) (net.Conn, error) {
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}

	local := conn.LocalAddr()
	addr, err := net.ResolveTCPAddr(local.Network(), local.String())
	if err != nil {
		return nil, err
	}

	d.ClientIP = addr.IP.String()

	return conn, nil
}
